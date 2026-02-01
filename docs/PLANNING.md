# Planning: Lifecycle

## Roadmap

### v0.x: Foundation (Current)

- [x] **Signal Layer**: `pkg/signal` with `SignalContext` (Dual Signal support).
- [x] **I/O Layer**: `pkg/termio` with `InterruptibleReader` and Windows `CONIN$` support.
- [x] **Testing**: Unit tests for core logic.
- [x] **Release Automation**: GoReleaser configuration.

### v1.0: Stability (Release Candidate)

- [x] **Integration**: Adopted by `trellis` v0.8+.

### v1.1: Stewardship & Robustness

Focus: Ensure the lifecycle is resilient, debuggable, and safe.

- [x] **Process Hygiene**: Abstraction for `PDeathSig` (Linux) and `JobObjects` (Windows) to ensure children die with the parent (prevent zombies - "Fail-Closed").
- [x] **Shutdown Timeouts**: Add helpers for `Shutdown(ctx, timeout)` to prevent hangs.
- [x] **Observability**: `SetLogger` and `MetricsProvider` for lifecycle monitoring without external dependencies.
- [x] **Lifecycle Hooks**: A standard `OnShutdown(func())` registry to simplify consumer cleanup logic.
- [x] **TermIO Automation Spike**: Verified "Peek & Abandon" behavior via `pkg/termio/blocking_test.go`.

### v1.2: Introspection & Visibility

Focus: runtime transparency. Allow the application to describe its own lifecycle configuration ("Introspection") to generate live documentation (Mermaid).

- [x] **Context Introspection**: Add `State() State` to `SignalContext` to expose immutable configuration (DTO pattern).
- [x] **Visualization**: Add `Mermaid(State) string` renderer to generation state diagrams from the introspection data.
- [x] **Stewardship**: Establish the "Introspection Pattern" (State DTOs) as a standard for future components (Process Workers, Supervisors).

### v1.3: Ecosystem Convergence (The Supervisor Pattern)

Focus: Robust management of child processes and support for "Agents" (Trellis/Arbour-like).

#### The Supervisor Pattern

- [x] **Worker Protocol**: Define `Worker` interface (Start, Stop, Wait, Info) for uniform management of processes and goroutines. (Implemented in `pkg/worker`)
- [x] **Process Worker**: Implement `Worker` for `exec.Cmd` utilizing `pkg/proc` for hygiene (Fail-Closed).
- [x] **Supervisor Implementation**: Create `Supervisor` to manage a set of `Worker`s with restart policies (OneForOne, OneForAll).
- [x] **Tree Topology**: Verify `Supervisor` can supervise other `Supervisor`s (Tree structure). (Verified in `pkg/supervisor/supervisor_test.go`)
- [x] **Unified Dashboard**: Synthesize `SignalContext` and `Worker` states into a single Mermaid diagram (The "Composite Introspection").
  - [x] Update `signal.State` to include runtime signal data.
  - [x] Implement `lifecycle.SystemDiagram` helper.

#### Ecosystem Interfaces

- [x] **Container Abstraction**: Define `Container` interface in `pkg/container` (Start, Stop, Logs) to decouple from Docker SDK.
- [x] **Reference Implementation**: Add a mock or shell-based implementation to validate the `Container` interface.
- [x] **Handover Protocol**: Standardize env vars (`LIFECYCLE_RESUME_ID`, `LIFECYCLE_PREV_EXIT`) for Supervisors to pass context to restarted Workers.

### v1.3.1: Supervisor Refinements

Focus: Hardening the Supervisor implementation and improving developer ergonomics based on ecosystem analysis.

- [x] **Backoff Strategy**: Implement exponential backoff for restarts to prevent "Tight Loop" CPU burning on immediate child failures. (Promoted from Backlog)
- [x] **Dynamic Topology**: Support adding/removing child workers at runtime (`Add(Spec)`) to support "Connection Manager" patterns, not just static Daemons.
- [x] **Task Adapter**: Introduce `worker.FromFunc(fn)` to allow simple functions to be supervised without full `struct` boilerplate. ("Lightweight Contract")

### v1.4: Durable Primitives (The Reliability Layer)

Focus: Features to support "Durable Execution" engines (like Trellis), distinguishing between "Stopping" and "Crashing".

- [x] **Critical Sections**: Implement `lifecycle.Do(ctx, fn)` to delay context cancellation for atomic operations (The "Shield").
- [x] **Shutdown Reason**: Add `Reason() enum` to `SignalContext` (Interrupt vs Terminate vs Timeout) to inform "Suspend" vs "Abort" decisions.
- [x] **Input Safety**: Rework `InterruptibleReader` to support "Buffered Peek" or "Shielded Return" to prevent data loss on cancellation.
- [x] **Resumable Worker**: Worker that can be paused/resumed via Token (Trellis convergence).
- [x] **Token Protocol**: Standardize passing `LIFECYCLE_RESUME_ID` (acts as Resume Token) via Env.

### v1.5: Maintenance Mode (LTS)

> [!IMPORTANT]
> **Feature Freeze**: v1.4 marks the completion of the "Death Management" feature set.
> Future v1.x releases will focus strictly on bug fixes, OS compatibility (e.g., BSD support), and performance.
> All new "Control Plane" features target v2.0.

- [x] **Cross-Platform Verification**: Added macOS CI support (BSD proxy) to ensure `termio.Open()` portability.

### v2.0: Control Plane (Event-Driven Lifecycle)

> [!NOTE]
> **Evolution Strategy**:
>
> - **v1.x**: Remains the stable, lightweight standard for **Death Management** (Shutdown & Signals). Ideal for simple CLIs.
> - **v2.x**: Introduces the **Control Plane** (Life Management). A superset for complex applications needing Event Routing, Hot Reloading, and Dynamic Reactions.

Focus: Transform `lifecycle` from a "Shutdown Manager" into a dynamic "Application Control Plane". Decouple "Events" (Triggers) from "Reactions" (Actions).

- [ ] **Event Sources (Inputs)**: Generalized interface for things that trigger lifecycle changes.
  - `OSSignalSource` (SIGINT, SIGTERM)
  - `WebhookSource` (Admin HTTP endpoints)
  - `HealthCheckSource` (Promoted from Backlog: `Probe()` failures trigger restarts/shutdowns)
  - `FileWatchSource` (Integration with Loam?)
  - **Progress Events**: `ctx.Progress(0.5)` or `lifecycle.Tick(ctx)` to drive UI/Loaders without coupling (Headless Timers).
- [ ] **Lifecycle Reactions (Outputs)**: Dynamic responses to events.
  - `Shutdown` (Current behavior)
  - `HotReload` (Promoted from Backlog: SIGHUP triggers config reload without context cancellation)
  - `Suspend/Resume` (For Durable Execution)
- [ ] **Managed Concurrency (The "lifecycle.Go" Pattern)**:
  - `lifecycle.Go(ctx, fn)`: A helper to spawn goroutines that are automatically tracked, waited on, and shielded from premature Context cancellation. Enforces "Safe Concurrency" by default.
- [ ] **Control Router**: Configurable logic mapping `Source -> Reaction` (e.g., "On SIGHUP -> Reload", "On SIGINT -> Shutdown").
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

- [x] **Test Coverage**: Robust coverage for `pkg/signal` and `pkg/metrics` using internal mocks.
- [x] **Refactoring**: Simplified `IsInterrupted` in `pkg/termio` and "Sober" refactoring of `pkg/signal` for readability.
