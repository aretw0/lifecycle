package events

import (
	"context"
	"testing"
)

func TestShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// cancel is passed to handler but context is separate?
	// Usually we cancel the context passed to app, so testing it requires checking ctx.Err()

	h := NewShutdown(cancel)

	if err := h.HandleEvent(ctx, ShutdownEvent{}); err != nil {
		t.Errorf("HandleEvent failed: %v", err)
	}

	if ctx.Err() == nil {
		t.Error("Context should be cancelled")
	}
}

func TestShutdownFunc(t *testing.T) {
	called := false
	fn := func() { called = true }

	h := NewShutdownFunc(fn)
	h.HandleEvent(context.Background(), ShutdownEvent{})

	if !called {
		t.Error("Shutdown function not called")
	}
}
