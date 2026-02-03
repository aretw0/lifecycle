# Planning: Lifecycle

## Roadmap

### v2.0: Control Plane (Event-Driven Lifecycle)

> [!NOTE]
> **Evolution Strategy**:
>
> - **v1.x**: Remains the stable, lightweight standard for **Death Management** (Shutdown & Signals). Ideal for simple CLIs.
> - **v2.x**: Introduces the **Control Plane** (Life Management). A superset for complex applications needing Event Routing, Hot Reloading, and Dynamic Reactions.

Focus: Transform `lifecycle` from a "Shutdown Manager" into a dynamic "Application Control Plane". Decouple "Events" (Triggers) from "Handlers" (Actions).

- [x] **Event Sources (Inputs)**: Generalized interface for things that trigger lifecycle changes.
  - [x] `OSSignalSource` (SIGINT, SIGTERM)
  - [x] `WebhookSource` (Admin HTTP endpoints - Skeleton)
  - [x] `HealthCheckSource` (Promoted from Backlog - Skeleton)
  - [x] `FileWatchSource` (Integration with Loam? - Skeleton)
  - [x] **Progress Events**: `ctx.Progress(0.5)` or `lifecycle.Tick(ctx)` to drive UI/Loaders without coupling (Headless Timers).
- [-] **Lifecycle Handlers (Outputs)**: Dynamic responses to events.
  - [x] `Shutdown` (Current behavior)
  - [x] `HotReload` (Promoted from Backlog: SIGHUP triggers config reload without context cancellation)
  - [x] `Suspend/Resume` (Verified with `examples/suspend`)
- [x] **Managed Concurrency (The "lifecycle.Go" Pattern)**:
  - `lifecycle.Go(ctx, fn)`: A helper to spawn goroutines that are automatically tracked, waited on, and shielded from premature Context cancellation. Enforces "Safe Concurrency" by default.
  - *Strategy*: Use a **Global WaitGroup** inside `pkg/runtime`. `Run` will `Wait()` on it after `Runnable` exits but before returning, guaranteeing no leaks. Panics in these goroutines should be caught and logged (metrics), but not crash the app (Recovery). `Supervisor` is reserved for "Services" that need restarts; `lifecycle.Go` is for "Tasks" that need cleanup.
- [x] **Control Router (Stdlib-Native Mux)**:
  - **Strategy**: Clone `net/http` API (`Handle`, `Use`, `ServeMux`) but for `lifecycle.Event`. Avoid external deps.
  - [x] **Pattern Matching**: Use `path.Match` (Glob) backend (`signal.*`) and support `regexp` (partial).
  - [x] **Middleware**: `Use(func(next Handler) Handler)` for Logging, Tracing, Recovery.
  - [x] **Concurrency**: Keep synchronous default for safety.
  - [x] **Introspection**: `Routes()` to list active bindings for visualization.
- [ ] **Ecosystem Integration (DX Layer)**:
  - **Visualization 2.0 (Overlay Pattern)**: Separate Topology (Static) from Status (Dynamic) to visualize everything, including missing/crashed nodes.
  - **Universal Introspection**: Public `Introspectable` interface (`State() any`) for generic adapters.
  - **Unified Observability**: Promote `pkg/metrics` as the standard bridge for Trellis/Loam.
- [x] Ergonomics & Interactions (The "Human" Layer)
  - **Focus**: Reduce boilerplate for interactive applications and robust worker patterns.
  - **Interactive Sources**:
    - [x] `InputSource`: Standardize CLI command inputs (`q`, `s`, `r`) with detached readers to avoid blocking shutdown. Tied to `ForceExit` threshold for resilience on Windows.
    - [x] **Interactive Router**: `NewInteractiveRouter` preset to wire up signals, input, and generic commands (suspend/resume/quit).
  - **Worker Pattern Helpers**:
    - [ ] `QuiescenceGate`: A reusable primitive (`sync.Cond` wrapper) for workers that need to pause safely without losing in-flight data.
- **Documentation**:
  - [ ] `RECIPES.md`: A cookbook for common patterns (Interactive Service, Hot Reload, File Watcher).

## Backlog

- **Raw Mode Helpers**: Consider wrapping `x/term` Raw Mode enter/restore logic.
- **Cross-Platform Control Signals**: Research/Implement universal triggers (e.g., named pipes, admin CLI, HTTP) to replace limited OS signals (SIGTSTP/SIGUSR) on Windows/Linux.
- **Shutdown Diagnostics**: Dump goroutine stacks or waitgroup state if shutdown timeout (ForceExit) is triggered, to aid debugging hangs.
- **Lifecycle Testkit**: Helpers to simulate signals/events and assert state transitions without `time.Sleep` hacks.
- **Parallel Hooks**: Research "Parallel Hooks with Dependency Mapping" for high-performance shutdown.
- **Supervisor Spec**: Allow defining per-child restart policies (Always, OnFailure, Never).
- **Circuit Breaker**: Implement "MaxRestarts within Duration" logic (Erlang style).
- **Priority Shutdown**: Explicit shutdown phases (e.g., "Critical", "Normal").

## Technical Debt

> [!NOTE]
> **Technical Debt**: Items here are not blockers but should be addressed in future releases.

- [ ] **Technical Debt Analysis**: Map out the full scope of the library and identify areas of technical debt so we can add items here.
