package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/events"
)

func TestChannelSource(t *testing.T) {
	ch := make(chan events.Event, 1)
	src := events.NewChannelSource(ch)

	if src.Events() != ch {
		t.Error("Events() did not return the provided channel")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start should block until context is cancelled
	errCh := make(chan error, 1)
	go func() {
		errCh <- src.Start(ctx)
	}()

	// Send an event
	expected := &events.SuspendEvent{}
	ch <- expected

	select {
	case e := <-src.Events():
		if e != expected {
			t.Errorf("Received wrong event: %v", e)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive event from ChannelSource")
	}

	// Cancel context to stop source
	cancel()

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Start() did not return after context cancellation")
	}
}
