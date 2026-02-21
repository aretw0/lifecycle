package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/events"
)

type mockEvent struct {
	topic string
}

func (m mockEvent) String() string {
	return m.topic
}

func TestNotify_Success(t *testing.T) {
	ch := make(chan events.Event, 1)
	handler := events.Notify(ch)
	ctx := context.Background()

	e := mockEvent{"test/topic"}

	err := handler.HandleEvent(ctx, e)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	select {
	case received := <-ch:
		if received.String() != e.String() {
			t.Errorf("expected event %q, got %q", e.String(), received.String())
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func TestNotify_ChannelFull(t *testing.T) {
	// Unbuffered channel with no active receiver will block immediately
	ch := make(chan events.Event)
	handler := events.Notify(ch)
	ctx := context.Background()

	e := mockEvent{"test/topic"}

	err := handler.HandleEvent(ctx, e)
	if err != events.ErrNotHandled {
		t.Fatalf("expected ErrNotHandled for full channel, got %v", err)
	}
}

func TestNotify_ContextCancelled(t *testing.T) {
	ch := make(chan events.Event)
	handler := events.Notify(ch)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	e := mockEvent{"test/topic"}

	err := handler.HandleEvent(ctx, e)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
}
