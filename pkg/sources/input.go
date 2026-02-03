package sources

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/aretw0/lifecycle/pkg/control"
	"github.com/aretw0/lifecycle/pkg/termio"
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
	// Default to a smart terminal reader if possible
	reader, err := termio.Open()
	if err != nil {
		slog.Debug("failed to open terminal", "error", err)
		reader = os.Stdin
	}

	s := &InputSource{
		r:      reader,
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

		// Use a manual read loop for maximum robustness against random interrupts (Ctrl+C on Windows)
		// bufio.Scanner is "sticky" on errors/EOF, which is bad if valid input comes after an interrupt.
		buffer := make([]byte, 1024)
		var lineBuilder strings.Builder
		eofCount := 0

		for {
			// Check context
			if ctx.Err() != nil {
				return
			}

			n, err := s.r.Read(buffer)

			// Handle Context Cancellation (Priority)
			if ctx.Err() != nil {
				return
			}

			if err != nil {
				if err == io.EOF {
					// "Fake EOF" Protection:
					// On Windows, Ctrl+C can cause a transient EOF on the read syscall.
					// We verify if it is persistent.
					eofCount++
					if eofCount > 3 {
						slog.Debug("input source: persistent EOF received, stopping")
						return
					}
					slog.Debug("input source: transient EOF (Ctrl+C?), retrying...", "attempt", eofCount)
					time.Sleep(100 * time.Millisecond)
					continue
				}
				// Other errors: Log and retry with backoff
				slog.Debug("input source: read error (retrying)", "error", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Successful read: Reset EOF counter
			eofCount = 0

			// Process read bytes
			chunk := buffer[:n]
			for _, b := range chunk {
				if b == '\n' || b == '\r' {
					// Line complete
					cmd := strings.TrimSpace(lineBuilder.String())
					lineBuilder.Reset()

					if cmd == "" {
						continue
					}

					event, ok := s.mappings[cmd]
					if !ok {
						fmt.Printf("Unknown command: %q. Try: [s]uspend, [r]esume, [q]uit\n", cmd)
						continue
					}

					select {
					case s.events <- event:
					case <-ctx.Done():
						return
					}
				} else {
					lineBuilder.WriteByte(b)
				}
			}
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return nil
}
