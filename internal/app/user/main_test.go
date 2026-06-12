package user

import (
	"context"
	"fmt"
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

	// Clean and Seed mandatory roles
	tables := []string{"audit\".\"audit_log", "auth", "user", "product", "role_feature", "feature", "role"}
	for _, table := range tables {
		database.DB.Exec(fmt.Sprintf("TRUNCATE TABLE \"%s\" CASCADE", table))
	}
	
	database.RunSeed(database.DB)

	code := m.Run()
	os.Exit(code)
}
