// Package control implements the Event-Driven Control Plane for the lifecycle library.
//
// It defines the core interfaces (Event, Source, Handler) and the Router component
// that decouples event production from event consumption.
//
// The Control Plane allows applications to react dynamically to system stimuli
// (signals, webhooks, file changes) rather than just shutting down.
//
// # Introspection
//
// Components implementing the Introspectable interface accept a State() method
// that returns a serializable view of their internal state. The Router exposes
// its topology (routes, middlewares, sources) this way.
//
//	router := control.NewRouter()
//	state := router.State() // Returns RouterState DTO
//
// # Interactive Applications
//
// For CLI tools, the package offers NewInteractiveRouter (moved to root package as a helper)
// which pre-wires InputSource (Stdin) and SmartSignalHandler.
//
// # Suspend and Resume
//
// The package supports "Durable Execution" semantics via SuspendEvent and ResumeEvent.
// Handlers can register hooks to persist state or pause processing.
//
//	handler := handlers.NewSuspendHandler()
//	handler.OnSuspend(func(ctx context.Context) error {
//	    log.Println("Persisting checkpoints...")
//	    return nil
//	})
package control
