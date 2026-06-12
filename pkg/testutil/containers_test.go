package testutil

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"gorm.io/gorm"
)

func TestSetupPostgresContainer_ErrorPaths(t *testing.T) {
	ctx := context.Background()

	t.Run("postgres.Run error", func(t *testing.T) {
		orig := postgresRunContainer
		postgresRunContainer = func(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*postgres.PostgresContainer, error) {
			return nil, errors.New("container error")
		}
		defer func() { postgresRunContainer = orig }()

		pg, err := SetupPostgresContainer(ctx)
		assert.Nil(t, pg)
		assert.ErrorContains(t, err, "container error")
	})

	t.Run("postgresConnectionString error", func(t *testing.T) {
		origRun := postgresRunContainer
		postgresRunContainer = func(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*postgres.PostgresContainer, error) {
			return &postgres.PostgresContainer{}, nil
		}
		defer func() { postgresRunContainer = origRun }()

		origConnStr := postgresConnectionString
		postgresConnectionString = func(ctx context.Context, c *postgres.PostgresContainer) (string, error) {
			return "", errors.New("conn string error")
		}
		defer func() { postgresConnectionString = origConnStr }()

		pg, err := SetupPostgresContainer(ctx)
		assert.Nil(t, pg)
		assert.ErrorContains(t, err, "conn string error")
	})

	t.Run("gormOpen error", func(t *testing.T) {
		origRun := postgresRunContainer
		postgresRunContainer = func(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*postgres.PostgresContainer, error) {
			return &postgres.PostgresContainer{}, nil
		}
		defer func() { postgresRunContainer = origRun }()

		origConnStr := postgresConnectionString
		postgresConnectionString = func(ctx context.Context, c *postgres.PostgresContainer) (string, error) {
			return "postgres://localhost:5432/test?sslmode=disable", nil
		}
		defer func() { postgresConnectionString = origConnStr }()

		origGorm := gormOpen
		gormOpen = func(dialector gorm.Dialector, config *gorm.Config) (*gorm.DB, error) {
			return nil, errors.New("gorm error")
		}
		defer func() { gormOpen = origGorm }()

		pg, err := SetupPostgresContainer(ctx)
		assert.Nil(t, pg)
		assert.ErrorContains(t, err, "gorm error")
	})

	t.Run("autoMigrate error", func(t *testing.T) {
		origRun := postgresRunContainer
		postgresRunContainer = func(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*postgres.PostgresContainer, error) {
			return &postgres.PostgresContainer{}, nil
		}
		defer func() { postgresRunContainer = origRun }()

		origConnStr := postgresConnectionString
		postgresConnectionString = func(ctx context.Context, c *postgres.PostgresContainer) (string, error) {
			return "postgres://localhost:5432/test?sslmode=disable", nil
		}
		defer func() { postgresConnectionString = origConnStr }()

		origGorm := gormOpen
		gormOpen = func(dialector gorm.Dialector, config *gorm.Config) (*gorm.DB, error) {
			return &gorm.DB{}, nil
		}
		defer func() { gormOpen = origGorm }()

		origMigrate := autoMigrate
		autoMigrate = func(ctx context.Context, db *gorm.DB) error {
			return errors.New("migrate error")
		}
		defer func() { autoMigrate = origMigrate }()

		pg, err := SetupPostgresContainer(ctx)
		assert.Nil(t, pg)
		assert.ErrorContains(t, err, "migrate error")
	})
}

func TestSetupRedisContainer_ErrorPaths(t *testing.T) {
	ctx := context.Background()

	t.Run("redis.Run error", func(t *testing.T) {
		orig := redisRunContainer
		redisRunContainer = func(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*redis.RedisContainer, error) {
			return nil, errors.New("container error")
		}
		defer func() { redisRunContainer = orig }()

		rd, err := SetupRedisContainer(ctx)
		assert.Nil(t, rd)
		assert.ErrorContains(t, err, "container error")
	})

	t.Run("redisConnectionString error", func(t *testing.T) {
		origRun := redisRunContainer
		redisRunContainer = func(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*redis.RedisContainer, error) {
			return &redis.RedisContainer{}, nil
		}
		defer func() { redisRunContainer = origRun }()

		origConnStr := redisConnectionString
		redisConnectionString = func(ctx context.Context, c *redis.RedisContainer) (string, error) {
			return "", errors.New("conn string error")
		}
		defer func() { redisConnectionString = origConnStr }()

		rd, err := SetupRedisContainer(ctx)
		assert.Nil(t, rd)
		assert.ErrorContains(t, err, "conn string error")
	})
}
