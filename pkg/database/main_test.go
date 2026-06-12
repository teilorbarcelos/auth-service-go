package database

import (
	"context"
	"log"
	"os"
	"testing"

	"backend-go/pkg/config"
	"backend-go/pkg/testutil"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	config.LoadConfig()

	ctx := context.Background()
	pg, err := testutil.SetupPostgresContainer(ctx)
	if err != nil {
		log.Fatalf("Falha ao subir Postgres Container: %v", err)
	}
	defer pg.Terminate(ctx)

	testDB = pg.DB
	DB = pg.DB

	connStr, _ := pg.ConnectionString(ctx, "sslmode=disable")
	config.AppConfig.DBUrl = connStr

	code := m.Run()
	os.Exit(code)
}
