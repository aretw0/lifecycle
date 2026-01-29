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

### v1.2: Ecosystem Convergence (The Supervisor Pattern)

Focus: Robust management of child processes and support for "Agents" (Trellis/Arbour-like).

- [ ] **Supervisor**: Implementation of a Supervisor pattern to manage groups of `exec.Cmd` (Restart policies, Group Shutdown).
- [ ] **Container Interface**: Definition of interfaces for container orchestration (without direct Docker SDK dependency).

### v1.3: Portability

- [ ] **BSD/Solaris Support**: Verify `termio.Open()` behavior on other Unixes.

## Backlog

- **Non-destructive I/O**: Research a "Buffered Peek" for `InterruptibleReader` to avoid data loss when context is cancelled during a read.
- **Raw Mode Helpers**: Consider wrapping `x/term` Raw Mode enter/restore logic if it becomes repetitive across projects?
- **Parallel Hooks**: Research "Parallel Hooks with Dependency Mapping" for high-performance shutdown scenarios (requested by user).

## Technical Debt

- [x] **Test Coverage**: Robust coverage for `pkg/signal` and `pkg/metrics` using internal mocks.
- [x] **Refactoring**: Simplified `IsInterrupted` in `pkg/termio` and "Sober" refactoring of `pkg/signal` for readability.
