package lifecycle

import (
	"context"
	"fmt"
	"os"

	"github.com/aretw0/lifecycle/pkg/events"
)

// InteractiveOption configures the interactive events.
type InteractiveOption func(*interactiveConfig)

type interactiveConfig struct {
	enableInput      bool
	enableSignal     bool
	commands         map[string]events.Handler
	shutdownFunc     func()
	defaultHandler   events.Handler
	interruptHandler events.Handler
	suspendHandler   events.Handler // Explicit suspend handler for automatic wiring
	unknownHandler   events.Handler // Handler for UnknownCommandEvent
	interruptEvent   events.Event
	inputOpts        []events.InputOption
}

// WithInput enables or disables the standard input source (stdin). Default is true.
func WithInput(enable bool) InteractiveOption {
	return func(c *interactiveConfig) {
		c.enableInput = enable
	}
}

// WithInputOptions passes generic InputOptions to the underlying InputSource.
// Use this to configure the reader, backoff, or other low-level settings.
func WithInputOptions(opts ...events.InputOption) InteractiveOption {
	return func(c *interactiveConfig) {
		c.inputOpts = append(c.inputOpts, opts...)
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

// WithDefaultHandler registers a handler for input that does not match any command.
// This enables "Passthrough" mode, where raw lines are routed to this handler
// via the "input/line" topic.
func WithDefaultHandler(handler events.Handler) InteractiveOption {
	return func(c *interactiveConfig) {
		c.defaultHandler = handler
	}
}

// WithUnknownHandler registers a handler for unknown commands.
// Use this to customize the visual feedback when a user types an invalid command.
func WithUnknownHandler(handler events.Handler) InteractiveOption {
	return func(c *interactiveConfig) {
		c.unknownHandler = handler
	}
}

// WithInterruptHandler configures the primary handler for the first interrupt signal (Ctrl+C).
// Use this for generic handlers that should run on "InterruptEvent" (e.g. clearing a line).
func WithInterruptHandler(handler events.Handler) InteractiveOption {
	return func(c *interactiveConfig) {
		c.interruptHandler = handler
		c.interruptEvent = events.InterruptEvent{}
	}
}

// WithSuspendOnInterrupt configures the handler to SUSPEND the application on the first interrupt signal.
// This preserves the "State Check" logic: if the handler is already suspended, the event is ignored.
// It explicitly sets the target event to SuspendEvent.
func WithSuspendOnInterrupt(h events.Handler) InteractiveOption {
	return func(c *interactiveConfig) {
		c.suspendHandler = h
		// Wrap with StateCheck if supported (The "Magic" is now explicit here)
		if sc, ok := h.(events.StateChecker); ok {
			c.interruptHandler = events.WithStateCheck(h, sc)
		} else {
			c.interruptHandler = h
		}
		c.interruptEvent = events.SuspendEvent{}
	}
}

// WithDefaultMappings enables the standard lifecycle command mappings (quit, suspend, etc.).
// By default, the InteractiveRouter starts with NO input mappings.
func WithDefaultMappings() InteractiveOption {
	return func(c *interactiveConfig) {
		c.inputOpts = append(c.inputOpts, events.WithDefaultMappings())
	}
}

// NewInteractiveRouter creates a router pre-configured for interactive CLI applications.
//
// It wires up:
//   - OS Signals (Interrupt/Term) -> Escalator (Interrupt first, then Quit)
//   - Input (Stdin) -> Router (reads lines as commands)
//   - Core Routes: "suspend", "resume" (if a SuspendHandler is provided)
//
// The router uses a "Double-Tap" strategy for interrupts (Ctrl+C):
//  1. First Signal: Triggers the configured InterruptHandler (or SuspendHandler).
//     For REPLs, this might clear the line. For services, this triggers suspension.
//  2. Second Signal: Forces the application to Quit.
//
// The quit handler is wrapped in events.Once() to ensure idempotency.
func NewInteractiveRouter(opts ...InteractiveOption) *events.Router {
	cfg := &interactiveConfig{
		enableInput:  true,
		enableSignal: true,
		commands:     make(map[string]events.Handler),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Default Intercept Handler (No-Op)
	if cfg.interruptHandler == nil {
		cfg.interruptHandler = events.HandlerFunc(func(ctx context.Context, e events.Event) error {
			return nil
		})
		cfg.interruptEvent = events.InterruptEvent{}
	}

	r := events.NewRouter()

	// 1. Core Lifecycle Routes (Suspend, Resume, Intercept)
	configureCoreRoutes(r, cfg)

	// 2. Custom Commands
	configureCommands(r, cfg)

	// 3. Quit & Signal Escalation Logic
	configureQuitLogic(r, cfg)

	// 4. Input & Signal Sources
	configureSources(r, cfg)

	return r
}

func configureCoreRoutes(r *events.Router, cfg *interactiveConfig) {
	// If a SuspendHandler was explicitly provided via WithSuspendOnInterrupt,
	// we automatically register the standard command routes for it.
	if cfg.suspendHandler != nil {
		h := cfg.suspendHandler
		r.Handle("lifecycle/suspend", h)
		r.Handle("command/suspend", h)
		r.Handle("lifecycle/resume", h)
		r.Handle("command/resume", h)
	} else if cfg.interruptHandler != nil {
		// If a generic InterruptHandler is provided, expose it via the "lifecycle/interrupt" topic.
		r.Handle("lifecycle/interrupt", cfg.interruptHandler)
	}
}

func configureCommands(r *events.Router, cfg *interactiveConfig) {
	for name, h := range cfg.commands {
		r.Handle("command/"+name, h)
	}

	// Default Handler (Passthrough Logic)
	if cfg.defaultHandler != nil {
		r.Handle("input/line", cfg.defaultHandler)
	}

	// Unknown Command Handler
	if cfg.unknownHandler != nil {
		r.Handle("input/unknown", cfg.unknownHandler)
	} else {
		// Default visual feedback for unknown commands
		r.Handle("input/unknown", events.HandlerFunc(func(ctx context.Context, e events.Event) error {
			if ue, ok := e.(events.UnknownCommandEvent); ok {
				fmt.Printf("Unknown command: %q. Try: %v\n", ue.Command, ue.Known)
			}
			return nil
		}))
	}
}

func configureQuitLogic(r *events.Router, cfg *interactiveConfig) {
	// We use an Escalator to implement the "Double-Tap" strategy:
	// 1st Signal: Intercept (Primary) -> e.g. Clear Line or Suspend
	// 2nd Signal: Quit (Fallback) -> Force Exit

	noOpQuit := events.HandlerFunc(func(ctx context.Context, e events.Event) error {
		return nil
	})

	// Resolve Quit Handler: User provided "WithShutdown" -> "WithCommand" -> No-Op
	var quitHandler events.Handler = noOpQuit

	if cfg.shutdownFunc != nil {
		quitHandler = events.NewShutdownFunc(cfg.shutdownFunc)
	}

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
	primaryHandler := events.WithFixedEvent(cfg.interruptHandler, cfg.interruptEvent)
	escalator := events.NewEscalator(primaryHandler, quitHandler)
	r.Handle("Signal(interrupt)", escalator)

	// Terminate Handler (Programmatic Exit)
	// Terminate needs a "Suspend" target. If we have one explicitly (cfg.suspendHandler), use it.
	var terminateTarget events.Handler = cfg.interruptHandler
	if cfg.suspendHandler != nil {
		terminateTarget = cfg.suspendHandler
	}

	terminateHandler := events.NewTerminate(terminateTarget, quitHandler)
	r.Handle("lifecycle/terminate", terminateHandler)
	r.Handle("command/terminate", terminateHandler)
	r.Handle("command/x", terminateHandler)
}

func configureSources(r *events.Router, cfg *interactiveConfig) {
	if cfg.enableSignal {
		r.AddSource(events.NewOSSignalSource(os.Interrupt))
	}
	if cfg.enableInput {
		var inputOpts []events.InputOption
		if len(cfg.inputOpts) > 0 {
			inputOpts = append(inputOpts, cfg.inputOpts...)
		}

		for name := range cfg.commands {
			inputOpts = append(inputOpts, events.WithInputMapping(name, events.InputEvent{Command: name}))
		}

		// Connect Passthrough if Default Handler is set
		if cfg.defaultHandler != nil {
			inputOpts = append(inputOpts, events.WithFallback(func(line string) events.Event {
				return events.LineEvent{Line: line}
			}))
		}

		r.AddSource(events.NewInputSource(inputOpts...))
	}
}
