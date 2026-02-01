# Planning: Lifecycle

## Roadmap

### v2.0: Control Plane (Event-Driven Lifecycle)

> [!NOTE]
> **Evolution Strategy**:
>
> - **v1.x**: Remains the stable, lightweight standard for **Death Management** (Shutdown & Signals). Ideal for simple CLIs.
> - **v2.x**: Introduces the **Control Plane** (Life Management). A superset for complex applications needing Event Routing, Hot Reloading, and Dynamic Reactions.

Focus: Transform `lifecycle` from a "Shutdown Manager" into a dynamic "Application Control Plane". Decouple "Events" (Triggers) from "Reactions" (Actions).

- [-] **Event Sources (Inputs)**: Generalized interface for things that trigger lifecycle changes.
  - [x] `OSSignalSource` (SIGINT, SIGTERM)
  - [x] `WebhookSource` (Admin HTTP endpoints - Skeleton)
  - [x] `HealthCheckSource` (Promoted from Backlog - Skeleton)
  - [x] `FileWatchSource` (Integration with Loam? - Skeleton)
  - [ ] **Progress Events**: `ctx.Progress(0.5)` or `lifecycle.Tick(ctx)` to drive UI/Loaders without coupling (Headless Timers).
- [ ] **Lifecycle Reactions (Outputs)**: Dynamic responses to events.
  - [x] `Shutdown` (Current behavior)
  - [ ] `HotReload` (Promoted from Backlog: SIGHUP triggers config reload without context cancellation)
  - [ ] `Suspend/Resume` (For Durable Execution)
- [x] **Managed Concurrency (The "lifecycle.Go" Pattern)**:
  - `lifecycle.Go(ctx, fn)`: A helper to spawn goroutines that are automatically tracked, waited on, and shielded from premature Context cancellation. Enforces "Safe Concurrency" by default.
- [ ] **Control Router (Stdlib-Native Mux)**:
  - **Strategy**: Clone `net/http` API (`Handle`, `Use`, `ServeMux`) but for `lifecycle.Event`. Avoid external deps.
  - [ ] **Pattern Matching**: Use `path.Match` (Glob) backend (`signal.*`) and support `regexp`.
  - [ ] **Middleware**: `Use(func(next Handler) Handler)` for Logging, Tracing, Recovery.
  - [ ] **Concurrency**: Keep synchronous default for safety, allow Async middleware.
  - [ ] **Introspection**: `Routes()` to list active bindings for visualization.
- [ ] **Ecosystem Integration (DX Layer)**:
  - **Visualization 2.0 (Overlay Pattern)**: Separate Topology (Static) from Status (Dynamic) to visualize missing/crashed nodes.
  - **Universal Introspection**: Public `Introspectable` interface (`State() any`) for generic adapters.
  - **Unified Observability**: Promote `pkg/metrics` as the standard bridge for Trellis/Loam.

## Backlog

- **Raw Mode Helpers**: Consider wrapping `x/term` Raw Mode enter/restore logic.
- **Parallel Hooks**: Research "Parallel Hooks with Dependency Mapping" for high-performance shutdown.
- **Supervisor Spec**: Allow defining per-child restart policies (Always, OnFailure, Never).
- **Circuit Breaker**: Implement "MaxRestarts within Duration" logic (Erlang style).
- **Priority Shutdown**: Explicit shutdown phases (e.g., "Critical", "Normal").

## Technical Debt

> [!NOTE]
> **Technical Debt**: Items here are not blockers but should be addressed in future releases.

- [ ] **Technical Debt Analysis**: Map out the full scope of the library and identify areas of technical debt so we can add items here.
