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

func TestDebounceHandler_MaxWait(t *testing.T) {
	capture := &captureHandler{}
	window := 50 * time.Millisecond
	maxWait := 100 * time.Millisecond

	handler := events.DebounceHandler(capture, window, nil, events.WithMaxWait(maxWait))
	ctx := context.Background()

	// Fire events every 30ms for 150ms total.
	// Window is 50ms, so normally trailing edge never fires until burst stops.
	for i := 0; i < 6; i++ {
		handler.HandleEvent(ctx, customEvent{payload: "spam"})
		time.Sleep(30 * time.Millisecond)
	}

	// Wait for the final trailing edge
	time.Sleep(window + 10*time.Millisecond)

	capture.mu.Lock()
	defer capture.mu.Unlock()

	// Expected:
	// t=0, t=30, t=60, t=90 (within 100ms maxWait)
	// t=120 forces flush (because 120ms >= 100ms) -> First event emitted
	// t=150 starts new burst
	// Trail edge handles t=150 -> Second event emitted
	if len(capture.received) != 2 {
		t.Fatalf("expected 2 events due to MaxWait forcing a flush mid-burst, got %d", len(capture.received))
	}
}
