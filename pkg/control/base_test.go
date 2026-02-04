package control

import (
	"testing"
)

func TestBaseSource_Events(t *testing.T) {
	base := NewBaseSource(10)

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
	base := NewBaseSource(10)

	// Create test event
	event := testEvent{name: "test"}

	// Emit in goroutine to avoid blocking
	done := make(chan bool)
	go func() {
		base.Emit(event)
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
	base := NewBaseSource(bufferSize)

	// Should be able to emit up to bufferSize without blocking
	for i := 0; i < bufferSize; i++ {
		base.Emit(testEvent{name: "event"})
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
	base := NewBaseSource(10)

	base.Close()

	// Receiving from closed channel should return zero value and false
	event, ok := <-base.Events()
	if ok {
		t.Errorf("Expected channel to be closed, but received event: %v", event)
	}
}

func TestBaseSource_EmitAfterClose(t *testing.T) {
	base := NewBaseSource(10)
	base.Close()

	// This should panic (sending on closed channel)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when emitting to closed channel")
		}
	}()

	base.Emit(testEvent{name: "test"})
}

func TestBaseSource_MultipleEmits(t *testing.T) {
	base := NewBaseSource(100)

	count := 50
	done := make(chan bool)

	// Emit multiple events concurrently
	go func() {
		for i := 0; i < count; i++ {
			base.Emit(testEvent{name: "event"})
		}
		done <- true
	}()

	// Receive all events
	for i := 0; i < count; i++ {
		<-base.Events()
	}

	<-done
}

// testEvent is a simple Event implementation for testing.
type testEvent struct {
	name string
}

func (e testEvent) String() string {
	return e.name
}
