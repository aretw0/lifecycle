package control

import (
	"context"
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
