package sources

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/aretw0/lifecycle/pkg/control"
)

// InputSource reads commands from an io.Reader (like Stdin) and maps them to lifecycle Events.
// It handles the "Detach" pattern to ensure shutdown is not blocked by read operations.
type InputSource struct {
	r        io.Reader
	events   chan control.Event
	mappings map[string]control.Event
	detached bool
}

// InputOption configures the InputSource.
type InputOption func(*InputSource)

// WithInputReader sets the reader (default: os.Stdin).
func WithInputReader(r io.Reader) InputOption {
	return func(s *InputSource) {
		s.r = r
	}
}

// WithInputMapping adds a custom command mapping.
// Default mappings:
// "s", "suspend" -> SuspendEvent
// "r", "resume"  -> ResumeEvent
// "q", "quit"    -> InputEvent{Command: "quit"}
func WithInputMapping(key string, event control.Event) InputOption {
	return func(s *InputSource) {
		s.mappings[key] = event
	}
}

// NewInputSource creates a new source for interactive commands.
func NewInputSource(opts ...InputOption) *InputSource {
	s := &InputSource{
		r:      os.Stdin,
		events: make(chan control.Event),
		mappings: map[string]control.Event{
			"s":       control.SuspendEvent{},
			"suspend": control.SuspendEvent{},
			"r":       control.ResumeEvent{},
			"resume":  control.ResumeEvent{},
			// For quit, we don't have a standard event yet in pkg/control that implies "Shutdown"
			// other than maybe context cancellation, but Source emits events.
			// Users usually handle specific "quit" logic or we can define a ShutdownEvent.
			// For now let's emit a generic InputEvent compatible with the example.
			"q":    InputEvent{Command: "quit"},
			"quit": InputEvent{Command: "quit"},
		},
	}

	for _, opt := range opts {
		opt(s)
	}
	return s
}

// InputEvent is a generic input event for unmapped or custom commands.
type InputEvent struct {
	Command string
}

func (e InputEvent) String() string {
	return fmt.Sprintf("input/%s", e.Command)
}

func (s *InputSource) Events() <-chan control.Event { return s.events }

func (s *InputSource) Start(ctx context.Context) error {
	slog.Info("lifecycle: input source started", "mappings", len(s.mappings))

	// Create a Done channel for the detached goroutine to signal exit
	// We don't wait for it if context dies first, implementing "Leak but Exit" pattern for blocked IO.
	readerDone := make(chan struct{})

	go func() {
		defer close(s.events)
		defer close(readerDone)

		for {
			// Check context before trying to read
			if ctx.Err() != nil {
				return
			}

			// We use a simple buffer reading loop.
			// Ideally we'd use a proper termio reader but for "Simple Command Input"
			// standard string reading is often what users want.
			var cmd string
			_, err := fmt.Fscanln(s.r, &cmd)

			// If we were cancelled during the read, we just exit (and leak this blocked read if Reader doesn't support SetReadDeadline)
			if ctx.Err() != nil {
				return
			}

			if err != nil {
				if err == io.EOF {
					return
				}
				// Retry on temp error, but don't hot loop
				time.Sleep(100 * time.Millisecond)
				continue
			}

			event, ok := s.mappings[cmd]
			if !ok {
				// Default behavior: Emit as InputEvent? Or just log?
				// Let's log for now to match the example ergonomics
				fmt.Printf("Unknown command: %q. Try: [s]uspend, [r]esume, [q]uit\n", cmd)
				continue
			}

			select {
			case s.events <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return nil
}
