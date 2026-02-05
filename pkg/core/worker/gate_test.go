package worker

import (
	"context"
	"testing"
	"time"
)

func TestSuspendGate_ContextCancel(t *testing.T) {
	g := NewSuspendGate()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	g.RequestPause() // Non-blocking

	err := g.Check(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got %v", err)
	}
}

func TestSuspendGate_Resume(t *testing.T) {
	g := NewSuspendGate()
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

func TestSuspendGate_Suspend(t *testing.T) {
	g := NewSuspendGate()
	ctx := context.Background()
	done := make(chan error)

	go func() {
		done <- g.Suspend(ctx)
	}()

	// Wait a bit to ensure Suspend is blocked
	time.Sleep(20 * time.Millisecond)

	// Call Check - this should unblock Suspend
	go g.Check(ctx)

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Expected nil error from Suspend, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Suspend didn't unblock in time")
	}
}



