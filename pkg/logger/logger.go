package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey string

const RequestIDKey ctxKey = "requestId"

var Log *zap.Logger

func init() {
	// Inicializa um logger básico para evitar nil pointer dereference antes do InitLogger ser chamado
	Log, _ = zap.NewDevelopment()
}

var buildLogger = func(config zap.Config, options ...zap.Option) (*zap.Logger, error) {
	return config.Build(options...)
}

func InitLogger(environment string) {
	var zapConfig zap.Config

	if environment == "production" {
		zapConfig = zap.NewProductionConfig()
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var err error
	Log, err = buildLogger(zapConfig, zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}
}

func Info(msg string, fields ...zap.Field) {
	Log.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Log.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	Log.Fatal(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	Log.Debug(msg, fields...)
}

func WithContext(ctx context.Context) *zap.Logger {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return Log.With(zap.String("requestId", requestID))
	}
	return Log
}

// Para manter compatibilidade com log padrão em alguns lugares se necessário
func Printf(format string, v ...interface{}) {
	Log.Sugar().Infof(format, v...)
}

func Fatalf(format string, v ...interface{}) {
	Log.Sugar().Fatalf(format, v...)
}
