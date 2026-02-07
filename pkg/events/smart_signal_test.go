package events

import (
	"context"
	"testing"
	"time"
)

type mockQuitHandler struct {
	quitCalled chan bool
}

func (m *mockQuitHandler) HandleEvent(ctx context.Context, e Event) error {
	m.quitCalled <- true
	return nil
}

func TestSmartSignalHandler(t *testing.T) {
	// Setup
	suspendHandler := NewSuspendHandler()
	quitHandler := &mockQuitHandler{quitCalled: make(chan bool, 1)}

	smartHandler := NewSmartSignalHandler(suspendHandler, quitHandler)

	ctx := context.Background()

	// Initial State: Not Suspended.
	// Handling an event should trigger Suspend (not Quit).

	// We need to ensure SuspendHandler handles the event correctly.
	// SuspendHandler likely needs hooks to be useful, but here we just check state transition.
	// Actually SuspendHandler.HandleEvent(SuspendEvent) should toggle state.

	// Create a mock event
	evt := mockEvent{name: "SIGINT"}

	// First Trigger: Should Suspend
	if err := smartHandler.HandleEvent(ctx, evt); err != nil {
		t.Errorf("HandleEvent failed: %v", err)
	}

	// Verify state is suspended
	state := suspendHandler.State().(map[string]any)
	if suspended, ok := state["suspended"].(bool); !ok || !suspended {
		t.Error("Expected to be suspended after first signal")
	}

	// Verify Quit was NOT called
	select {
	case <-quitHandler.quitCalled:
		t.Error("Quit handler should not be called on first signal")
	default:
		// OK
	}

	// Second Trigger: Already Suspended -> Trigger Quit.
	if err := smartHandler.HandleEvent(ctx, evt); err != nil {
		t.Errorf("HandleEvent (2nd) failed: %v", err)
	}

	// Verify Quit WAS called
	select {
	case <-quitHandler.quitCalled:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("Quit handler should be called on second signal")
	}

	// Third Trigger: Already Quitting -> Ignored (Idempotent).
	// Reset mock channel
	close(quitHandler.quitCalled)
	quitHandler.quitCalled = make(chan bool, 1)

	if err := smartHandler.HandleEvent(ctx, evt); err != nil {
		t.Errorf("HandleEvent (3rd) failed: %v", err)
	}

	select {
	case <-quitHandler.quitCalled:
		t.Error("Quit handler should not be called again if already quitting")
	default:
		// OK
	}
}
