package sources

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aretw0/lifecycle/pkg/control"
	"github.com/aretw0/lifecycle/pkg/termio"
)

// InputSource reads commands from an io.Reader (like Stdin) and maps them to lifecycle Events.
// It handles the "Detach" pattern to ensure shutdown is not blocked by read operations.
type InputSource struct {
	control.BaseSource
	r              io.Reader
	mappings       map[string]control.Event
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
func WithInputMapping(key string, event control.Event) InputOption {
	return func(s *InputSource) {
		s.mappings[key] = event
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
		BaseSource: control.NewBaseSource(10),
		r:          reader,
		backoff:    100 * time.Millisecond,
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
	slog.Info("lifecycle: input source started", "mappings", len(s.mappings))

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
			if shouldStop := s.handleReadError(ctx, err, &eofCount); shouldStop {
				return
			}
			continue
		}

		// Successful read: Reset EOF counter
		eofCount = 0
		s.processChunk(ctx, buffer[:n], &lineBuilder)
	}
}

func (s *InputSource) handleReadError(ctx context.Context, err error, eofCount *int) bool {
	if err == io.EOF {
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

	// desisted if we exceeded the threshold and we are not in unsafe mode
	if *eofCount > threshold && !unsafe {
		slog.Debug("input source: persistent EOF received, stopping", "attempts", *eofCount, "limit", threshold)
		return true
	}

	slog.Debug("input source: transient EOF (Ctrl+C?), retrying...",
		"attempt", *eofCount,
		"limit", func() string {
			if unsafe {
				return "inf"
			}
			return fmt.Sprintf("%d", threshold)
		}())

	// If we are still running, it means the context didn't cancel on Ctrl+C.
	// This indicates a REPL-like behavior where we should clear the line.
	s.Emit(control.ClearLineEvent{})

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
		s.Emit(event)
		return false
	}
}
