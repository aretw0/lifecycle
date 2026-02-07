package events_test

import (
	"context"
	"strings"
	"testing"

	"github.com/aretw0/lifecycle/pkg/events"
)

type customHandler struct{}

func (h *customHandler) HandleEvent(ctx context.Context, e events.Event) error { return nil }

func MyTestHandler(ctx context.Context, e events.Event) error { return nil }

type mockSource struct {
	events chan events.Event
}

func (s *mockSource) Events() <-chan events.Event { return s.events }
func (s *mockSource) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func TestRouter_Introspection(t *testing.T) {
	router := events.NewRouter()

	// 1. Register different types of handlers
	router.HandleFunc("func.handler", MyTestHandler)
	router.Handle("interface.handler", &customHandler{})

	// 2. Add middleware and source
	router.Use(func(next events.Handler) events.Handler { return next })
	router.AddSource(&mockSource{events: make(chan events.Event)})

	// 3. Verify Routes()
	routes := router.Routes()
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}

	foundFunc, foundInterface := false, false
	for _, r := range routes {
		if r.Pattern == "func.handler" {
			foundFunc = true
			if !strings.Contains(r.Handler, "MyTestHandler") {
				t.Errorf("expected handler name to contain MyTestHandler, got %s", r.Handler)
			}
		}
		if r.Pattern == "interface.handler" {
			foundInterface = true
			if r.Handler != "*events_test.customHandler" {
				t.Errorf("expected handler name *events_test.customHandler, got %s", r.Handler)
			}
		}
	}

	if !foundFunc || !foundInterface {
		t.Error("missing expected routes in introspection")
	}

	// 4. Verify State()
	state := router.State().(events.RouterState)
	if state.Middlewares != 1 {
		t.Errorf("expected 1 middleware, got %d", state.Middlewares)
	}
	if state.Sources != 1 {
		t.Errorf("expected 1 source, got %d", state.Sources)
	}
	if state.Running {
		t.Error("expected router not to be running")
	}

	// 5. Test running state
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go router.Start(ctx)

	// Wait a bit for start
	for i := 0; i < 10; i++ {
		state = router.State().(events.RouterState)
		if state.Running {
			break
		}
		// Minimal wait
	}

	if !state.Running {
		t.Error("expected router to be running in state")
	}
}

func TestRouter_RoutesIsolation(t *testing.T) {
	router := events.NewRouter()
	router.HandleFunc("a", MyTestHandler)

	routes1 := router.Routes()
	router.HandleFunc("b", MyTestHandler)
	routes2 := router.Routes()

	if len(routes1) != 1 {
		t.Errorf("routes1 should have 1 route, got %d", len(routes1))
	}
	if len(routes2) != 2 {
		t.Errorf("routes2 should have 2 routes, got %d", len(routes2))
	}
}
