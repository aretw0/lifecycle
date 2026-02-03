/*
Package lifecycle provides a robust control plane for modern Go applications.
It unifies Death Management (Signals, Shutdown) with Life Management (Supervision, Events).

# Core Concepts

The library is built around three pillars:

 1. Signal Context (The Foundation):
    manages the application's lifecycle, handling OS signals (SIGINT/SIGTERM) and
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
    Decouples "Events" (Triggers) from "Reactions" (Actions) via a Router.
    This enables dynamic behaviors like Hot Reload, Configuration Updates, etc.

    router := lifecycle.NewRouter()
    router.On("Signal(interrupt)", func(ctx context.Context) error { ... })

# Root Package Aliases

For convenience, this package exposes the most commonly used types and constructors
from the internal `pkg/` structure.

  - lifecycle.NewSignalContext -> pkg/signal.NewContext
  - lifecycle.NewGroup         -> group.NewGroup (Root Primitive)
  - lifecycle.NewSupervisor    -> pkg/supervisor.New
  - lifecycle.NewRouter        -> pkg/control.NewRouter
  - lifecycle.Receive          -> pkg/runtime.Receive (Safe Iterator)

# Configuration

Configuration is done via Functional Options (SignalOption) and Struct Specs (SupervisorSpec).
We adopt the "Stdlib Pattern": providing a `DefaultRouter` for convenience ("Managed Global State") while allowing explicit `NewRouter` for strict isolation.
*/
package lifecycle
