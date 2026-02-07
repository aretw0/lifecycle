package events

import (
	"context"
	"testing"
	"time"
)

func TestRouter_State(t *testing.T) {
	r := NewRouter()

	// Register some routes
	r.HandleFunc("test/a", func(ctx context.Context, e Event) error { return nil })
	r.HandleFunc("test/b", func(ctx context.Context, e Event) error { return nil })

	// Add middleware
	r.Use(func(next Handler) Handler {
		return next
	})

	// Start in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		r.Start(ctx)
	}()

	// Wait for running state
	timeout := time.After(1 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	started := false
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for router to start")
		case <-ticker.C:
			s, ok := r.State().(RouterState)
			if !ok {
				continue
			}
			if s.Running {
				started = true
				goto verified
			}
		}
	}
verified:
	if !started {
		t.Fatal("Router failed to start")
	}

	// Check State
	state := r.State()
	routerState, ok := state.(RouterState)
	if !ok {
		t.Fatalf("State() returned unexpected type: %T", state)
	}

	if !routerState.Running {
		t.Error("Expected Running=true")
	}

	if routerState.Middlewares != 1 {
		t.Errorf("Expected 1 middleware, got %d", routerState.Middlewares)
	}

	if len(routerState.Routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routerState.Routes))
	}

	// Verify routes content
	routeMap := make(map[string]bool)
	for _, route := range routerState.Routes {
		routeMap[route.Pattern] = true
	}

	if !routeMap["test/a"] {
		t.Error("Missing route test/a")
	}
	if !routeMap["test/b"] {
		t.Error("Missing route test/b")
	}
}
