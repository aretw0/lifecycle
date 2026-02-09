package events_test

import (
	"context"
	"testing"

	"github.com/aretw0/lifecycle/pkg/events"
)

type mockHandler struct {
	called bool
	err    error
}

func (m *mockHandler) HandleEvent(ctx context.Context, e events.Event) error {
	m.called = true
	return m.err
}

func TestEscalator_PrimaryHandling(t *testing.T) {
	primary := &mockHandler{}
	fallback := &mockHandler{}
	escalator := events.NewEscalator(primary, fallback)

	_ = escalator.HandleEvent(context.Background(), nil)

	if !primary.called {
		t.Error("Primary handler should be called")
	}
	if fallback.called {
		t.Error("Fallback handler should NOT be called")
	}
}

func TestEscalator_Escalation(t *testing.T) {
	primary := &mockHandler{err: events.ErrNotHandled}
	fallback := &mockHandler{}
	escalator := events.NewEscalator(primary, fallback)

	_ = escalator.HandleEvent(context.Background(), nil)

	if !primary.called {
		t.Error("Primary handler should be called first")
	}
	if !fallback.called {
		t.Error("Fallback handler should be called after ErrNotHandled")
	}

	// Reset state
	primary.called = false
	fallback.called = false

	// Second call should skip Primary and go straight to Fallback (Escalated state)
	_ = escalator.HandleEvent(context.Background(), nil)

	if primary.called {
		t.Error("Primary handler should NOT be called in escalated state")
	}
	if !fallback.called {
		t.Error("Fallback handler should be called in escalated state")
	}
}
