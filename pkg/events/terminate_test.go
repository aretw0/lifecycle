package events

import (
	"context"
	"errors"
	"testing"
)

type mockHandler struct {
	called bool
	err    error
}

func (m *mockHandler) HandleEvent(ctx context.Context, e Event) error {
	m.called = true
	return m.err
}

func TestTerminateHandler_Success(t *testing.T) {
	suspend := &mockHandler{}
	shutdown := &mockHandler{}

	h := NewTerminate(suspend, shutdown)

	err := h.HandleEvent(context.Background(), TerminateEvent{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !suspend.called {
		t.Error("suspend handler not called")
	}
	if !shutdown.called {
		t.Error("shutdown handler not called")
	}
}

func TestTerminateHandler_SuspendFailure_Continue(t *testing.T) {
	suspend := &mockHandler{err: errors.New("suspend failed")}
	shutdown := &mockHandler{}

	// Default is ContinueOnFailure = true
	h := NewTerminate(suspend, shutdown)

	err := h.HandleEvent(context.Background(), TerminateEvent{})
	if err == nil {
		t.Error("expected error from suspend")
	}

	if !suspend.called {
		t.Error("suspend handler not called")
	}
	if !shutdown.called {
		t.Error("shutdown handler should be called on continue")
	}
}

func TestTerminateHandler_SuspendFailure_Stop(t *testing.T) {
	suspend := &mockHandler{err: errors.New("suspend failed")}
	shutdown := &mockHandler{}

	h := NewTerminate(suspend, shutdown, WithContinueOnFailure(false))

	err := h.HandleEvent(context.Background(), TerminateEvent{})
	if err == nil {
		t.Error("expected error from suspend")
	}

	if !shutdown.called {
		// OK
	} else {
		t.Error("shutdown handler should NOT be called on failure")
	}
}
