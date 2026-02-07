package introspection

import (
	"context"
	"testing"
	"time"
)

func TestNewWatcherAdapter(t *testing.T) {
	initialState := MockState{Value: "initial"}

	// Create a mock TypedWatcher (using exported type from introspection_test.go)
	mock := NewMockTypedWatcher(initialState)

	// Create the adapter
	adapter := NewWatcherAdapter("test-component", mock)

	if adapter.componentType != "test-component" {
		t.Errorf("Expected component type 'test-component', got %s", adapter.componentType)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consuming snapshots
	snapshotCh := adapter.Snapshots(ctx)

	// Trigger a change in the mock
	newState := MockState{Value: "updated"}
	change := StateChange[MockState]{
		ComponentID:   "test-id",
		ComponentType: "test-component",
		OldState:      initialState,
		NewState:      newState,
		Timestamp:     time.Now(),
	}
	mock.SendChange(change)

	// Verify we receive the adapted snapshot
	select {
	case snap := <-snapshotCh:
		if snap.ComponentID != "test-id" {
			t.Errorf("Expected component ID 'test-id', got %s", snap.ComponentID)
		}
		if snap.ComponentType != "test-component" {
			t.Errorf("Expected component type 'test-component', got %s", snap.ComponentType)
		}

		// Type assertion on payload (which is 'any' in StateSnapshot)
		state, ok := snap.Payload.(MockState)
		if !ok {
			t.Errorf("Expected payload type MockState, got %T", snap.Payload)
		} else if state.Value != "updated" {
			t.Errorf("Expected payload value 'updated', got %s", state.Value)
		}

	case <-time.After(500 * time.Millisecond):
		t.Error("Timeout waiting for snapshot")
	}
}
