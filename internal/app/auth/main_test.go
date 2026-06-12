package auth

import (
	"context"
	"log"
	"os"
	"testing"

	"backend-go/pkg/cache"
	"backend-go/pkg/config"
	"backend-go/pkg/database"
	"backend-go/pkg/testutil"
	"github.com/redis/go-redis/v9"
)

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	config.LoadConfig()

	ctx := context.Background()

	// Setup Postgres
	pg, err := testutil.SetupPostgresContainer(ctx)
	if err != nil {
		log.Fatalf("Falha ao subir Postgres Container: %v", err)
	}
	defer pg.Terminate(ctx)
	database.DB = pg.DB
	connStr, _ := pg.ConnectionString(ctx, "sslmode=disable")
	config.AppConfig.DBUrl = connStr

	// Setup Redis
	rd, err := testutil.SetupRedisContainer(ctx)
	if err != nil {
		log.Fatalf("Falha ao subir Redis Container: %v", err)
	}
	defer rd.Terminate(ctx)

	opts, _ := redis.ParseURL(rd.URI)
	cache.RedisClient = redis.NewClient(opts)

	code := m.Run()
	os.Exit(code)
}
