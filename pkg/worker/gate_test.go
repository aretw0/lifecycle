package worker

import (
	"context"
	"testing"
	"time"
)

func TestQuiescenceGate_ContextCancel(t *testing.T) {
	g := NewQuiescenceGate()
	g.RequestPause()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := g.Check(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got %v", err)
	}
}

func TestQuiescenceGate_Resume(t *testing.T) {
	g := NewQuiescenceGate()
	g.RequestPause()

	ctx := context.Background()
	done := make(chan error)

	go func() {
		done <- g.Check(ctx)
	}()

	// Wait a bit to ensure it's blocked
	time.Sleep(20 * time.Millisecond)

	g.Resume()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Expected nil error after resume, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Gate didn't resume in time")
	}
}

func TestQuiescenceGate_WaitPaused(t *testing.T) {
	g := NewQuiescenceGate()
	g.RequestPause()

	ctx := context.Background()

	// Start worker check in background
	go g.Check(ctx)

	// Wait for worker to reach quiescence
	err := g.WaitPaused(ctx)
	if err != nil {
		t.Errorf("WaitPaused failed: %v", err)
	}
}
