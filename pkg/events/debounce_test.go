package events_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/events"
)

// captureHandler stores the events it receives for assertion
type captureHandler struct {
	mu       sync.Mutex
	received []events.Event
}

func (h *captureHandler) HandleEvent(ctx context.Context, e events.Event) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.received = append(h.received, e)
	return nil
}

// customEvent is a mock event that supports a custom string payload
type customEvent struct {
	payload string
}

func (c customEvent) String() string {
	return c.payload
}

func TestDebounceHandler_DefaultKeepLatest(t *testing.T) {
	capture := &captureHandler{}
	window := 50 * time.Millisecond

	// Nil merge function means default "keep latest" behavior
	handler := events.DebounceHandler(capture, window, nil)
	ctx := context.Background()

	// Fire 3 events rapidly (well within the 50ms window)
	handler.HandleEvent(ctx, customEvent{"event-1"})
	handler.HandleEvent(ctx, customEvent{"event-2"})
	handler.HandleEvent(ctx, customEvent{"event-3"})

	// Wait for the window to pass
	time.Sleep(window + 10*time.Millisecond)

	capture.mu.Lock()
	defer capture.mu.Unlock()

	if len(capture.received) != 1 {
		t.Fatalf("expected exactly 1 event to be emitted, got %d", len(capture.received))
	}

	eventName := capture.received[0].String()
	if eventName != "event-3" {
		t.Errorf("expected final event to be 'event-3', got %q", eventName)
	}
}

func TestDebounceHandler_CustomMerge(t *testing.T) {
	capture := &captureHandler{}
	window := 50 * time.Millisecond

	// Custom merge function concatenates payloads
	mergeFunc := func(a, b events.Event) events.Event {
		return customEvent{payload: a.String() + "," + b.String()}
	}

	handler := events.DebounceHandler(capture, window, mergeFunc)
	ctx := context.Background()

	// Fire 3 events rapidly
	handler.HandleEvent(ctx, customEvent{"A"})
	handler.HandleEvent(ctx, customEvent{"B"})
	handler.HandleEvent(ctx, customEvent{"C"})

	// Wait for the window to pass
	time.Sleep(window + 10*time.Millisecond)

	capture.mu.Lock()
	defer capture.mu.Unlock()

	if len(capture.received) != 1 {
		t.Fatalf("expected exactly 1 event to be emitted, got %d", len(capture.received))
	}

	eventName := capture.received[0].String()
	if eventName != "A,B,C" {
		t.Errorf("expected final event to be 'A,B,C', got %q", eventName)
	}
}

func TestDebounceHandler_Cancellation(t *testing.T) {
	capture := &captureHandler{}
	window := 50 * time.Millisecond

	handler := events.DebounceHandler(capture, window, nil)

	// Create a context that will be cancelled before the timer fires
	ctx, cancel := context.WithCancel(context.Background())

	handler.HandleEvent(ctx, customEvent{"event-1"})

	// Cancel immediately
	cancel()

	// Wait for the window to pass
	time.Sleep(window + 10*time.Millisecond)

	capture.mu.Lock()
	defer capture.mu.Unlock()

	if len(capture.received) != 0 {
		t.Fatalf("expected 0 events to be emitted because context was cancelled, got %d", len(capture.received))
	}
}
