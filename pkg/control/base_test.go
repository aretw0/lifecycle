package control

import (
	"context"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/metrics"
	"github.com/aretw0/lifecycle/pkg/metrics/mock"
)

func TestBaseSource_Events(t *testing.T) {
	base := NewBaseSource("test", 10)

	events := base.Events()
	if events == nil {
		t.Fatal("Events() returned nil channel")
	}

	// Should be receive-only
	_, ok := interface{}(events).(<-chan Event)
	if !ok {
		t.Error("Events() should return receive-only channel")
	}
}

func TestBaseSource_Emit(t *testing.T) {
	base := NewBaseSource("test", 10)

	// Create test event
	event := testEvent{name: "test"}

	// Emit in goroutine to avoid blocking
	done := make(chan bool)
	go func() {
		_ = base.Emit(context.Background(), event)
		done <- true
	}()

	// Receive event
	select {
	case received := <-base.Events():
		if received.String() != "test" {
			t.Errorf("Expected 'test', got %q", received.String())
		}
	case <-done:
		t.Fatal("Emit completed before event was received")
	}

	<-done // Wait for goroutine
}

func TestBaseSource_BufferSize(t *testing.T) {
	bufferSize := 3
	base := NewBaseSource("test", bufferSize)

	// Should be able to emit up to bufferSize without blocking
	ctx := context.Background()
	for i := 0; i < bufferSize; i++ {
		_ = base.Emit(ctx, testEvent{name: "event"})
	}

	// Verify all events are in buffer
	for i := 0; i < bufferSize; i++ {
		select {
		case <-base.Events():
			// Good
		default:
			t.Fatalf("Expected %d events in buffer, got fewer", bufferSize)
		}
	}
}

func TestBaseSource_Close(t *testing.T) {
	base := NewBaseSource("test", 10)

	base.Close()

	// Receiving from closed channel should return zero value and false
	event, ok := <-base.Events()
	if ok {
		t.Errorf("Expected channel to be closed, but received event: %v", event)
	}
}

func TestBaseSource_EmitAfterClose(t *testing.T) {
	base := NewBaseSource("test", 10)
	base.Close()

	// This should panic (sending on closed channel)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when emitting to closed channel")
		}
	}()

	_ = base.Emit(context.Background(), testEvent{name: "test"})
}

func TestBaseSource_MultipleEmits(t *testing.T) {
	base := NewBaseSource("test", 100)

	count := 50
	done := make(chan bool)

	// Emit multiple events concurrently
	go func() {
		ctx := context.Background()
		for i := 0; i < count; i++ {
			_ = base.Emit(ctx, testEvent{name: "event"})
		}
		done <- true
	}()

	// Receive all events
	for i := 0; i < count; i++ {
		<-base.Events()
	}

	<-done
}

func TestBaseSource_TryEmit(t *testing.T) {
	base := NewBaseSource("test", 1)

	if !base.TryEmit(testEvent{name: "1"}) {
		t.Error("TryEmit failed on empty buffer")
	}

	if base.TryEmit(testEvent{name: "2"}) {
		t.Error("TryEmit succeeded on full buffer")
	}

	<-base.Events() // Empty one
	if !base.TryEmit(testEvent{name: "3"}) {
		t.Error("TryEmit failed after buffer cleared")
	}
}

func TestBaseSource_Metrics(t *testing.T) {
	m := mock.New()
	metrics.SetProvider(m)
	defer metrics.SetProvider(&metrics.NoOpProvider{})

	name := "metric-source"
	base := NewBaseSource(name, 1) // Buffer size 1

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Emit should increment total emissions
	_ = base.Emit(ctx, testEvent{name: "1"})
	if m.EventsEmitted[name] != 1 {
		t.Errorf("Expected 1 emission, got %d", m.EventsEmitted[name])
	}

	// 2. Test Backpressure Metrics
	// Buffer is now full because "1" is in it (1 byte buffer).
	// Next Emit should block and trigger waiting metrics.
	blockDone := make(chan bool)
	go func() {
		_ = base.Emit(ctx, testEvent{name: "2"})
		blockDone <- true
	}()

	// Wait a bit for it to block
	time.Sleep(50 * time.Millisecond)

	m.Mu.Lock()
	waiting := m.EventsWaiting[name]
	m.Mu.Unlock()

	if waiting != 1 {
		t.Errorf("Expected 1 goroutine waiting, got %d", waiting)
	}

	// Unblock by receiving
	<-base.Events()
	<-blockDone

	m.Mu.Lock()
	waitingAfter := m.EventsWaiting[name]
	duration := m.EventBlockDurations[name]
	m.Mu.Unlock()

	if waitingAfter != 0 {
		t.Errorf("Expected 0 goroutines waiting after unblock, got %d", waitingAfter)
	}

	if duration <= 0 {
		t.Error("Expected positive block duration metric to be recorded")
	}
}

// testEvent is a simple Event implementation for testing.
type testEvent struct {
	name string
}

func (e testEvent) String() string {
	return e.name
}
