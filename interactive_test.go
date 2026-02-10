package lifecycle

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/events"
)

func TestInteractiveRouter_Passthrough(t *testing.T) {
	// Create a pipe to simulate stdin
	r := strings.NewReader("Hello\nquit\n")

	received := make(chan string, 1)

	// Handler to capture passthrough
	handler := events.HandlerFunc(func(ctx context.Context, e events.Event) error {
		if le, ok := e.(events.LineEvent); ok {
			received <- le.Line
		}
		return nil
	})

	router := NewInteractiveRouter(
		WithDefaultHandler(handler),
		WithDefaultMappings(), // Enable standard commands
		WithInputOptions(events.WithInputReader(r)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start router
	done := make(chan error)
	go func() {
		done <- router.Start(ctx)
	}()

	select {
	case line := <-received:
		if line != "Hello" {
			t.Errorf("Expected 'Hello', got '%s'", line)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for input")
	}

	cancel()
	<-done
}

func TestInteractiveRouter_UnknownCommand(t *testing.T) {
	// Create a pipe to simulate stdin
	r := strings.NewReader("weird_command\nquit\n")

	received := make(chan events.UnknownCommandEvent, 1)

	// Intercept the unknown command event to verify it was emitted
	unknownHandler := events.HandlerFunc(func(ctx context.Context, e events.Event) error {
		if ue, ok := e.(events.UnknownCommandEvent); ok {
			received <- ue
		}
		return nil
	})

	router := NewInteractiveRouter(
		WithDefaultMappings(), // Enable standard commands
		WithInputOptions(events.WithInputReader(r)),
	)
	// Register our verification handler for the topic
	router.Handle("input/unknown", unknownHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start router
	done := make(chan error)
	go func() {
		done <- router.Start(ctx)
	}()

	select {
	case ue := <-received:
		if ue.Command != "weird_command" {
			t.Errorf("Expected 'weird_command', got '%s'", ue.Command)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for unknown command event")
	}

	cancel()
	<-done
}
