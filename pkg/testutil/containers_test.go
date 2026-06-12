package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	testcontainersredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"gorm.io/gorm"
)

func TestSetupPostgresContainer(t *testing.T) {
	t.Run("Error - Already Cancelled Context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		container, err := SetupPostgresContainer(ctx)
		assert.Error(t, err)
		assert.Nil(t, container)
	})

	t.Run("Error - Timeout during setup", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}
		// A very short timeout that is already expired
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond)

		container, err := SetupPostgresContainer(ctx)
		assert.Error(t, err)
		assert.Nil(t, container)
	})

	t.Run("Error - ConnectionString failure", func(t *testing.T) {
		// Mock ConnectionString to return an error
		old := postgresConnectionString
		postgresConnectionString = func(ctx context.Context, c *postgres.PostgresContainer) (string, error) {
			return "", fmt.Errorf("forced connection string error")
		}
		defer func() { postgresConnectionString = old }()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		container, err := SetupPostgresContainer(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced connection string error")
		assert.Nil(t, container)
	})

	t.Run("Error - Gorm Open failure", func(t *testing.T) {
		// Mock gormOpen to return an error
		old := gormOpen
		gormOpen = func(dialector gorm.Dialector, config *gorm.Config) (*gorm.DB, error) {
			return nil, fmt.Errorf("forced gorm open error")
		}
		defer func() { gormOpen = old }()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		container, err := SetupPostgresContainer(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced gorm open error")
		assert.Nil(t, container)
	})

	t.Run("Error - AutoMigrate failure", func(t *testing.T) {
		// Mock autoMigrate to return an error
		old := autoMigrate
		autoMigrate = func(ctx context.Context, db *gorm.DB) error {
			return fmt.Errorf("forced automigrate error")
		}
		defer func() { autoMigrate = old }()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		container, err := SetupPostgresContainer(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "falha no automigrate de teste: forced automigrate error")
		assert.Nil(t, container)
	})

	t.Run("Success", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		container, err := SetupPostgresContainer(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, container)
		assert.NotNil(t, container.DB)

		// Verify DB is working
		var result int
		err = container.DB.Raw("SELECT 1").Scan(&result).Error
		assert.NoError(t, err)
		assert.Equal(t, 1, result)

		err = container.Terminate(ctx)
		assert.NoError(t, err)
	})
}

func TestSetupRedisContainer(t *testing.T) {
	t.Run("Error - Already Cancelled Context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		container, err := SetupRedisContainer(ctx)
		assert.Error(t, err)
		assert.Nil(t, container)
	})

	t.Run("Error - Timeout during setup", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}
		// A very short timeout that is already expired
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond)

		container, err := SetupRedisContainer(ctx)
		assert.Error(t, err)
		assert.Nil(t, container)
	})

	t.Run("Error - ConnectionString failure", func(t *testing.T) {
		// Mock ConnectionString to return an error
		old := redisConnectionString
		redisConnectionString = func(ctx context.Context, c *testcontainersredis.RedisContainer) (string, error) {
			return "", fmt.Errorf("forced redis connection string error")
		}
		defer func() { redisConnectionString = old }()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		container, err := SetupRedisContainer(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced redis connection string error")
		assert.Nil(t, container)
	})

	t.Run("Success", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		container, err := SetupRedisContainer(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, container)
		assert.NotEmpty(t, container.URI)

		// Verify Redis is working
		opts, err := redis.ParseURL(container.URI)
		assert.NoError(t, err)
		client := redis.NewClient(opts)
		defer client.Close()

		err = client.Ping(ctx).Err()
		assert.NoError(t, err)

		err = container.Terminate(ctx)
		assert.NoError(t, err)
	})
}
