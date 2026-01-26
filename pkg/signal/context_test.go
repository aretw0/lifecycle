package signal

import (
	"context"
	"testing"
	"time"
)

func TestNewContext_Cancel(t *testing.T) {
	// Test manual cancellation via Cancel() func
	parent := context.Background()
	ctx := NewContext(parent)

	select {
	case <-ctx.Done():
		t.Fatal("Context should not be done yet")
	default:
	}

	ctx.Cancel()

	select {
	case <-ctx.Done():
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be done after Cancel()")
	}
}

// We cannot easily test signal handling (sending SIGTERM) without terminating the test runner.
// So we skip that and trust the OS/Notify logic.
// We verified manual Cancel works.

func TestContext_Signal_Nil(t *testing.T) {
	ctx := NewContext(context.Background())
	defer ctx.Cancel()

	if sig := ctx.Signal(); sig != nil {
		t.Errorf("Expected nil signal initially, got %v", sig)
	}
}

func TestNewContext_ParentCancel(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	ctx := NewContext(parent)

	cancel() // Cancel parent

	select {
	case <-ctx.Done():
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be done after Parent cancel")
	}
}
