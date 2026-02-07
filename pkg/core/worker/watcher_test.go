package worker

import (
	"context"
	"testing"
	"time"
)

func TestBaseWorker_Watch(t *testing.T) {
	bw := NewBaseWorker("test-worker")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Subscribe
	ch := bw.Watch(ctx)

	// 2. Emit change
	oldState := State{Status: StatusPending}
	newState := State{Status: StatusRunning}
	bw.emitStateChange(oldState, newState)

	// 3. Verify receipt
	select {
	case change := <-ch:
		if change.ComponentID != "test-worker" {
			t.Errorf("Expected ComponentID test-worker, got %s", change.ComponentID)
		}
		if change.ComponentType != "worker" {
			t.Errorf("Expected ComponentType worker, got %s", change.ComponentType)
		}
		if change.OldState.Status != StatusPending {
			t.Errorf("Expected OldState Pending, got %v", change.OldState.Status)
		}
		if change.NewState.Status != StatusRunning {
			t.Errorf("Expected NewState Running, got %v", change.NewState.Status)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for state change event")
	}

	// 4. Verify cleanup on context cancel
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("Channel should be closed after context cancel")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for channel close")
	}

	bw.watchersMu.Lock()
	if len(bw.stateWatchers) != 0 {
		t.Errorf("Expected 0 watchers, got %d", len(bw.stateWatchers))
	}
	bw.watchersMu.Unlock()
}
