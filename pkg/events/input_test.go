package events

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestInputSource(t *testing.T) {
	r, w, _ := os.Pipe()

	source := NewInputSource(WithInputReader(r))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := source.Events()

	go func() {
		_ = source.Start(ctx)
	}()

	// Test mapping: "suspend"
	_, _ = w.Write([]byte("suspend\n"))

	select {
	case ev := <-ch:
		if _, ok := ev.(SuspendEvent); !ok {
			t.Errorf("Expected SuspendEvent, got %T (%s)", ev, ev.String())
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for suspend event")
	}

	// Test unknown command
	unknownCalled := make(chan string, 1)
	source.unknownHandler = func(cmd string, known []string) {
		unknownCalled <- cmd
	}

	_, _ = w.Write([]byte("unknown\n"))
	select {
	case cmd := <-unknownCalled:
		if cmd != "unknown" {
			t.Errorf("Expected unknown command 'unknown', got '%s'", cmd)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for unknown handler call")
	}
}
