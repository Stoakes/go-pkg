package log_test

import (
	"context"
	"testing"

	"github.com/Stoakes/go-pkg/log"
	"go.uber.org/zap/zapcore"
)

func TestBuildLogger(t *testing.T) {
	zapLogLevel := log.Setup(context.Background(), log.Options{
		Debug:         false,
		LogLevel:      "info",
		AppName:       "hello-app",
		DisableFields: true,
	})
	if zapLogLevel.Level() != zapcore.InfoLevel {
		t.Errorf("Error with log level,  expecting %v got %v", zapcore.InfoLevel, zapLogLevel.Level())
	}

	// Debug flag overrides log level
	zapLogLevel = log.Setup(context.Background(), log.Options{
		Debug:         true,
		LogLevel:      "warn",
		AppName:       "hello-app",
		DisableFields: true,
	})
	if zapLogLevel.Level() != zapcore.DebugLevel {
		t.Errorf("Error with log level & flag debug,  expecting %v got %v", zapcore.InfoLevel, zapLogLevel.Level())
	}
}
