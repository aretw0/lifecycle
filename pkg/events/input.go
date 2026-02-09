package events

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/termio"
)

// InputSource reads commands from an io.Reader (like Stdin) and maps them to lifecycle
// It handles the "Detach" pattern to ensure shutdown is not blocked by read operations.
type InputSource struct {
	BaseSource
	r              io.Reader
	mappings       map[string]Event
	unknownHandler func(cmd string, known []string)
	detached       bool
	backoff        time.Duration
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

// WithRawInput configures the InputSource for "Data-Only" mode.
// It clears default mappings and sets the "Unknown Handler" to the provided function,
// effectively treating every line as a data payload.
func WithRawInput(handler func(line string)) InputOption {
	return func(s *InputSource) {
		s.mappings = make(map[string]Event) // Clear mappings
		s.unknownHandler = func(cmd string, _ []string) {
			handler(cmd)
		}
	}
}

// WithUnknownHandler configures a custom handler for unknown commands.
// The handler receives the unknown command and a sorted list of known commands.
func WithUnknownHandler(fn func(cmd string, known []string)) InputOption {
	return func(s *InputSource) {
		s.unknownHandler = fn
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
		BaseSource: NewBaseSource("input", 10),
		r:          reader,
		backoff:    100 * time.Millisecond,
		mappings: map[string]Event{
			"s":         SuspendEvent{},
			"suspend":   SuspendEvent{},
			"r":         ResumeEvent{},
			"resume":    ResumeEvent{},
			"q":         ShutdownEvent{Reason: "manual"},
			"quit":      ShutdownEvent{Reason: "manual"},
			"exit":      ShutdownEvent{Reason: "manual"},
			"x":         TerminateEvent{},
			"terminate": TerminateEvent{},
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	// Default handler if none provided
	if s.unknownHandler == nil {
		s.unknownHandler = func(cmd string, known []string) {
			fmt.Printf("Unknown command: %q. Try: %v\n", cmd, known)
		}
	}

	return s
}

// InputEvent is a generic input event for unmapped or custom commands.
type InputEvent struct {
	Command string
}

func (e InputEvent) String() string {
	return fmt.Sprintf("command/%s", e.Command)
}

func (s *InputSource) Start(ctx context.Context) error {
	slog.Debug("lifecycle: input source started", "mappings", len(s.mappings))

	// Create a Done channel for the detached goroutine to signal exit
	// We don't wait for it if context dies first, implementing "Leak but Exit" pattern for blocked IO.
	readerDone := make(chan struct{})

	go func() {
		defer s.Close()
		defer close(readerDone)
		s.readLoop(ctx)
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return nil
}

func (s *InputSource) readLoop(ctx context.Context) {
	// Use a manual read loop for maximum robustness against random interrupts (Ctrl+C on Windows)
	// bufio.Scanner is "sticky" on errors/EOF, which is bad if valid input comes after an interrupt.
	buffer := make([]byte, 1024)
	var lineBuilder strings.Builder
	eofCount := 0

	for {
		if ctx.Err() != nil {
			return
		}

		n, err := s.r.Read(buffer)

		// Handle Context Cancellation (Priority)
		if ctx.Err() != nil {
			return
		}

		if err != nil {
			if shouldStop := s.handleReadError(ctx, err, &eofCount, &lineBuilder); shouldStop {
				return
			}
			continue
		}

		// Successful read: Reset EOF counter
		eofCount = 0
		s.processChunk(ctx, buffer[:n], &lineBuilder)
	}
}

func (s *InputSource) handleReadError(ctx context.Context, err error, eofCount *int, lineBuilder *strings.Builder) bool {
	if termio.IsInterrupted(err) {
		// On Windows, Ctrl+C can cause a transient EOF or another interrupted error.
		// We treat these as Interruptions.
		lineBuilder.Reset()
		_ = s.Emit(ctx, ClearLineEvent{})
		return s.handleEOF(ctx, eofCount)
	}

	// Other errors: Log and retry with backoff
	slog.Debug("input source: read error (retrying)", "error", err)
	time.Sleep(s.backoff)
	return false
}

func (s *InputSource) handleEOF(ctx context.Context, eofCount *int) bool {
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
			threshold = sc.ForceExitThreshold()
		}
	}

	*eofCount++

	// threshold exceeded: stop the source
	if *eofCount > threshold && !unsafe {
		slog.Debug("input source: persistent EOF received, stopping", "attempts", *eofCount, "limit", threshold)
		return true
	}

	slog.Debug("input source: transient interrupt, retrying...",
		"attempt", *eofCount,
		"limit", func() string {
			if unsafe {
				return "inf"
			}
			return fmt.Sprintf("%d", threshold)
		}())

	time.Sleep(s.backoff)
	return false
}

func (s *InputSource) processChunk(ctx context.Context, chunk []byte, lineBuilder *strings.Builder) {
	for _, b := range chunk {
		if b == '\n' || b == '\r' {
			// Line complete
			cmd := strings.TrimSpace(lineBuilder.String())
			lineBuilder.Reset()

			if cmd == "" {
				continue
			}

			if stop := s.processCommand(ctx, cmd); stop {
				return
			}
		} else {
			lineBuilder.WriteByte(b)
		}
	}
}

func (s *InputSource) processCommand(ctx context.Context, cmd string) bool {
	event, ok := s.mappings[cmd]
	if !ok {
		// Generate sorted known commands
		known := make([]string, 0, len(s.mappings))
		for k := range s.mappings {
			known = append(known, k)
		}
		sort.Strings(known)

		s.unknownHandler(cmd, known)
		return false
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
