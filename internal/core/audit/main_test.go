package audit

import (
	"context"
	"log"
	"os"
	"testing"

	"backend-go/internal/core/models"
	"backend-go/pkg/config"
	"backend-go/pkg/database"
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
	database.DB = pg.DB
	connStr, _ := pg.ConnectionString(ctx, "sslmode=disable")
	config.AppConfig.DBUrl = connStr

	// Register hooks for the test DB
	RegisterAuditHooks(testDB)
	
	// Create tables needed specifically for hooks tests
	testDB.AutoMigrate(&AuditTestModel{}, &models.ErrorLog{})

	code := m.Run()
	os.Exit(code)
}
