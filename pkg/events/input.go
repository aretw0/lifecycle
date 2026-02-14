package events

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/procio/scan"
	"github.com/aretw0/procio/termio"
)

// InputSource reads commands from an io.Reader (like Stdin) and maps them to lifecycle
// It handles the "Detach" pattern to ensure shutdown is not blocked by read operations.
type InputSource struct {
	BaseSource
	r        io.Reader
	mappings map[string]Event
	fallback func(cmd string) Event
	detached bool
	backoff  time.Duration
	bufSize  int
}

// InputOption configures the InputSource.
type InputOption func(*InputSource)

// WithInputReader sets the reader (default: os.Stdin).
func WithInputReader(r io.Reader) InputOption {
	return func(s *InputSource) {
		s.r = r
	}
}

// WithInputBackoff configures the duration to wait before retrying interruptions or errors.
// Default: 100ms.
func WithInputBackoff(d time.Duration) InputOption {
	return func(s *InputSource) {
		s.backoff = d
	}
}

// WithInputBufferSize sets the size of the internal read buffer.
// Default: 1024 bytes.
func WithInputBufferSize(size int) InputOption {
	return func(s *InputSource) {
		if size <= 0 {
			size = 1024
		}
		s.bufSize = size
	}
}

// WithInputEventBuffer sets the size of the event channel buffer.
// Default: 10.
func WithInputEventBuffer(size int) InputOption {
	return func(s *InputSource) {
		if size < 0 {
			size = 0
		}
		// We recreate the BaseSource to apply the new buffer size
		s.BaseSource = NewBaseSource("input", size)
	}
}

// WithInputMapping adds a custom command mapping.
// Default mappings:
// "s", "suspend" -> SuspendEvent
// "r", "resume"  -> ResumeEvent
// "q", "quit"    -> InputEvent{Command: "quit"}
func WithInputMapping(key string, event Event) InputOption {
	return func(s *InputSource) {
		s.mappings[key] = event
	}
}

// WithInputMappings adds multiple command mappings at once.
func WithInputMappings(mappings map[string]Event) InputOption {
	return func(s *InputSource) {
		for k, v := range mappings {
			s.mappings[k] = v
		}
	}
}

// WithInputCommands is a low-level helper to allowlist simple commands.
// It maps each string "cmd" to InputEvent{Command: "cmd"}.
// Use this if you want to define valid inputs without defining handlers here.
func WithInputCommands(commands ...string) InputOption {
	return func(s *InputSource) {
		for _, cmd := range commands {
			s.mappings[cmd] = InputEvent{Command: cmd}
		}
	}
}

// WithInputHandlers is a high-level helper to synchronize InputSource with Router.
// It extracts the keys from the handler map and allowlists them as valid commands.
// This ensures that any command you have a handler for is also a valid input.
func WithInputHandlers(handlers map[string]Handler) InputOption {
	return func(s *InputSource) {
		for cmd := range handlers {
			s.mappings[cmd] = InputEvent{Command: cmd}
		}
	}
}

// WithRawInput configures the InputSource for "Data-Only" mode.
// It clears default mappings and sets a Fallback to capture everything.
func WithRawInput(handler func(line string)) InputOption {
	return func(s *InputSource) {
		s.mappings = make(map[string]Event) // Clear mappings
		s.fallback = func(cmd string) Event {
			handler(cmd)
			// Return a no-op event to satisfy the interface, or we could define a HandledEvent.
			// But since the handler is called directly here (legacy support), we just emit a LineEvent
			// for consistency, though the user handler has already run.
			return LineEvent{Line: cmd}
		}
	}
}

// WithFallback configures a factory to generate events for unknown commands.
// If set, this takes precedence over the default UnknownCommandEvent.
func WithFallback(factory func(line string) Event) InputOption {
	return func(s *InputSource) {
		s.fallback = factory
	}
}

