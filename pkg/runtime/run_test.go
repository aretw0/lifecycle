package runtime

import (
	"context"
	"errors"
	"testing"
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
