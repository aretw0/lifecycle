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
	"github.com/aretw0/lifecycle/pkg/core/termio"
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
	buffer := make([]byte, s.bufSize)
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
	log.Debug("input source: read error (retrying)", "error", err)
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
		log.Debug("input source: persistent EOF received, stopping", "attempts", *eofCount, "limit", threshold)
		return true
	}

	log.Debug("input source: transient interrupt, retrying...",
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
		if b == '\r' {
			continue // Ignore Carriage Return (CRLF handling)
		}
		if b == '\n' {
			// Line complete
			cmd := strings.TrimSpace(lineBuilder.String())
			lineBuilder.Reset()

			if cmd == "" {
				_ = s.Emit(ctx, LineEvent{Line: ""})
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
