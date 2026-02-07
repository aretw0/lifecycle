package worker

import (
	"context"
	"testing"
	"time"
)

func TestBaseWorker_DefaultMethods(t *testing.T) {
	base := NewBaseWorker("TestWorker")

	// Test String
	if got := base.String(); got != "TestWorker" {
		t.Errorf("String() = %q, want %q", got, "TestWorker")
	}

	// Test State
	state := base.State()
	if state.Name != "TestWorker" {
		t.Errorf("State().Name = %q, want %q", state.Name, "TestWorker")
	}

	// Test Stop (should be no-op)
	ctx := context.Background()
	if err := base.Stop(ctx); err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}

	// Test Wait (channel should exist)
	if base.Wait() == nil {
		t.Error("Wait() returned nil channel")
	}
}

func TestBaseWorker_StartFunc(t *testing.T) {
	base := NewBaseWorker("AsyncWorker")

	called := false
	err := base.StartFunc(context.Background(), func(ctx context.Context) error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("StartFunc() returned error: %v", err)
	}

	// Wait for goroutine to complete
	select {
	case err := <-base.Wait():
		if err != nil {
			t.Errorf("Worker function returned error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for worker to complete")
	}

	if !called {
		t.Error("Worker function was not called")
	}
}

func TestBaseWorker_StartFunc_ContextCancellation(t *testing.T) {
	base := NewBaseWorker("CancelWorker")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := base.StartFunc(ctx, func(ctx context.Context) error {
		return ctx.Err()
	})

	if err != nil {
		t.Errorf("StartFunc() returned error: %v", err)
	}

	// Wait for goroutine to complete
	select {
	case err := <-base.Wait():
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for worker to complete")
	}
}

func TestBaseWorker_Embedding(t *testing.T) {
	// Test that embedding works correctly
	type CustomWorker struct {
		BaseWorker
		customField string
	}

	worker := &CustomWorker{
		BaseWorker:  *NewBaseWorker("Custom"),
		customField: "test",
	}

	// Verify embedded methods work
	if worker.String() != "Custom" {
		t.Errorf("Embedded String() = %q, want %q", worker.String(), "Custom")
	}

	if worker.customField != "test" {
		t.Error("Custom field not preserved")
	}

	// Verify can call StartFunc through embedding
	done := make(chan bool)
	worker.StartFunc(context.Background(), func(ctx context.Context) error {
		close(done)
		return nil
	})

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("StartFunc through embedding did not work")
	}
}

func TestBaseWorker_ExportState(t *testing.T) {
	w := NewBaseWorker("test")

	state := w.ExportState(func(s *State) {
		s.Metadata = map[string]string{"foo": "bar"}
	})

	if state.Name != "test" {
		t.Errorf("Expected name test, got %s", state.Name)
	}
	if state.Metadata["foo"] != "bar" {
		t.Errorf("Expected metadata foo=bar, got %s", state.Metadata["foo"])
	}
}
