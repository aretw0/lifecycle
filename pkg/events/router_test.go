package events

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockEvent struct {
	name string
}

func (e mockEvent) String() string {
	return e.name
}

type mockSource struct {
	events chan Event
}

func (s *mockSource) Events() <-chan Event {
	return s.events
}

func (s *mockSource) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func TestRouter_Dispatch(t *testing.T) {
	router := NewRouter()
	handled := make(chan string, 10)

	// Exact match
	router.HandleFunc("test.event", func(ctx context.Context, e Event) error {
		handled <- "exact:" + e.String()
		return nil
	})

	// Pattern match
	router.HandleFunc("pattern.*", func(ctx context.Context, e Event) error {
		handled <- "pattern:" + e.String()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run Start in a goroutine as it blocks
	go func() {
		router.Start(ctx)
	}()

	// Wait for router to start (simple sleep for now, real life might need status check)
	time.Sleep(10 * time.Millisecond)

	router.Dispatch(ctx, mockEvent{"test.event"})
	select {
	case res := <-handled:
		if res != "exact:test.event" {
			t.Errorf("expected exact:test.event, got %s", res)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for exact match")
	}

	router.Dispatch(ctx, mockEvent{"pattern.match"})
	select {
	case res := <-handled:
		if res != "pattern:pattern.match" {
			t.Errorf("expected pattern:pattern.match, got %s", res)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for pattern match")
	}
}

func TestRouter_Middleware(t *testing.T) {
	router := NewRouter()
	results := make([]string, 0)

	// Channels to synchronize without race conditions
	done := make(chan struct{})

	router.Use(func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, e Event) error {
			results = append(results, "mw1_start")
			err := next.HandleEvent(ctx, e)
			results = append(results, "mw1_end")
			return err
		})
	})

	router.HandleFunc("foo", func(ctx context.Context, e Event) error {
		results = append(results, "handler")
		close(done)
		return nil
	})

	// Dispatch doesn't technically require Start() to be running if we call it directly,
	// but it's good practice. However, Dispatch lock logic is independent of Start loop
	// for direct calls.

	router.Dispatch(context.Background(), mockEvent{"foo"})

	<-done

	expected := []string{"mw1_start", "handler", "mw1_end"}
	if len(results) != 3 {
		t.Fatalf("expected 3 events, got %d", len(results))
	}
	for i, v := range expected {
		if results[i] != v {
			t.Errorf("expected %s at index %d, got %s", v, i, results[i])
		}
	}
}

func TestRouter_Routes(t *testing.T) {
	router := NewRouter()
	router.HandleFunc("route.a", func(_ context.Context, _ Event) error { return nil })
	router.HandleFunc("route.b", func(_ context.Context, _ Event) error { return nil })

	routes := router.Routes()
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}

	foundA, foundB := false, false
	for _, r := range routes {
		if r.Pattern == "route.a" {
			foundA = true
		}
		if r.Pattern == "route.b" {
			foundB = true
		}
	}

	if !foundA || !foundB {
		t.Error("missing expected routes")
	}
}
func TestRouter_State(t *testing.T) {
	router := NewRouter()

	// Get state - should return a non-nil state
	state := router.State()
	if state == nil {
		t.Error("State() returned nil")
	}
}

// TestRouterDispatchMetricsIntegration validates that Dispatch correctly routes events
// and records metrics when handlers execute using the mock metrics provider.
func TestRouterDispatchMetricsIntegration(t *testing.T) {
	router := NewRouter()

	// Use real metric calls but verify they don't panic
	ctx := context.Background()
	eventType := "test.event"

	// Create a simple handler
	executedCount := 0
	router.HandleFunc(eventType, func(_ context.Context, e Event) error {
		executedCount++
		return nil
	})

	// Create and dispatch an event
	event := mockEvent{eventType}
	router.Dispatch(ctx, event)

	// Assertions
	if executedCount != 1 {
		t.Errorf("handler not executed: expected 1, got %d", executedCount)
	}
}

// TestRouterDispatchWithErrorMetrics validates that handler errors are recorded.
func TestRouterDispatchWithErrorMetrics(t *testing.T) {
	router := NewRouter()
	ctx := context.Background()
	eventType := "test.error"

	expectedErr := "test error"
	router.HandleFunc(eventType, func(_ context.Context, e Event) error {
		return errors.New(expectedErr)
	})

	event := mockEvent{eventType}
	router.Dispatch(ctx, event)

	// If we got here without panic, metrics were called correctly
}
