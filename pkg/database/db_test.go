package database

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"backend-go/pkg/config"
	"backend-go/pkg/retry"
	"gorm.io/gorm"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type mockMigrator struct {
	upFunc func() error
}

func (m *mockMigrator) Up() error {
	return m.upFunc()
}

func TestConnectDB(t *testing.T) {
	// Backup original values
	origEnv := config.AppConfig.Environment
	origDBUrl := config.AppConfig.DBUrl
	origFatalf := logFatalf
	origGormOpen := gormOpen
	origAutoMigrate := dbAutoMigrate
	origRunMigrations := runMigrations
	origDB := DB
	origRetry := dbRetryConfig
	dbRetryConfig = retry.Config{MaxAttempts: 2, Delay: time.Millisecond, Factor: 1.0}

	defer func() {
		config.AppConfig.Environment = origEnv
		config.AppConfig.DBUrl = origDBUrl
		logFatalf = origFatalf
		gormOpen = origGormOpen
		dbAutoMigrate = origAutoMigrate
		runMigrations = origRunMigrations
		DB = origDB
		dbRetryConfig = origRetry
	}()

	t.Run("Success in test mode", func(t *testing.T) {
		config.AppConfig.Environment = "test"
		gormOpen = gorm.Open
		dbAutoMigrate = func(db *gorm.DB, dst ...interface{}) error { return nil }
		logFatalf = origFatalf
		
		ConnectDB()
		assert.NotNil(t, DB)
	})

	t.Run("Success in production mode", func(t *testing.T) {
		config.AppConfig.Environment = "production"
		// Mock gormOpen to use sqlite even in "production" mode for the test
		gormOpen = func(dialector gorm.Dialector, opts ...gorm.Option) (*gorm.DB, error) {
			return testDB, nil
		}
		dbAutoMigrate = func(db *gorm.DB, dst ...interface{}) error { return nil }
		runMigrations = func() {}
		logFatalf = origFatalf

		ConnectDB()
		assert.NotNil(t, DB)
		// Verification that we passed through the production check could be logger level check, 
		// but since we mocked gormOpen we can't easily check the internal gormConfig.
		// However, the branch is covered.
	})

	t.Run("Failure on gorm.Open", func(t *testing.T) {
		config.AppConfig.Environment = "test"
		gormOpen = func(dialector gorm.Dialector, opts ...gorm.Option) (*gorm.DB, error) {
			return nil, errors.New("connection error")
		}
		
		logFatalf = func(format string, v ...interface{}) {
			panic("fatal: connection")
		}

		assert.PanicsWithValue(t, "fatal: connection", func() {
			ConnectDB()
		})
	})

	t.Run("Failure on pgx.ParseConfig", func(t *testing.T) {
		config.AppConfig.Environment = "test"
		origDBUrl := config.AppConfig.DBUrl
		config.AppConfig.DBUrl = "invalid://%"
		defer func() { config.AppConfig.DBUrl = origDBUrl }()

		logFatalf = func(format string, v ...interface{}) {
			panic("fatal: parse")
		}

		assert.PanicsWithValue(t, "fatal: parse", func() {
			ConnectDB()
		})
	})

	t.Run("Failure on AutoMigrate", func(t *testing.T) {
		config.AppConfig.Environment = "test"
		gormOpen = func(dialector gorm.Dialector, opts ...gorm.Option) (*gorm.DB, error) {
			return testDB, nil
		}
		dbAutoMigrate = func(db *gorm.DB, dst ...interface{}) error {
			return errors.New("migration error")
		}
		
		logFatalf = func(format string, v ...interface{}) {
			panic("fatal: migration")
		}

		assert.PanicsWithValue(t, "fatal: migration", func() {
			ConnectDB()
		})
	})
}

func TestDefaultRunMigrations(t *testing.T) {
	origMigrateNew := migrateNew
	origFatalf := logFatalf
	defer func() {
		migrateNew = origMigrateNew
		logFatalf = origFatalf
	}()

	t.Run("Success or No Change", func(t *testing.T) {
		migrateNew = func(sourceURL, databaseURL string) (migrator, error) {
			return &mockMigrator{
				upFunc: func() error {
					return nil // or migrate.ErrNoChange
				},
			}, nil
		}
		
		assert.NotPanics(t, func() {
			defaultRunMigrations()
		})

		migrateNew = func(sourceURL, databaseURL string) (migrator, error) {
			return &mockMigrator{
				upFunc: func() error {
					return migrate.ErrNoChange
				},
			}, nil
		}
		
		assert.NotPanics(t, func() {
			defaultRunMigrations()
		})
	})

	t.Run("Failure on migrate.New", func(t *testing.T) {
		migrateNew = func(sourceURL, databaseURL string) (migrator, error) {
			return nil, errors.New("new error")
		}
		logFatalf = func(format string, v ...interface{}) {
			panic("fatal: new")
		}
		assert.PanicsWithValue(t, "fatal: new", func() {
			defaultRunMigrations()
		})
	})

	t.Run("Failure on m.Up", func(t *testing.T) {
		migrateNew = func(sourceURL, databaseURL string) (migrator, error) {
			return &mockMigrator{
				upFunc: func() error {
					return errors.New("up error")
				},
			}, nil
		}
		
		logFatalf = func(format string, v ...interface{}) {
			panic("fatal: up")
		}
		
		assert.PanicsWithValue(t, "fatal: up", func() {
			defaultRunMigrations()
		})
	})

	t.Run("Real Integration test branch coverage", func(t *testing.T) {
		// Just to be sure we also run the real one if possible
		tmpDir := t.TempDir()
		dummyFile := filepath.Join(tmpDir, "000001_init.up.sql")
		os.WriteFile(dummyFile, []byte("SELECT 1;"), 0644)

		migrateNew = func(sourceURL, databaseURL string) (migrator, error) {
			return migrate.New("file://"+tmpDir, config.AppConfig.DBUrl)
		}
		
		assert.NotPanics(t, func() {
			defaultRunMigrations()
		})
	})
}
