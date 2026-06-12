package cache

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"backend-go/pkg/config"
)

func TestConnectRedis(t *testing.T) {
	// Backup original values
	origEnv := config.AppConfig.Environment
	origURL := config.AppConfig.RedisUrl
	origFatalf := logFatalf
	origMiniredisRun := miniredisRun
	origClient := RedisClient
	
	defer func() {
		config.AppConfig.Environment = origEnv
		config.AppConfig.RedisUrl = origURL
		logFatalf = origFatalf
		miniredisRun = origMiniredisRun
		RedisClient = origClient
	}()

	t.Run("Success in test environment", func(t *testing.T) {
		config.AppConfig.Environment = "test"
		miniredisRun = miniredis.Run // Ensure real miniredis
		logFatalf = origFatalf
		
		ConnectRedis()
		assert.NotNil(t, RedisClient)
		err := RedisClient.Ping(context.Background()).Err()
		assert.NoError(t, err)
	})

	t.Run("Failure starting miniredis", func(t *testing.T) {
		config.AppConfig.Environment = "test"
		miniredisRun = func() (*miniredis.Miniredis, error) {
			return nil, errors.New("miniredis error")
		}
		
		logFatalf = func(format string, v ...interface{}) {
			panic("fatal: miniredis")
		}
		
		assert.PanicsWithValue(t, "fatal: miniredis", func() {
			ConnectRedis()
		})
	})

	t.Run("Success in production environment", func(t *testing.T) {
		config.AppConfig.Environment = "production"
		// Use miniredis but simulate it being a real redis URL
		mr, _ := miniredis.Run()
		defer mr.Close()
		config.AppConfig.RedisUrl = fmt.Sprintf("redis://%s", mr.Addr())
		logFatalf = origFatalf
		
		ConnectRedis()
		assert.NotNil(t, RedisClient)
		assert.Equal(t, mr.Addr(), RedisClient.Options().Addr)
	})

	t.Run("Failure parsing Redis URL", func(t *testing.T) {
		config.AppConfig.Environment = "production"
		config.AppConfig.RedisUrl = "!!invalid-url!!" // redis.ParseURL fails on this
		
		logFatalf = func(format string, v ...interface{}) {
			panic("fatal: parse")
		}
		
		assert.PanicsWithValue(t, "fatal: parse", func() {
			ConnectRedis()
		})
	})

	t.Run("Failure pinging Redis", func(t *testing.T) {
		config.AppConfig.Environment = "production"
		// Valid URL but nothing listening on that port
		// Using a likely unused port on localhost
		config.AppConfig.RedisUrl = "redis://localhost:9999"
		
		logFatalf = func(format string, v ...interface{}) {
			panic("fatal: ping")
		}
		
		assert.PanicsWithValue(t, "fatal: ping", func() {
			ConnectRedis()
		})
	})
}
