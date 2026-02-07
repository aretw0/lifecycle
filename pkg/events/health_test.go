package events

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestHealthCheckSource(t *testing.T) {
	// 1. Setup healthy check
	check := func(ctx context.Context) error { return nil }

	source := NewHealthCheckSource("test", check,
		WithHealthInterval(10*time.Millisecond),
		WithHealthStrategy(TriggerLevel), // Emit every time
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := source.Events()
	go source.Start(ctx)

	// Expect UP event
	select {
	case e := <-ch:
		he, ok := e.(HealthEvent)
		if !ok || he.Status != "UP" {
			t.Errorf("Expected UP event, got %v", e)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for health event")
	}
}

func TestHealthCheckSource_EdgeTrigger(t *testing.T) {
	// Toggle health safely
	var mu sync.Mutex
	healthy := true

	check := func(ctx context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		if healthy {
			return nil
		}
		return errors.New("failed")
	}

	source := NewHealthCheckSource("test", check,
		WithHealthInterval(10*time.Millisecond),
		WithHealthStrategy(TriggerEdge),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := source.Events()
	go source.Start(ctx)

	// 1. Initial State: Expect immediate UP event (transition from empty state).
	select {
	case e := <-ch:
		if he := e.(HealthEvent); he.Status != "UP" {
			t.Error("Initial event should be UP")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout 1")
	}

	// 2. Stay healthy -> No event (Edge)
	select {
	case <-ch:
		t.Error("Should not emit if status unchanged")
	case <-time.After(50 * time.Millisecond):
		// OK
	}

	// 3. Become unhealthy -> Emit DOWN
	mu.Lock()
	healthy = false
	mu.Unlock()
	select {
	case e := <-ch:
		if he := e.(HealthEvent); he.Status != "DOWN" {
			t.Error("Expected DOWN event")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout 2")
	}
}
