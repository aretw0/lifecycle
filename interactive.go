package lifecycle

import (
	"context"
	"os"

	"github.com/aretw0/lifecycle/pkg/events"
)

// InteractiveOption configures the interactive events.
type InteractiveOption func(*interactiveConfig)

type interactiveConfig struct {
	enableInput  bool
	enableSignal bool
	commands     map[string]events.Handler
	shutdownFunc func()
}

// WithInput enables or disables the standard input source (stdin). Default is true.
func WithInput(enable bool) InteractiveOption {
	return func(c *interactiveConfig) {
		c.enableInput = enable
	}
}

// WithSignal enables or disables the OS signal source (Interrupt, Term). Default is true.
func WithSignal(enable bool) InteractiveOption {
	return func(c *interactiveConfig) {
		c.enableSignal = enable
	}
}

// WithCommand adds a custom command handler.
// Example: WithCommand("stats", statsHandler) will route "command/stats" to statsHandler.
func WithCommand(name string, handler events.Handler) InteractiveOption {
	return func(c *interactiveConfig) {
		c.commands[name] = handler
	}
}

// WithShutdown registers a function to be called when a "quit" or "shutdown" event occurs.
// This simplifies wiring up the main loop exit mechanism (e.g. closing a quit channel).
func WithShutdown(fn func()) InteractiveOption {
	return func(c *interactiveConfig) {
		c.shutdownFunc = fn
	}
}

// NewInteractiveRouter creates a router pre-configured for interactive CLI applications.
//
// It wires up:
//   - OS Signals (Interrupt/Term) -> Escalator (Intercept first, then Quit)
//   - Input (Stdin) -> Router (reads lines as commands)
//   - Commands: "suspend", "resume" -> SuspendHandler
//
// The 'interceptHandler' is the primary action taken on the first Signal(interrupt).
// For REPLs, this might be a simple line-clearer. For durable services, it might
// be a full SuspendHandler.
//
// The quit handler is automatically wrapped in events.Once() to ensure that
// user-provided shutdown logic is executed exactly once, even if multiple
// shutdown events (e.g. SIGINT followed by typing 'q') are received.
func NewInteractiveRouter(interceptHandler events.Handler, opts ...InteractiveOption) *events.Router {
	cfg := &interactiveConfig{
		enableInput:  true,
		enableSignal: true,
		commands:     make(map[string]events.Handler),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	r := events.NewRouter()

	// 1. Standard Routes
	if sh, ok := interceptHandler.(events.SuspendableHandler); ok {
		r.Handle("lifecycle/suspend", sh)
		r.Handle("lifecycle/intercept", sh)
		r.Handle("command/suspend", sh)
		r.Handle("lifecycle/resume", sh)
		r.Handle("command/resume", sh)
	} else {
		// Generic Intercept Handler
		r.Handle("lifecycle/intercept", interceptHandler)
	}

	// 2. Custom Commands
	for name, h := range cfg.commands {
		r.Handle("command/"+name, h)
	}

	// 3. Smart Signal Handling (via Escalator)
	// We use an Escalator to implement the "Double-Tap" strategy:
	// 1st Signal: Intercept (Primary) -> e.g. Clear Line or Suspend
	// 2nd Signal: Quit (Fallback) -> Force Exit

	// We provide a no-op quit handler here because the actual "Exit" is often handled by the
	// runtime observing the signal cancellation propagation or by the user hitting Ctrl+C twice (Force Exit).
	noOpQuit := events.HandlerFunc(func(ctx context.Context, e events.Event) error {
		return nil
	})

	// Resolve Quit Handler: User provided "WithShutdown" -> "WithCommand" -> No-Op
	var quitHandler events.Handler = noOpQuit

	if cfg.shutdownFunc != nil {
		quitHandler = events.NewShutdownFunc(cfg.shutdownFunc)
	}

	// 4. Resolve Quit Logic & Escalation
	// Precedence:
	// 1. Explicit WithCommand("quit", ...)
	// 2. WithShutdown(...) convenience helper
	// 3. Default No-Op (relies on Signal Force Exit)

	if h, ok := cfg.commands["quit"]; ok {
		quitHandler = h
	}

	// Route "lifecycle/shutdown" (emitted by 'q'/'quit' in InputSource) to the resolved quit handler
	r.Handle("lifecycle/shutdown", quitHandler)
	// Route "command/quit" if not already set (if using WithShutdown)
	if _, ok := cfg.commands["quit"]; !ok {
		r.Handle("command/quit", quitHandler)
	}

	// Construct Escalator
	// We wrap the intercept handler in a StateCheck middleware if it implements StateChecker.
	// This preserves the legacy behavior where StateChecker implied "Check before Intercept".
	var primaryHandler events.Handler = interceptHandler

	// Event Selection Logic:
	// If StateChecker -> SuspendEvent (default)
	// If !StateChecker -> InterceptEvent (interactive)
	targetEvent := events.Event(events.SuspendEvent{})
	if _, ok := interceptHandler.(events.StateChecker); !ok {
		targetEvent = events.InterceptEvent{}
	} else {
		// If StateChecker is present, we wrap with StateCheck middleware
		// to skip Intercept if already Active.
		primaryHandler = events.WithStateCheck(interceptHandler, interceptHandler.(events.StateChecker))
	}

	// Apply Event Transformation (Signal -> Suspend/Intercept)
	primaryHandler = events.WithFixedEvent(primaryHandler, targetEvent)

	// Use generic Escalator: Primary -> Quit
	escalator := events.NewEscalator(primaryHandler, quitHandler)
	r.Handle("Signal(interrupt)", escalator)

	// 5. High-level Facilitators (Composition)
	terminateHandler := events.NewTerminate(interceptHandler, quitHandler)
	r.Handle("lifecycle/terminate", terminateHandler)
	r.Handle("command/terminate", terminateHandler)
	r.Handle("command/x", terminateHandler)

	// 6. Sources
	if cfg.enableSignal {
		r.AddSource(events.NewOSSignalSource(os.Interrupt))
	}
	if cfg.enableInput {
		var inputOpts []events.InputOption
		for name := range cfg.commands {
			inputOpts = append(inputOpts, events.WithInputMapping(name, events.InputEvent{Command: name}))
		}
		r.AddSource(events.NewInputSource(inputOpts...))
	}

	return r
}
