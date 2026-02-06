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

func TestInfo_Logging(t *testing.T) {
	buf := &bytes.Buffer{}
	customLogger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	SetLogger(customLogger)
	defer SetLogger(nil)

	Info("test message", "key", "value")

	output := buf.String()
	if output == "" {
		t.Error("Info() did not produce output")
	}

	if !bytes.Contains([]byte(output), []byte("test message")) {
		t.Errorf("Output does not contain 'test message': %s", output)
	}
}

func TestWarn_Logging(t *testing.T) {
	buf := &bytes.Buffer{}
	customLogger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))
	SetLogger(customLogger)
	defer SetLogger(nil)

	Warn("warning message", "severity", "high")

	output := buf.String()
	if output == "" {
		t.Error("Warn() did not produce output")
	}

	if !bytes.Contains([]byte(output), []byte("warning message")) {
		t.Errorf("Output does not contain 'warning message': %s", output)
	}
}

func TestError_Logging(t *testing.T) {
	buf := &bytes.Buffer{}
	customLogger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	SetLogger(customLogger)
	defer SetLogger(nil)

	Error("error message", "error_code", 500)

	output := buf.String()
	if output == "" {
		t.Error("Error() did not produce output")
	}

	if !bytes.Contains([]byte(output), []byte("error message")) {
		t.Errorf("Output does not contain 'error message': %s", output)
	}
}

func TestDebug_Logging_BelowLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	customLogger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelInfo, // Debug is below Info
	}))
	SetLogger(customLogger)
	defer SetLogger(nil)

	Debug("debug message")

	output := buf.String()
	if output != "" {
		t.Errorf("Debug() should not produce output at INFO level, got: %s", output)
	}
}

func TestDebug_Logging_AboveLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	customLogger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	SetLogger(customLogger)
	defer SetLogger(nil)

	Debug("debug message", "detail", "test")

	output := buf.String()
	if output == "" {
		t.Error("Debug() did not produce output at DEBUG level")
	}

	if !bytes.Contains([]byte(output), []byte("debug message")) {
		t.Errorf("Output does not contain 'debug message': %s", output)
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
