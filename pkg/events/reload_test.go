package events

import (
	"context"
	"errors"
	"testing"
)

func TestReloadHandler(t *testing.T) {
	called := false
	cb := func(ctx context.Context) error {
		called = true
		return nil
	}

	h := NewReload(cb)

	if err := h.HandleEvent(context.Background(), ReloadEvent{}); err != nil {
		t.Errorf("HandleEvent failed: %v", err)
	}

	if !called {
		t.Error("OnReload not called")
	}
}

func TestReloadHandler_Error(t *testing.T) {
	cb := func(ctx context.Context) error {
		return errors.New("reload failed")
	}

	h := NewReload(cb)
	err := h.HandleEvent(context.Background(), ReloadEvent{})
	if err == nil || err.Error() != "reload failed" {
		t.Error("Expected error from OnReload")
	}
}
