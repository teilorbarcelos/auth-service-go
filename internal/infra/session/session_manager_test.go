package session

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/redis/go-redis/v9"
	"backend-go/pkg/cache"
	"backend-go/pkg/config"
)

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	config.LoadConfig()
	cache.ConnectRedis()

	code := m.Run()
	os.Exit(code)
}

func TestSessionManager_InvalidateUserSessions(t *testing.T) {
	sm := NewSessionManager()
	ctx := context.Background()
	userId := "user123"
	roleId := "admin"
	verKey := fmt.Sprintf(sessionVersionKeyFormat, userId)

	t.Run("Invalidate bumps version key", func(t *testing.T) {
		// Clean up
		cache.RedisClient.Del(ctx, verKey)

		err := sm.InvalidateUserSessions(userId, roleId)
		assert.NoError(t, err)

		val, err := cache.RedisClient.Get(ctx, verKey).Int()
		assert.NoError(t, err)
		assert.Equal(t, 1, val)
	})

	t.Run("Invalidate bumps version incrementally", func(t *testing.T) {
		cache.RedisClient.Set(ctx, verKey, 5, 0)
		defer cache.RedisClient.Del(ctx, verKey)

		err := sm.InvalidateUserSessions(userId, roleId)
		assert.NoError(t, err)

		val, err := cache.RedisClient.Get(ctx, verKey).Int()
		assert.NoError(t, err)
		assert.Equal(t, 6, val)
	})
}

func TestSessionManager_InvalidateRoleSessions(t *testing.T) {
	sm := NewSessionManager()
	ctx := context.Background()
	roleId := "manager"

	key1 := fmt.Sprintf("session:role:%s:user:u1:1", roleId)
	key2 := fmt.Sprintf("session:role:%s:user:u2:2", roleId)
	cache.RedisClient.Set(ctx, key1, "data", 0)
	cache.RedisClient.Set(ctx, key2, "data", 0)

	t.Run("Invalidate existing role sessions", func(t *testing.T) {
		err := sm.InvalidateRoleSessions(roleId)
		assert.NoError(t, err)

		val1 := cache.RedisClient.Exists(ctx, key1).Val()
		val2 := cache.RedisClient.Exists(ctx, key2).Val()
		assert.Equal(t, int64(0), val1)
		assert.Equal(t, int64(0), val2)
	})

	t.Run("Invalidate non-existing role sessions", func(t *testing.T) {
		err := sm.InvalidateRoleSessions("nonexistent_role")
		assert.NoError(t, err)
	})
}

func TestSessionManager_DeleteByPattern_Error(t *testing.T) {
	sm := NewSessionManager()
	ctx := context.Background()

	t.Run("Redis Scan error", func(t *testing.T) {
		originalClient := cache.RedisClient
		cache.RedisClient.Close()

		err := sm.InvalidateRoleSessions("any")
		assert.Error(t, err)

		cache.RedisClient = originalClient
		cache.ConnectRedis()
	})

	t.Run("Redis Del error on InvalidateUserSessions", func(t *testing.T) {
		userId := "del-err-user"
		refreshKey := fmt.Sprintf("session:role:admin:user:%s:refresh:hash123", userId)
		cache.RedisClient.Set(ctx, refreshKey, "1", 0)
		defer cache.RedisClient.Del(ctx, refreshKey)

		hook := &delErrorHook{enabled: true}
		cache.RedisClient.AddHook(hook)
		defer func() { hook.enabled = false }()

		err := sm.InvalidateUserSessions(userId, "admin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced del error")
	})

	t.Run("InvalidateUserSessions INCR error", func(t *testing.T) {
		originalClient := cache.RedisClient
		cache.RedisClient.Close()

		err := sm.InvalidateUserSessions("any", "")
		assert.Error(t, err)

		cache.RedisClient = originalClient
		cache.ConnectRedis()
	})
}

func TestSessionManager_CreateSession(t *testing.T) {
	sm := NewSessionManager()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		payload := map[string]interface{}{"key": "value"}
		err := sm.CreateSession(ctx, "u1", "r1", "hash", payload, 0)
		assert.NoError(t, err)

		key := "session:role:r1:user:u1:access:hash"
		exists := cache.RedisClient.Exists(ctx, key).Val()
		assert.Equal(t, int64(1), exists)
	})

	t.Run("Marshal Error", func(t *testing.T) {
		payload := map[string]interface{}{"key": func() {}}
		err := sm.CreateSession(ctx, "u2", "r1", "hash", payload, 0)
		assert.Error(t, err)
	})
}

func TestSessionManager_CreateRefreshToken(t *testing.T) {
	sm := NewSessionManager()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		err := sm.CreateRefreshToken(ctx, "u1", "r1", "hash", 0)
		assert.NoError(t, err)

		key := "session:role:r1:user:u1:refresh:hash"
		exists := cache.RedisClient.Exists(ctx, key).Val()
		assert.Equal(t, int64(1), exists)
	})
}

func TestSessionManager_SessionVersion(t *testing.T) {
	sm := NewSessionManager()
	ctx := context.Background()
	userId := "ver-test-user"

	t.Run("GetSessionVersion returns error when key not set", func(t *testing.T) {
		cache.RedisClient.Del(ctx, fmt.Sprintf(sessionVersionKeyFormat, userId))
		_, err := sm.GetSessionVersion(ctx, userId)
		assert.Error(t, err)
	})

	t.Run("Set then Get session version", func(t *testing.T) {
		err := sm.SetSessionVersion(ctx, userId, 42)
		assert.NoError(t, err)

		ver, err := sm.GetSessionVersion(ctx, userId)
		assert.NoError(t, err)
		assert.Equal(t, 42, ver)
	})

	t.Run("InvalidateUserSessions bumps version", func(t *testing.T) {
		cache.RedisClient.Del(ctx, fmt.Sprintf(sessionVersionKeyFormat, userId))
		sm.InvalidateUserSessions(userId, "")
		ver, err := sm.GetSessionVersion(ctx, userId)
		assert.NoError(t, err)
		assert.Equal(t, 1, ver)
	})
}

type delErrorHook struct {
	enabled bool
}

func (h *delErrorHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *delErrorHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if h.enabled && cmd.Name() == "del" {
			return fmt.Errorf("forced del error")
		}
		return next(ctx, cmd)
	}
}

func (h *delErrorHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
