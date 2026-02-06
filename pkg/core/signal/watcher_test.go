package signal

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestFromContext(t *testing.T) {
	t.Run("ValidContext", func(t *testing.T) {
		sc := NewContext(context.Background())
		defer sc.Stop()

		retrieved, ok := FromContext(sc.Context)
		if !ok {
			t.Error("FromContext failed to retrieve signal.Context")
		}
		if retrieved != sc {
			t.Error("Retrieved context does not match original")
		}
	})

	t.Run("WrappedContext", func(t *testing.T) {
		sc := NewContext(context.Background())
		defer sc.Stop()

		wrapped := context.WithValue(sc.Context, "foo", "bar")
		retrieved, ok := FromContext(wrapped)
		if !ok {
			t.Error("FromContext failed to retrieve signal.Context from wrapped context")
		}
		if retrieved != sc {
			t.Error("Retrieved context does not match original")
		}
	})

	t.Run("InvalidContext", func(t *testing.T) {
		_, ok := FromContext(context.Background())
		if ok {
			t.Error("FromContext should return !ok for standard context")
		}
	})
}

func TestContext_Watch(t *testing.T) {
	sc := NewContext(context.Background())
	defer sc.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := sc.Watch(ctx)

	// Trigger a change: First, simulate a signal reception
	sc.mu.Lock()
	sc.signalCount = 1
	sc.sigVal = os.Interrupt
	sc.mu.Unlock()
	sc.emitStateChange(State{}, sc.State()) // Force an emission

	select {
	case change := <-ch:
		if change.NewState.Status.SignalCount != 1 {
			t.Errorf("Expected SignalCount 1, got %d", change.NewState.Status.SignalCount)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for first state change event")
	}

	// Trigger a second change: ResetSignalCount
	sc.ResetSignalCount()

	select {
	case change := <-ch:
		if change.ComponentType != "signal" {
			t.Errorf("Expected ComponentType 'signal', got %s", change.ComponentType)
		}
		// Based on ResetSignalCount implementation, newState.Status.SignalCount should be 0
		if change.NewState.Status.SignalCount != 0 {
			t.Errorf("Expected SignalCount 0, got %d", change.NewState.Status.SignalCount)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for reset state change event")
	}

	// Test cleanup: cancel the local watch context
	cancel()

	// Wait a bit for cleanup goroutine
	time.Sleep(50 * time.Millisecond)

	sc.watchersMu.RLock()
	watcherCount := len(sc.stateWatchers)
	sc.watchersMu.RUnlock()

	if watcherCount != 0 {
		t.Errorf("Expected 0 watchers after cancellation, got %d", watcherCount)
	}

	// Verify channel is closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("Watcher channel should be closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timed out waiting for channel closure")
	}
}

func TestContext_Watch_Blocking(t *testing.T) {
	sc := NewContext(context.Background())
	defer sc.Stop()

	// Create a watcher with a small buffer (it's 10 in implementation)
	// We want to verify that a slow consumer doesn't block the system.
	_ = sc.Watch(context.Background())

	// Fill buffer + 1
	for i := 0; i < 12; i++ {
		sc.ResetSignalCount()
	}

	// If we reach here, it's non-blocking (Success)
}

func TestContext_Wait(t *testing.T) {
	sc := NewContext(context.Background())
	defer sc.Stop()

	sc.OnShutdown(func() {
		time.Sleep(50 * time.Millisecond)
	})

	// Trigger shutdown manually via Cancel
	sc.Cancel()

	start := time.Now()
	sc.Wait()
	elapsed := time.Since(start)

	if elapsed < 50*time.Millisecond {
		t.Errorf("Wait() returned too early, expected at least 50ms, got %v", elapsed)
	}
}

func TestContext_Shutdown(t *testing.T) {
	sc := NewContext(context.Background())
	defer sc.Stop()

	sc.Shutdown() // Alias for Cancel

	select {
	case <-sc.Done():
		if sc.Reason() != ReasonManualCancel {
			t.Errorf("Expected ReasonManualCancel, got %v", sc.Reason())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Context not cancelled after Shutdown()")
	}
}

func TestContext_emitStateChange_Deduplication(t *testing.T) {
	sc := NewContext(context.Background())
	defer sc.Stop()

	ch := sc.Watch(context.Background())

	state := sc.State()
	// Emitting same state twice
	sc.emitStateChange(state, state)

	select {
	case <-ch:
		t.Error("Should not emit change if state didn't change")
	case <-time.After(100 * time.Millisecond):
		// Success
	}
}
