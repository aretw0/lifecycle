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
			"q":       control.ShutdownEvent{Reason: "manual"},
			"quit":    control.ShutdownEvent{Reason: "manual"},
			"exit":    control.ShutdownEvent{Reason: "manual"},
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
					// We tie the "Despair Threshold" to the Signal Context's ForceExit limit.
					threshold := 3 // Default fallback
					unsafe := false

					// Check Context State via structural interface (avoid circular imports)
					if sc, ok := ctx.(interface {
						IsUnsafe() bool
						ForceExitThreshold() int
					}); ok {
						unsafe = sc.IsUnsafe()
						if !unsafe {
							// We allow exactly as many "fake EOFs" as the user allowed signals.
							// This ensures that if they configured ForceExit(5), the input
							// source doesn't die on the 4th Ctrl+C.
							threshold = sc.ForceExitThreshold()
						}
					}

					eofCount++

					// desisted if we exceeded the threshold and we are not in unsafe mode
					if eofCount > threshold && !unsafe {
						slog.Debug("input source: persistent EOF received, stopping", "attempts", eofCount, "limit", threshold)
						return
					}

					slog.Debug("input source: transient EOF (Ctrl+C?), retrying...",
						"attempt", eofCount,
						"limit", func() string {
							if unsafe {
								return "inf"
							}
							return fmt.Sprintf("%d", threshold)
						}())

					// If we are still running, it means the context didn't cancel on Ctrl+C.
					// This indicates a REPL-like behavior where we should clear the line.
					select {
					case s.events <- control.ClearLineEvent{}:
					default:
					}

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
