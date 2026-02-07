package log

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestSetLogger_NilLogger(t *testing.T) {
	// Set to a custom logger
	customLogger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	SetLogger(customLogger)

	if GetLogger() != customLogger {
		t.Error("SetLogger(customLogger) did not set the logger")
	}

	// Reset to nil (should use default)
	SetLogger(nil)
	if GetLogger() == customLogger {
		t.Error("SetLogger(nil) should reset to default logger")
	}
}

func TestSetLogger_CustomLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	customLogger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	SetLogger(customLogger)
	defer SetLogger(nil) // Reset

	retrieved := GetLogger()
	if retrieved != customLogger {
		t.Error("GetLogger() did not return the custom logger")
	}
}

func TestGetLogger_Default(t *testing.T) {
	// Reset to default
	SetLogger(nil)

	logger := GetLogger()
	if logger == nil {
		t.Error("GetLogger() returned nil")
	}
}

func TestGetLogger_Concurrent(t *testing.T) {
	// Verify concurrent access doesn't panic
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			_ = GetLogger()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestSetLogger_Concurrent(t *testing.T) {
	original := GetLogger()
	defer SetLogger(original)

	done := make(chan bool)

	for i := 0; i < 5; i++ {
		go func() {
			buf := &bytes.Buffer{}
			customLogger := slog.New(slog.NewTextHandler(buf, nil))
			SetLogger(customLogger)
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	// Should not panic
	_ = GetLogger()
}

// Consolidated behavioral test for logging levels
func TestLogging_Behavior(t *testing.T) {
	tests := []struct {
		name     string
		level    slog.Level
		logFunc  func(string, ...any)
		msg      string
		contains string
	}{
		{"Info", slog.LevelInfo, Info, "info msg", "info msg"},
		{"Warn", slog.LevelWarn, Warn, "warn msg", "warn msg"},
		{"Error", slog.LevelError, Error, "error msg", "error msg"},
		{"Debug", slog.LevelDebug, Debug, "debug msg", "debug msg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			customLogger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: tt.level}))
			SetLogger(customLogger)
			defer SetLogger(nil)

			tt.logFunc(tt.msg)

			output := buf.String()
			if !bytes.Contains([]byte(output), []byte(tt.contains)) {
				t.Errorf("%s() did not produce expected output: %s", tt.name, output)
			}
		})
	}
}

func TestWithContext(t *testing.T) {
	ctx := context.Background()
	logger := WithContext(ctx)

	if logger == nil {
		t.Error("WithContext() returned nil")
	}

	// Should return the same logger as GetLogger()
	if logger != GetLogger() {
		t.Error("WithContext() should return GetLogger()")
	}
}

func TestWithContext_CustomLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	customLogger := slog.New(slog.NewTextHandler(buf, nil))
	SetLogger(customLogger)
	defer SetLogger(nil)

	ctx := context.Background()
	logger := WithContext(ctx)

	if logger != customLogger {
		t.Error("WithContext() should return the custom logger")
	}
}

func TestMultiple_LogCalls(t *testing.T) {
	buf := &bytes.Buffer{}
	customLogger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	SetLogger(customLogger)
	defer SetLogger(nil)

	Info("info", "a", 1)
	Warn("warn", "b", 2)
	Error("error", "c", 3)
	Debug("debug", "d", 4)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("info")) {
		t.Error("Output missing 'info'")
	}
	if !bytes.Contains([]byte(output), []byte("warn")) {
		t.Error("Output missing 'warn'")
	}
	if !bytes.Contains([]byte(output), []byte("error")) {
		t.Error("Output missing 'error'")
	}
	if !bytes.Contains([]byte(output), []byte("debug")) {
		t.Error("Output missing 'debug'")
	}
}
