package lifecycle

import (
	"context"
	"os"

	"github.com/aretw0/lifecycle/pkg/control"
	"github.com/aretw0/lifecycle/pkg/handlers"
	"github.com/aretw0/lifecycle/pkg/sources"
)

// InteractiveOption configures the interactive router.
type InteractiveOption func(*interactiveConfig)

type interactiveConfig struct {
	enableInput  bool
	enableSignal bool
	commands     map[string]control.Handler
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
func WithCommand(name string, handler control.Handler) InteractiveOption {
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
//   - OS Signals (Interrupt/Term) -> SmartSignalHandler (Suspend first, then Quit)
//   - Input (Stdin) -> Router (reads lines as commands)
//   - Commands: "suspend", "resume" -> SuspendHandler
//
// To handle "Quit", it routes "command/quit" to a generic no-op handler.
// A true shutdown is triggered by the runtime observing the signal context or by a custom
// handler that cancels a context.
func NewInteractiveRouter(suspendHandler *handlers.SuspendHandler, opts ...InteractiveOption) *control.Router {
	cfg := &interactiveConfig{
		enableInput:  true,
		enableSignal: true,
		commands:     make(map[string]control.Handler),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	r := control.NewRouter()

	// 1. Standard Routes
	r.Handle("lifecycle/suspend", suspendHandler)
	r.Handle("command/suspend", suspendHandler)

	r.Handle("lifecycle/resume", suspendHandler)
	r.Handle("command/resume", suspendHandler)

	// 2. Custom Commands
	for name, h := range cfg.commands {
		r.Handle("command/"+name, h)
	}

	// 3. Smart Signal Handling
	// The SmartSignalHandler intercepts SIGINT.
	// If the system is running, it triggers a Suspend via suspendHandler.
	// If/When it decides to Quit (e.g. system is already suspended), it delegates to the next handler.
	// We provide a no-op handler here because the actual "Exit" is often handled by the
	// runtime observing the signal cancellation propagation or by the user hitting Ctrl+C twice (Force Exit).
	noOpQuit := control.HandlerFunc(func(ctx context.Context, e control.Event) error {
		return nil
	})

	// Resolve Quit Handler: User provided "WithShutdown" -> "WithCommand" -> No-Op
	var quitHandler control.Handler = noOpQuit

	if cfg.shutdownFunc != nil {
		quitHandler = control.HandlerFunc(func(ctx context.Context, e control.Event) error {
			cfg.shutdownFunc()
			return nil
		})
	}

	// 4. Resolve Quit Logic
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

	smartHandler := handlers.NewSmartSignalHandler(suspendHandler, quitHandler)
	r.Handle("Signal(interrupt)", smartHandler)

	// 5. Sources
	if cfg.enableSignal {
		r.AddSource(sources.NewOSSignalSource(os.Interrupt))
	}
	if cfg.enableInput {
		r.AddSource(sources.NewInputSource())
	}

	return r
}
