package runtime

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockRunnable implements Runnable for testing
type MockRunnable struct {
	started bool
	err     error
}

func (m *MockRunnable) Start(ctx context.Context) error {
	m.started = true
	return m.err
}

func TestRun_WithJob(t *testing.T) {
	// Test v1-style Job (RunnableFunc)
	executed := false
	job := Job(func(ctx context.Context) error {
		executed = true
		return nil
	})

	// Run blocks until the Runnable finishes.
	// Since this Job returns immediately, Run should also return immediately.
	// The SignalContext created inside Run will be canceled upon exit.

	err := Run(job)
	if err != nil {
		t.Errorf("Run failed: %v", err)
	}
	if !executed {
		t.Error("Job was not executed")
	}
}

func TestRun_WithRunnable(t *testing.T) {
	// Test interface implementation
	mock := &MockRunnable{}

	err := Run(mock)
	if err != nil {
		t.Errorf("Run failed: %v", err)
	}
	if !mock.started {
		t.Error("Runnable.Start was not called")
	}
}

func TestRun_RunnableError(t *testing.T) {
	expectedErr := errors.New("startup failed")
	mock := &MockRunnable{err: expectedErr}

	err := Run(mock)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err != expectedErr {
		// Run propagates errors returned by the Runnable.
		if err.Error() != expectedErr.Error() {
			t.Errorf("Expected %v, got %v", expectedErr, err)
		}
	}
}

func TestRun_WaitsForBackgroundTasks(t *testing.T) {
	// This test verifies that Run waits for goroutines started with lifecycle.Go
	finished := make(chan struct{})

	job := Job(func(ctx context.Context) error {
		Go(ctx, func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			close(finished)
			return nil
		})
		return nil
	})

	start := time.Now()
	err := Run(job)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Run failed: %v", err)
	}

	select {
	case <-finished:
		// Success
	default:
		t.Error("Run returned before background task finished")
	}

	if duration < 50*time.Millisecond {
		t.Errorf("Run returned too early: %v", duration)
	}
}

func TestRun_LeakWarning(t *testing.T) {
	// This test exercises the timeout logic in waitForTasks.
	// We use a very short shutdown timeout.

	job := Job(func(ctx context.Context) error {
		// Start a goroutine that doesn't respect context and hangs
		Go(ctx, func(ctx context.Context) error {
			// Simulate a leak
			time.Sleep(200 * time.Millisecond)
			return nil
		})
		return nil
	})

	start := time.Now()
	// Set a timeout shorter than the sleep
	err := Run(job, WithShutdownTimeout(50*time.Millisecond))
	duration := time.Since(start)

	// Check that the duration is within an expected range, indicating the leak warning logic was triggered.
	// The goroutine sleeps for 200ms, and the timeout is 50ms, so it should take at least 50ms (the timeout)
	// but less than the full 200ms + some overhead (e.g., 10ms for safety).
	if duration < 50*time.Millisecond || duration > 210*time.Millisecond {
		t.Errorf("Run returned too early or too late, expected around 200ms (goroutine sleep) but got %v", duration)
	}

	if err != nil {
		t.Errorf("Run failed: %v", err)
	}

	// It should have taken at least 200ms because waitForTasks blocks until <-done
	// (meaning it waits for the leaked task to eventually finish even after warning)
	if duration < 200*time.Millisecond {
		t.Errorf("Run returned too early, should have waited for leaked task: %v", duration)
	}
}
