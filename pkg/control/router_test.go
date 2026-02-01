package control

import (
	"context"
	"testing"
	"time"
)

// MockSource is a manual source for testing.
type MockSource struct {
	ch chan Event
}

func (s *MockSource) Events() <-chan Event { return s.ch }
func (s *MockSource) Start(ctx context.Context) error {
	defer close(s.ch)
	<-ctx.Done()
	return nil
}

type MockEvent struct {
	Name string
}

func (e MockEvent) String() string { return e.Name }

func TestRouter_Dispatch(t *testing.T) {
	router := NewRouter()
	src := &MockSource{ch: make(chan Event, 1)}
	router.AddSource(src)

	reactionCalled := make(chan bool)
	router.On("MockEvent:Test", func(ctx context.Context) error {
		reactionCalled <- true
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start router in background
	go router.Start(ctx)

	// Emit event
	src.ch <- MockEvent{Name: "MockEvent:Test"}

	select {
	case <-reactionCalled:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for reaction")
	}
}
