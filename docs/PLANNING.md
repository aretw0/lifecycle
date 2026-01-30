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
- [ ] **Supervisor Implementation**: Create `Supervisor` to manage a set of `Worker`s with restart policies (OneForOne, OneForAll).
- [ ] **Tree Topology**: Verify `Supervisor` can supervise other `Supervisor`s (Tree structure).

#### Ecosystem Interfaces

- [ ] **Container Abstraction**: Define `Container` interface in `pkg/container` (Start, Stop, Logs) to decouple from Docker SDK.
- [ ] **Reference Implementation**: Add a mock or shell-based implementation to validate the `Container` interface.
- [ ] **Handover Protocol**: Standardize env vars (`LIFECYCLE_RESUME_ID`, `LIFECYCLE_PREV_EXIT`) for Supervisors to pass context to restarted Workers.

### v1.4: Durable Primitives (The Reliability Layer)

Focus: Features to support "Durable Execution" engines (like Trellis), distinguishing between "Stopping" and "Crashing".

- [ ] **Critical Sections**: Implement `lifecycle.Do(ctx, fn)` to delay context cancellation for atomic operations (The "Shield").
- [ ] **Shutdown Reason**: Add `Reason() enum` to `SignalContext` (Interrupt vs Terminate vs Timeout) to inform "Suspend" vs "Abort" decisions.
- [ ] **Input Safety**: Rework `InterruptibleReader` to support "Buffered Peek" or "Shielded Return" to prevent data loss on cancellation.

### v1.5: Portability

- [ ] **BSD/Solaris Support**: Verify `termio.Open()` behavior on other Unixes.

## Backlog

- **Raw Mode Helpers**: Consider wrapping `x/term` Raw Mode enter/restore logic if it becomes repetitive across projects?
- **Parallel Hooks**: Research "Parallel Hooks with Dependency Mapping" for high-performance shutdown scenarios (requested by user).
- **Composite Introspection Dashboard**: Define a standard way (or package) to aggregate `State()` from SignalContext and Workers to render a unified system diagram (e.g. at Supervisor level).

## Technical Debt

- [x] **Test Coverage**: Robust coverage for `pkg/signal` and `pkg/metrics` using internal mocks.
- [x] **Refactoring**: Simplified `IsInterrupted` in `pkg/termio` and "Sober" refactoring of `pkg/signal` for readability.
