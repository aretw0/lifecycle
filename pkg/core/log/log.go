package log

import (
	"context"
	"log/slog"
	"os"
	"sync"
)

var (
	defaultLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	loggerMu sync.RWMutex
	logger   = defaultLogger
)

// SetLogger overrides the global logger used by the library.
func SetLogger(l *slog.Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	if l == nil {
		logger = defaultLogger
		return
	}
	logger = l
}

// GetLogger returns the current global logger.
func GetLogger() *slog.Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return logger
}

// Info logs a message at LevelInfo.
func Info(msg string, args ...any) {
	GetLogger().Info(msg, args...)
}

// Warn logs a message at LevelWarn.
func Warn(msg string, args ...any) {
	GetLogger().Warn(msg, args...)
}

// Error logs a message at LevelError.
func Error(msg string, args ...any) {
	GetLogger().Error(msg, args...)
}

// Debug logs a message at LevelDebug.
func Debug(msg string, args ...any) {
	GetLogger().Debug(msg, args...)
}

// WithContext returns the logger with context.
func WithContext(ctx context.Context) *slog.Logger {
	return GetLogger() // Placeholder if we want to extract from ctx later
}