// NewInputSource creates a new source for interactive commands.
func NewInputSource(opts ...InputOption) *InputSource {
	// Default to a smart terminal reader if possible
	reader, err := termio.Open()
	if err != nil {
		log.Debug("failed to open terminal", "error", err)
		reader = os.Stdin
	}

	s := &InputSource{
		BaseSource: NewBaseSource("input", 10),
		r:          reader,
		backoff:    100 * time.Millisecond,
		mappings:   make(map[string]Event),
		bufSize:    1024,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// WithDefaultMappings adds the standard lifecycle command mappings:
// suspend, resume, q, quit, exit, x, terminate.
func WithDefaultMappings() InputOption {
	return func(s *InputSource) {
		defaults := map[string]Event{
			"suspend":   SuspendEvent{},
			"s":         SuspendEvent{},
			"resume":    ResumeEvent{},
			"r":         ResumeEvent{},
			"q":         ShutdownEvent{Reason: "manual"},
			"quit":      ShutdownEvent{Reason: "manual"},
			"exit":      ShutdownEvent{Reason: "manual"},
			"x":         TerminateEvent{},
			"terminate": TerminateEvent{},
		}
		for k, v := range defaults {
			s.mappings[k] = v
		}
	}
}

// InputEvent is a generic input event for unmapped or custom commands.
type InputEvent struct {
	Command string
}

func (e InputEvent) String() string {
	return fmt.Sprintf("command/%s", e.Command)
}

// LineEvent represents raw text input that didn't match a command.
// Topic: "input/line"
type LineEvent struct {
	Line string
}

func (e LineEvent) String() string {
	return "input/line"
}

// UnknownCommandEvent is emitted when a command is not found in the mappings
// and no fallback is configured.
// Topic: "input/unknown"
type UnknownCommandEvent struct {
	Command string
	Known   []string
}

func (e UnknownCommandEvent) String() string {
	return "input/unknown"
}

func (s *InputSource) Start(ctx context.Context) error {
	log.Debug("lifecycle: input source started", "mappings", len(s.mappings))

	// Create a Done channel for the detached goroutine to signal exit
	readerDone := make(chan struct{})

	go func() {
		defer s.Close()
		defer close(readerDone)

		// Dynamic Configuration based on Context (SignalContext awareness)
		threshold := 3
		unsafe := false
		if sc, ok := ctx.(interface {
			IsUnsafe() bool
			ForceExitThreshold() int
		}); ok {
			unsafe = sc.IsUnsafe()
			if !unsafe {
				threshold = sc.ForceExitThreshold()
			}
		}

		// Configure the robust scanner from procio
		scanner := scan.NewScanner(s.r,
			scan.WithBufferSize(s.bufSize),
			scan.WithBackoff(s.backoff),
			scan.WithThreshold(threshold),
			scan.WithUnsafeMode(unsafe),
			scan.WithLineHandler(func(line string) {
				cmd := strings.TrimSpace(line)
				if cmd == "" {
					_ = s.Emit(ctx, LineEvent{Line: ""})
					return
				}
				// processCommand returns true if we should stop (e.g. context done)
				// But Scanner doesn't support early exit via callback yet,
				// so we rely on context cancellation which processCommand checks.
				s.processCommand(ctx, cmd)
			}),
			scan.WithClearHandler(func() {
				_ = s.Emit(ctx, ClearLineEvent{})
			}),
			scan.WithErrorHandler(func(err error) {
				log.Debug("input source: read error (retrying)", "error", err)
			}),
		)

		scanner.Start(ctx)
	}()

	// Wait for context cancellation or reader completion
	select {
	case <-ctx.Done():
		return nil
	case <-readerDone:
		return nil
	}
}

func (s *InputSource) processCommand(ctx context.Context, cmd string) bool {
	event, ok := s.mappings[cmd]
	if !ok {
		// Fallback for unknown commands (e.g. Passthrough)
		if s.fallback != nil {
			event = s.fallback(cmd)
		} else {
			// Unknown Handler (Event-based Error Reporting)
			// Generate sorted known commands
			known := make([]string, 0, len(s.mappings))
			for k := range s.mappings {
				known = append(known, k)
			}
			sort.Strings(known)

			event = UnknownCommandEvent{
				Command: cmd,
				Known:   known,
			}
		}
	}

	select {
	case <-ctx.Done():
		return true
	default:
		if err := s.Emit(ctx, event); err != nil {
			return true
		}
		return false
	}
}
