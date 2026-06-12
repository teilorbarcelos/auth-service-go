package config

import (
	"backend-go/pkg/logger"

	"github.com/spf13/viper"
)

type Config struct {
	Environment       string `mapstructure:"ENVIRONMENT"`
	Port              string `mapstructure:"PORT"`
	Host              string `mapstructure:"HOST"`
	DBUrl             string `mapstructure:"DATABASE_URL"`
	RedisUrl          string `mapstructure:"REDIS_URL"`
	RabbitMQUrl       string `mapstructure:"RABBITMQ_URL"`
	JWTSecret         string `mapstructure:"JWT_SECRET"`
	JWTIssuer         string `mapstructure:"JWT_ISSUER"`
	TrustedProxies    string `mapstructure:"TRUSTED_PROXIES"`
	JWTAudience       string `mapstructure:"JWT_AUDIENCE"`
	RateLimitMax      int    `mapstructure:"RATE_LIMIT_MAX"`
	RateLimitWindow   string `mapstructure:"RATE_LIMIT_WINDOW"`
	FirstUserEmail    string `mapstructure:"FIRST_USER"`
	FirstUserPassword string `mapstructure:"FIRST_PASSWORD"`
	LogLevel          string `mapstructure:"LOG_LEVEL"`
	PdfServiceUrl     string `mapstructure:"PDF_SERVICE_URL"`
	DBMaxOpenConns       int    `mapstructure:"DB_MAX_OPEN_CONNS"`
	DBMaxIdleConns       int    `mapstructure:"DB_MAX_IDLE_CONNS"`
	DBConnMaxLifetime    string `mapstructure:"DB_CONN_MAX_LIFETIME"`
	DBConnMaxIdleTime    string `mapstructure:"DB_CONN_MAX_IDLE_TIME"`
	JWTAccessExpiry     string `mapstructure:"JWT_ACCESS_EXPIRY"`
	JWTRefreshExpiry    string `mapstructure:"JWT_REFRESH_EXPIRY"`
	DBStatementTimeout   int    `mapstructure:"DB_STATEMENT_TIMEOUT"`
	DBIdleInTxTimeout    int    `mapstructure:"DB_IDLE_IN_TX_TIMEOUT"`
}

var AppConfig Config

var (
	logFatalf      = logger.Fatalf
	viperUnmarshal = viper.Unmarshal
)

func LoadConfig() {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("PORT", "3000")
	viper.SetDefault("HOST", "0.0.0.0")
	viper.SetDefault("RATE_LIMIT_MAX", 100)
	viper.SetDefault("RATE_LIMIT_WINDOW", "1m")
	viper.SetDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable") // NOSONAR
	viper.SetDefault("REDIS_URL", "redis://localhost:6379")
	viper.SetDefault("FIRST_USER", "admin@email.com")
	viper.SetDefault("FIRST_PASSWORD", "admin@123") // NOSONAR
	viper.SetDefault("PDF_SERVICE_URL", "http://localhost:8889")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 50)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 10)
	viper.SetDefault("DB_CONN_MAX_LIFETIME", "30m")
	viper.SetDefault("DB_CONN_MAX_IDLE_TIME", "5m")
	viper.SetDefault("JWT_ISSUER", "backend-go")
	viper.SetDefault("JWT_AUDIENCE", "backend-go-api")
	viper.SetDefault("TRUSTED_PROXIES", "")
	viper.SetDefault("JWT_ACCESS_EXPIRY", "15m")
	viper.SetDefault("JWT_REFRESH_EXPIRY", "168h")
	viper.SetDefault("DB_STATEMENT_TIMEOUT", 30000)
	viper.SetDefault("DB_IDLE_IN_TX_TIMEOUT", 60000)

	if err := viper.ReadInConfig(); err != nil {
		logger.Log.Sugar().Warnf("Aviso: arquivo .env não encontrado, usando variáveis de ambiente: %v", err)
	}

	if err := viperUnmarshal(&AppConfig); err != nil {
		logFatalf("Falha ao parsear configurações: %v", err)
	}
}
