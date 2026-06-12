package testutil

import (
	"context"
	"fmt"
	"time"

	"backend-go/internal/core/models"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type PostgresContainer struct {
	*postgres.PostgresContainer
	DB *gorm.DB
}

type RedisContainer struct {
	*redis.RedisContainer
	URI string
}

var (
	// Internal hooks for testing
	postgresConnectionString = func(ctx context.Context, c *postgres.PostgresContainer) (string, error) {
		return c.ConnectionString(ctx, "sslmode=disable")
	}
	redisConnectionString = func(ctx context.Context, c *redis.RedisContainer) (string, error) {
		return c.ConnectionString(ctx)
	}
	gormOpen = func(dialector gorm.Dialector, config *gorm.Config) (*gorm.DB, error) {
		return gorm.Open(dialector, config)
	}
	autoMigrate = func(ctx context.Context, db *gorm.DB) error {
		db.Exec("CREATE SCHEMA IF NOT EXISTS audit")
		return db.WithContext(ctx).AutoMigrate(
			&models.AuditLog{},
			&models.Role{},
			&models.Feature{},
			&models.RoleFeature{},
			&models.Auth{},
			&models.User{},
			&models.Product{},
		)
	}
)

func SetupPostgresContainer(ctx context.Context) (*PostgresContainer, error) {

	dbName := "testdb"
	dbUser := "postgres"
	dbPassword := "postgres"

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		return nil, err
	}

	connStr, err := postgresConnectionString(ctx, pgContainer)
	if err != nil {
		return nil, err
	}

	db, err := gormOpen(gormpostgres.Open(connStr), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		return nil, err
	}

	// Always run AutoMigrate in tests to ensure all tables exist, 
	// especially since migrations might be empty or incomplete during development.
	err = autoMigrate(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("falha no automigrate de teste: %v", err)
	}

	return &PostgresContainer{
		PostgresContainer: pgContainer,
		DB:                db,
	}, nil
}

func SetupRedisContainer(ctx context.Context) (*RedisContainer, error) {
	redisContainer, err := redis.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections"),
		),
	)
	if err != nil {
		return nil, err
	}

	uri, err := redisConnectionString(ctx, redisContainer)
	if err != nil {
		return nil, err
	}

	return &RedisContainer{
		RedisContainer: redisContainer,
		URI:            uri,
	}, nil
}
