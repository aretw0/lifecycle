package runtime_test

import (
	"context"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/runtime"
)

func TestBlockWithTimeout_Success(t *testing.T) {
	done := make(chan struct{})
	go func() {
		close(done)
	}()

	err := runtime.BlockWithTimeout(done, 1*time.Second)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestBlockWithTimeout_Timeout(t *testing.T) {
	done := make(chan struct{})
	// Do NOT close done

	start := time.Now()
	err := runtime.BlockWithTimeout(done, 50*time.Millisecond)
	duration := time.Since(start)

	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}

	if duration < 50*time.Millisecond {
		t.Errorf("function returned too early: %v", duration)
	}
}
