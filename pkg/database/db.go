package database

import (
	"log"
	"os"
	"strconv"
	"time"

	"backend-go/pkg/logger"
	"backend-go/pkg/retry"

	"backend-go/internal/core/models"
	"backend-go/pkg/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

var DB *gorm.DB

var (
	logFatalf         = logger.Fatalf
	gormOpen          = gorm.Open
	dbRetryConfig     = retry.DefaultConfig
	dbAutoMigrate     = func(db *gorm.DB, dst ...interface{}) error { return db.AutoMigrate(dst...) }
	runMigrations     = defaultRunMigrations
	migrateNew        = func(sourceURL, databaseURL string) (migrator, error) {
		return migrate.New(sourceURL, databaseURL)
	}
)

type migrator interface {
	Up() error
}

func ConnectDB() {
	var err error

	gormConfig := &gorm.Config{
		Logger: gormlogger.New(
			log.New(os.Stderr, "\r\n", log.LstdFlags),
			gormlogger.Config{
				SlowThreshold: 100 * time.Millisecond,
				LogLevel:      gormlogger.Warn,
				Colorful:      true,
			},
		),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		PrepareStmt: true,
		NowFunc:     func() time.Time { return time.Now().UTC() },
	}

	if config.AppConfig.Environment == "production" {
		gormConfig.Logger = gormlogger.Default.LogMode(gormlogger.Error)
	}

	connConfig, err := pgx.ParseConfig(config.AppConfig.DBUrl)
	if err != nil {
		logFatalf("Falha ao parsear DSN: %v", err)
	}
	if config.AppConfig.DBStatementTimeout > 0 {
		connConfig.RuntimeParams["statement_timeout"] = strconv.Itoa(config.AppConfig.DBStatementTimeout)
	}
	if config.AppConfig.DBIdleInTxTimeout > 0 {
		connConfig.RuntimeParams["idle_in_transaction_session_timeout"] = strconv.Itoa(config.AppConfig.DBIdleInTxTimeout)
	}

	sqlDB := stdlib.OpenDB(*connConfig)
	sqlDB.SetMaxOpenConns(config.AppConfig.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(config.AppConfig.DBMaxIdleConns)
	if lifetime, err := time.ParseDuration(config.AppConfig.DBConnMaxLifetime); err == nil {
		sqlDB.SetConnMaxLifetime(lifetime)
	}
	if idleTime, err := time.ParseDuration(config.AppConfig.DBConnMaxIdleTime); err == nil {
		sqlDB.SetConnMaxIdleTime(idleTime)
	}

	err = retry.Do(func() error {
		var innerErr error
		DB, innerErr = gormOpen(postgres.New(postgres.Config{
			Conn: sqlDB,
		}), gormConfig)
		return innerErr
	}, dbRetryConfig, "conexão com banco de dados")

	if err != nil {
		logFatalf("Falha ao conectar no banco de dados: %v", err)
	}

	if config.AppConfig.Environment == "production" {
		runMigrations()
	} else {
		logger.Info("Rodando AutoMigrate...")
		DB.Exec("CREATE SCHEMA IF NOT EXISTS audit")
		err = dbAutoMigrate(
			DB,
			&models.AuditLog{},
			&models.ErrorLog{},
			&models.Role{},
			&models.Feature{},
			&models.RoleFeature{},
			&models.Auth{},
			&models.User{},
			&models.Product{},
		)
		if err != nil {
			logFatalf("Erro no AutoMigrate: %v", err)
		}
	}

	RunSeed(DB)

	logger.Info("Conexão com PostgreSQL estabelecida com sucesso.")
}

func defaultRunMigrations() {
	m, err := migrateNew(
		"file://database/migrations",
		config.AppConfig.DBUrl,
	)
	if err != nil {
		logFatalf("Falha ao preparar migrações: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		logFatalf("Falha ao executar migrações: %v", err)
	}

	logger.Info("Migrações aplicadas com sucesso.")
}
