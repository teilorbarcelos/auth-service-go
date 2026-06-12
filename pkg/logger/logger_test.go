package logger

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestInitLogger(t *testing.T) {
	// Test production
	InitLogger("production")
	assert.NotNil(t, Log)

	// Test development/default
	InitLogger("development")
	assert.NotNil(t, Log)
}

func TestInitLogger_Panic(t *testing.T) {
	// Save old builder and restore after test
	oldBuilder := buildLogger
	defer func() { buildLogger = oldBuilder }()

	// Mock builder to return error
	buildLogger = func(config zap.Config, options ...zap.Option) (*zap.Logger, error) {
		return nil, assert.AnError
	}

	assert.Panics(t, func() {
		InitLogger("development")
	})
}

func TestLoggingMethods(t *testing.T) {
	observedLogger, logs := observer.New(zap.DebugLevel)
	// Replace global Log
	oldLog := Log
	Log = zap.New(observedLogger)
	defer func() { Log = oldLog }()

	Info("info message", zap.String("key", "value"))
	assert.Equal(t, 1, logs.FilterMessage("info message").Len())

	Warn("warn message")
	assert.Equal(t, 1, logs.FilterMessage("warn message").Len())

	Error("error message")
	assert.Equal(t, 1, logs.FilterMessage("error message").Len())

	Debug("debug message")
	assert.Equal(t, 1, logs.FilterMessage("debug message").Len())

	Printf("printf message %s", "formatted")
	assert.Equal(t, 1, logs.FilterMessage("printf message formatted").Len())
}

func TestWithContext(t *testing.T) {
	observedLogger, logs := observer.New(zap.DebugLevel)
	oldLog := Log
	Log = zap.New(observedLogger)
	defer func() { Log = oldLog }()

	// Without requestId
	ctx := context.Background()
	l := WithContext(ctx)
	l.Info("no request id")
	assert.Equal(t, 1, logs.FilterMessage("no request id").Len())

	// With requestId
	ctx = context.WithValue(context.Background(), RequestIDKey, "123")
	l = WithContext(ctx)
	l.Info("with request id")
	assert.Equal(t, 1, logs.FilterMessage("with request id").Len())
	
	// Check if requestId field is present
	found := false
	for _, entry := range logs.FilterMessage("with request id").All() {
		for _, field := range entry.Context {
			if field.Key == "requestId" && field.String == "123" {
				found = true
				break
			}
		}
	}
	assert.True(t, found)
}

func TestFatal(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		Fatal("fatal message")
		return
	}
	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestFatalf(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		Fatalf("fatalf message %s", "formatted")
		return
	}
	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestFatalf")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
