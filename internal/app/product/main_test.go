package product

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/teilorbarcelos/auth-service-go/pkg/config"
	"github.com/teilorbarcelos/auth-service-go/pkg/database"
	"github.com/teilorbarcelos/auth-service-go/pkg/testutil"
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

	code := m.Run()
	os.Exit(code)
}
