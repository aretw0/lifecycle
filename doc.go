/*
Package lifecycle provides a robust control plane for modern Go applications.
It unifies Death Management (Signals, Shutdown) with Life Management (Supervision, Events).

# Core Concepts

The library is built around three pillars:

 1. Signal Context (The Foundation):
    Manages the application's lifecycle, handling OS signals (SIGINT/SIGTERM) and
    propagating cancellation via context.Context.

    ctx := lifecycle.NewSignalContext(context.Background())
    <-ctx.Done()

 2. Supervisor (The Bridge):
    Manages a tree of Workers (Processes, Containers, Goroutines), ensuring they
    adhere to the parent's lifecycle. It supports restart strategies (OneForOne,
    OneForAll) and persistent identity (Handover).

    sup := lifecycle.NewSupervisor("root", lifecycle.SupervisorStrategyOneForOne)
    sup.Add(spec)

 3. Control Plane (The Vision):
    Decouples "Events" (Triggers) from "Handlers" (Reactions) via a Router.
    This allows the application to react to external stimuli (Input, Webhooks) dynamically.

    router := lifecycle.NewRouter()
    router.Handle("Signal(interrupt)", handler)

# Root Package Aliases

For convenience, this package exposes the most commonly used types and constructors
from the internal `pkg/` structure, grouped by functionality:

  - Runtime:     lifecycle.Run (supports WithLogger, WithMetrics), lifecycle.Go, lifecycle.Do
  - Signals:     lifecycle.NewSignalContext, lifecycle.WithForceExit
  - Supervision: lifecycle.NewSupervisor, lifecycle.NewProcessWorker
  - Control:     lifecycle.NewRouter, lifecycle.Handle
  - Interactive: lifecycle.NewInteractiveRouter, lifecycle.WithShutdown

# Interactive Applications

For CLI tools, use the Interactive Router preset to wire up signals and input automatically:

	router := lifecycle.NewInteractiveRouter(suspendHandler, lifecycle.WithShutdown(quitFunc))
	lifecycle.Run(router)

# Configuration

Configuration is done via Functional Options (SignalOption) and Struct Specs (SupervisorSpec).
We adopt the "Stdlib Pattern": providing a `DefaultRouter` for convenience ("Managed Global State")
while allowing explicit `NewRouter` for strict isolation.
*/
package lifecycle
