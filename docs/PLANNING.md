# Planning: Lifecycle

> [!IMPORTANT]
> **Strategy Clarification (2026-02-15)**:
>
> - **v1.5+ introduces breaking changes** — The Control Plane architecture (originally planned as "v2") is now released as v1.5 to avoid `go.mod` migration overhead. See [MIGRATION.md](MIGRATION.md) for upgrade guidance.
> - **Roadmap is work-driven, not date-driven** — Organized by iterative releases (v1.6, v1.7) instead of fixed timelines.
> - **Working solo with AI partners** — Planning reflects single-maintainer capacity with agent collaboration.

---

## v1.5.0 (Released) ✅

> [!IMPORTANT]
> **v1.5 Stabilization Complete**: All architectural split work and baseline test coverage milestones have been achieved. The v1.5 Control Plane foundation is stable and ready for production use.
>
> **Breaking Changes**: v1.5 introduces the Event-Driven Control Plane, which includes breaking API changes from v1.4. See [MIGRATION.md](MIGRATION.md) for details.

### Completed Work (v1.5 Control Plane)

- [x] **Event Sources (Inputs)**: Generalized interface for lifecycle changes.
  - [x] `OSSignalSource` (SIGINT, SIGTERM)
  - [x] `WebhookSource` (Admin HTTP endpoints)
  - [x] `HealthCheckSource` (Promoted from Backlog)
  - [x] `FileWatchSource` (Integration with Loam?)
  - [x] **Progress Events**: `ctx.Progress(0.5)` or `lifecycle.Tick(ctx)` to drive UI/Loaders
  
- [x] **Lifecycle Handlers (Outputs)**: Dynamic responses to events.
  - [x] `Shutdown` (Current behavior)
  - [x] `HotReload` (Cross-platform config reload via FileWatchSource, see `examples/reload/`)
  - [x] `Suspend/Resume` (Verified with `examples/suspend`)
  
- [x] **Managed Concurrency**: `lifecycle.Go(ctx, fn)` pattern
  - Automatically tracked, waited on, panic-safe
  - Global WaitGroup in `pkg/runtime`
  - `Supervisor` for services, `Go` for tasks
  
- [x] **Control Router**: Stdlib-native mux for events
  - [x] Pattern matching (glob via `path.Match`, partial regex)
  - [x] Middleware chains (logging, tracing, recovery)
  - [x] Synchronous default for safety
  - [x] Introspection (`Routes()` for visualization)
  
- [x] **Ergonomics & Interactions**:
  - [x] `InputSource` (detached readers, Windows-resilient)
  - [x] `NewInteractiveRouter` preset
  - [x] `QuiescenceGate` (context-aware pause primitive)
  
- [x] **Relational Reliability**:
  - [x] Shutdown diagnostics (goroutine stack dumps)
  - [x] Supervisor circuit breaker
  - [x] Restart policies (`Always`, `OnFailure`, `Never`)
  
- [x] **Documentation**:
  - [x] `RECIPES.md` cookbook

---

### v1.5 Refinement (Stable Blockers)

#### 🏗️ Architecture Refinement

> [!NOTE]
> **Architectural Split Completed (v1.5)**:
> The `core/events` split has been successfully implemented:
>
> - Clean separation between Foundation (`pkg/core`) and Control Plane (`pkg/events`)
> - Facade API design (`lifecycle.go` as unified entry point)
> - All examples updated to use new structure
> - Zero breaking changes for basic `import "github.com/aretw0/lifecycle"` usage

- [x] **Analyze Current Package Structure**:
  - [x] Map dependency graph of `pkg/*` packages
  - [x] Identify coupling between foundation (signal, I/O) and control plane (router, handlers)
  - [x] Document current import patterns and potential cycles
  
- [x] **Design Core/Events Split**:
  - [x] Architectural analysis performed during v1.5 development.
  - [x] Validated with existing examples (successfully refactored).
  
- [x] **Implement Modular Architecture**:
  - [x] Create `core/` package structure.
  - [x] Create `events/` package structure.
  - [x] Refactor `lifecycle.go` as facade.
  - [x] Update all examples to use new structure.
  - [x] Ensure backward compatibility for `import "github.com/aretw0/lifecycle"` users.

#### 🛰️ Architecture: Procio Extraction (Completed)

> [!IMPORTANT]
> **Refactoring Objective**:
> We have successfully extracted the core process and I/O primitives into **[procio](https://github.com/aretw0/procio)**. This reduces the dependency footprint and elevates these primitives to first-class status.

- [x] **Transition to Multi-Module (Internal Isolation)**:
  - [x] Initialize `go.work` (Dev Mode).
  - [x] Extracted `proc`, `termio`, `scan` to `procio`.
  - [x] `lifecycle` now depends on `procio` v0.1.2.

- [x] **Promote `proc` & `termio` Ecosystem**:
  - [x] **Procio** is now a standalone library.
  - [x] Specialized documentation created in `procio` repo.
  - [x] `lifecycle` acts as the "Policy Engine" over `procio` "Mechanics".

#### 🎨 Architecture: Introspection Extraction (Completed)

> [!IMPORTANT]
> **Refactoring Objective**:
> We have successfully extracted generic visualization primitives into **[introspection](https://github.com/aretw0/introspection)**. This decouples domain logic from presentation logic and enables broader adoption of `lifecycle`'s introspection patterns.

- [x] **Primitive Promotion**:
  - [x] Removed `pkg/core/introspection` package (~1500 lines).
  - [x] Extracted generic diagram builders (`TreeDiagram`, `StateMachineDiagram`, `ComponentDiagram`).
  - [x] `lifecycle` now depends on `introspection` v0.1.2.

- [x] **Adapter Pattern**:
  - [x] Created `diagram_config.go` (centralized configuration).
  - [x] Moved `NodeStyler`/`NodeLabeler` to `pkg/core/worker`.
  - [x] Moved `PrimaryStyler`/`PrimaryLabeler` to `pkg/core/signal`.
  - [x] Updated `SystemDiagram` to use `introspection.ComponentDiagram`.

- [x] **Code Cleanup ("Fat Removal")**:
  - [x] Removed `RenderTreeFragment` from `worker/diagram.go`.
  - [x] Removed `RenderFragment` from `signal/diagram.go`.
  - [x] Delegated all Mermaid string generation to external library.

#### 🧪 Test Coverage Sanity

> See **[TESTING.md](TESTING.md)** for our "Honest Coverage" policy and exclusions.

- [x] **Analysis**: Identified gaps in `supervisor` and `router`.
- [x] **Implementation**:
  - [x] `pkg/core/supervisor`: Backoff & State Tests
  - [x] `pkg/events/router`: Introspection Tests
- [x] `pkg/events/suspend`: Robustness Tests
- [x] **Verification**:
  - [x] Verify mapped exclusions (`metrics`, `termio`) are documented.
  - [x] Achieve target coverage for Core Logic.

---

## v1.5.1 (lint patch) ✅

- [x] **Lint**: Add lint target and fix all issues (golangci-lint)

## v1.5.2 (Alignment patch) ✅

- [x] **Analysis**: Panic recovery in `lifecycle.Go()` only logs the panic value. To diagnose root causes, stack traces are essential, but not in production (log noise vs. disk I/O cost).
- [x] **Design**: Implement conditional stack capture—log full `runtime/debug.Stack()` only when debug logging is enabled (`slog.LevelDebug`).
- [x] **Implementation**:
  - [x] Modified `pkg/core/runtime/task.go` panic handler to check logger level before capturing stack.
  - [x] Added `import "runtime/debug"` for stack extraction.
  - [x] Stack traces appear in development (helpful for debugging), omitted in production (reduces overhead).
- [x] **Documentation**:
  - [x] Create `docs/MIGRATION.md` with v1.5 upgrade guide and patterns.
  - [x] Update documentation and examples to align with v1.5 changes.
  - [x] Updated code comment explaining the strategy (aligns with TECHNICAL.md §2.4).
  - [x] Non-breaking change (conditional capture has zero overhead if debug is disabled).

---

## Roadmap: Iterative Releases

### v1.6: Core Stability & Observability (Major Milestone + Patches)

**Focus**: Extensible panic observability and a clean migration entry point.

#### v1.6.0 (Major Milestone): Extensible Panic Observability + Migration Entry Points

**Context**: v1.5.2 introduced conditional stack traces (Nivel 1). Now extend this with full customization via the `Observer` pattern and formalize gradual adoption.

> [!IMPORTANT]
> **v1.6.0 Completed (2026-02-16)**: All items below have been implemented, tested, and documented. Ready for release.
>
> - Core feature: `Observer.OnGoroutinePanicked(any, []byte)` hook with stack capture control
> - Migration entry point: `lifecycle.Context()` for gradual adoption
> - Documentation: TECHNICAL.md (§8.1 cleanup pattern + §14 observability), MIGRATION.md condensed
> - Examples: `context/main.go` (new), `observability/main.go` (enhanced)
> - Tests: All 15 tests passing, 3 stack-capture scenarios covered

- [x] **Observer Interface Extension**:
  - [x] Add `OnGoroutinePanicked(recovered any, stack []byte)` method to `pkg/core/observe.Observer`.
  - [x] Allows consumers to route panics to external systems (Sentry, Datadog, structured logging backends).
  - [x] Stack bytes are pre-captured (no performance penalty if observer is unused).

- [x] **GoOption Configuration** (Explicit Control):
  - [x] Add `WithStackCapture(bool)` option to `lifecycle.Go()`.
  - [x] Default: Auto-detect (capture if debug level enabled, otherwise skip).
  - [x] Allows forced capture in production for critical tasks, or forced skip in debug for performance testing.
  - [x] Example:

    ```go
    lifecycle.Go(ctx, criticalTask, lifecycle.WithStackCapture(true))   // Always capture
    lifecycle.Go(ctx, debugTask, lifecycle.WithStackCapture(false))     // Never capture
    lifecycle.Go(ctx, normalTask)                                        // Auto-detect
    ```

- [x] **Worker Integration Pattern**:
  - [x] Document how custom worker implementations (e.g., Loam's `watchWorker`) can leverage the same pattern.
  - [x] Provide example in `examples/` showing panic recovery + stack capture for custom workers.
  - [x] **Protected Resource Cleanup Pattern** (Discovered in Loam's debouncer - Feb 2026):
    - When async callbacks (e.g., debouncer timers) send on channels, formalize this 3-stage shutdown:
      1. **STOP**: Flag to reject new work (e.g., `debouncer.closed = true`)
      2. **WAIT**: Use `sync.WaitGroup` to track in-flight callbacks, wait before proceeding
      3. **CLOSE**: Safe to close resources (channels, etc.) - no goroutines will try to send
    - [x] Document in TECHNICAL.md (new section after "Worker Protocol" §8)
    - [x] Provide `stopAndWait(timeout)` helper pattern / example
    - [x] Applicable to: Watchers, debouncers, queue processors, any system with **async callbacks + channel closure**

- [x] **Downstream Adoption** (Ready for Implementation):
  - [x] Loam `pkg/adapters/fs` can migrate from custom panic handling to `WithStackCapture(true)`.
  - [x] Trellis task execution can use `OnGoroutinePanicked` hook for operational dashboards.

- [x] **Migration API (Hybrid Setup Pattern)**:
  - [x] Implement `lifecycle.Context()` for manual setup:

    ```go
    func main() {
        ctx := lifecycle.Context() // returns signal context
        defer lifecycle.Shutdown(ctx)
        
        myExistingApp(ctx) // just swap context.Background() → lifecycle.Context()
    }
    ```

  - [x] Document in `docs/MIGRATION.md`:
    - [x] Pattern A: `lifecycle.Run` (zero config, recommended)
    - [x] Pattern B: `lifecycle.Context` (manual, for gradual migration)
    - [x] Trade-offs of each approach
  - [x] Create example for both patterns

#### v1.6.1 (Patch): Technical Debt Mapping & Documentation

> [!IMPORTANT]
> **v1.6.1 Completed (2026-02-16)**: Technical debt has been mapped and documented.
> This patch focuses on **Transparency & API Clarity**, not on performance measurement or new tests.
> Benchmarks and comprehensive testing of router limitations are planned for **v1.6.2**.

- [x] **Debt Mapping & Documentation**:
  - [x] Scan codebase for `TODO`, `FIXME`, `HACK` comments (1 TODO found: router.go:192 "Optimize if many routes")
  - [x] Identify unstable/experimental APIs (all APIs either Stable or marked as such)
  - [x] Document known limitations clearly in new [LIMITATIONS.md](LIMITATIONS.md):
    - [x] macOS: No PDeathSig (best-effort zombie prevention)
    - [x] `path.Match`: Glob only, not full regex
    - [x] Performance: ~5-10μs overhead per `lifecycle.Go` call
    - [x] Coverage: High (>80%) for core logic; excluded metrics/termio/filewatch
    - [x] Compatibility matrix: Go 1.20+, Windows, macOS, Linux
  
- [x] **Documentation & Code Annotations**:
  - [x] Created [LIMITATIONS.md](LIMITATIONS.md) (comprehensive; 200+ lines)
  - [x] Added §15 "Known Limitations" to TECHNICAL.md with cross-links
  - [x] Updated TESTING.md to reference compliance matrix
  - [x] Code annotations in 4 files to clarify API stability:
    - [x] `pkg/core/observe/observe.go`: Observer marked Stable (v1.6.0)
    - [x] `pkg/events/router.go`: Glob-only pattern syntax documented
    - [x] `pkg/core/runtime/task.go`: Three-mode stack capture explained
    - [x] `lifecycle.go`: APIs Context() and WithStackCapture() stabilized

---

### v1.6.2 (Patch): Performance Benchmarks & Consolidation

**Focus**: Measure actual overhead, consolidate multi-source documentation, provide optimization path.

> [!IMPORTANT]
> **v1.6.2 Completed (2026-02-17)**: All benchmarking and documentation consolidation work has been implemented.
>
> - Comprehensive benchmarks: `pkg/core/runtime/benchmark_test.go` and `pkg/events/router_benchmark_test.go`
> - Real performance data: Measured overhead for `lifecycle.Go`, router patterns, supervisor introspection
> - Documentation consolidated: Single source of truth in LIMITATIONS.md with cross-references
> - Automation: `make benchmark` target and platform-specific scripts ready

#### v1.6.2 Completed Work

- [x] **Benchmarks** (pkg/core/runtime/benchmark_test.go):
  - [x] `lifecycle.Go` vs `go func()` + manual WaitGroup
  - [x] Router matching: exact match vs glob match (O(1) vs O(n))
  - [x] Observer invocation overhead (with OnGoroutinePanicked hook)
  - [x] Stack capture overhead (enabled vs disabled vs auto-detect)
  - [x] Large supervision tree traversal (10, 100, 1000 workers)

- [x] **Benchmarks** (pkg/events/router_benchmark_test.go):
  - [x] Route matching performance scaling (10-1000 routes)
  - [x] Middleware chain overhead (0, 1, 5, 10 middleware)
  - [x] Router throughput (sequential and parallel dispatching)
  - [x] Memory allocation patterns
  - [x] Concurrent read/write contention

- [x] **Router Documentation Consolidation**:
  - [x] Simplified router.go inline comments (removed detailed examples)
  - [x] LIMITATIONS.md now single source of truth for pattern syntax
  - [x] Added benchmark data table with real measurements
  - [x] TECHNICAL.md updated with cross-link to LIMITATIONS.md

- [x] **Baseline Unknowns → Known**:
  - [x] Measured introspection overhead (State() calls on 10/100/1000 worker trees)
  - [x] Measured Router throughput (events/sec with varying route counts)
  - [x] Measured memory footprint patterns
  - [x] Updated LIMITATIONS.md with all measured data and methodology

- [x] **Automation & Tooling**:
  - [x] Created `scripts/run_benchmarks.ps1` (Windows)
  - [x] Created `scripts/run_benchmarks.sh` (Unix)
  - [x] Added `make benchmark` target to Makefile

---

### v1.6.3 (Patch): Workers withLock sync and etc

- [x] **Some Workers**: Add `withLock` and `withLockResult` for atomic reads.
- [x] **SetEnv and SetOutput**: To return error if process is already stopped.
- [x] **State**: Added timestamps.

> **Note:** The exceptions and limitations of the `withLock/withLockResult` pattern are now documented in [docs/LIMITATIONS.md](LIMITATIONS.md) to centralize technical knowledge and facilitate maintenance. See there for updated details.

---

### v1.6.4 (Patch): Signal Busy Loop Fix

- [x] **Fix**: Corrected a busy loop in `signal.Context` where the monitor loop would consume 100% CPU after context cancellation.
  - The `Done` channel case is now disabled after cancellation while keeping signal monitoring active for "Force Exit" logic.
- [x] **Verification**: Verified via tests.

---

### v1.6.5 (Patch): Hardening & Security

**Focus**: Critical security fixes and test stability for immediate production reliability.

- [x] **Security (WebhookSource)**:
  - [x] Implement `http.MaxBytesReader` in `pkg/events/webhook.go`.
  - [x] Prevent OOM (Out Of Memory) attacks by limiting request body size (e.g., 1MB default). Added `WithMaxPayloadBytes` option.
- [x] **Test Stability (FileWatchSource)**:
  - [x] Replace `time.Sleep` in `pkg/events/filewatch_test.go` with deterministic synchronization.
  - [x] Ensure watcher is ready before triggering file events in tests.
- [x] **Documentation Sync**:
  - [x] Update `PLANNING.md` and `TECHNICAL.md` to remove "Skeleton" labels from functional components.
- [x] **Generic Worker Hardening (Race Condition)**:
  - [x] Implement utility `lifecycle.StopAndWait(ctx, worker)` to atomically stop and wait for worker termination.
  - [x] Address race condition where `Stop` returns before `stdout/stderr` buffers are closed or background routines finish (discovered in Trellis usage).
- [x] **Sober Edit (API Consistency)**:
  - [x] Enforce Facade Pattern in `lifecycle.go` to never leak internal types (e.g., `signal.Context`, `worker.BaseWorker`).

---

### v1.7.0 (Minor): Advanced Watching, Throttling & Event Broker Delegation ✅

**Focus**: Evolve the Control Plane with robust watching patterns, generic event utilities, and event broker responsibilities to offload downstream projects (like Loam).

> [!IMPORTANT]
> **v1.7.0 Completed (2026-02-21)**: All advanced watching and delegation items below have been implemented, tested, and documented.
>
> - `events.Notify(ch)` pattern implemented for Pub/Sub and safe external channel integration.
> - `events.DebounceHandler` implemented to throttle high-frequency burst events.
> - `FileWatchSource` expanded with `WithRecursive(true)` and `WithFilter` options for deep, targeted file monitoring.
> - `RECIPES.md` updated with the new "Inhibition Pattern" recipe.

- [x] **Channel Subscriptions (Pub/Sub nativo no Router)**:
  - [x] Implement `events.Notify(c chan<- Event)` (inspired by `signal.Notify`).
  - [x] Allow Router to return native Go channels (`<-chan Event`) instead of only pushing via callbacks, enabling synchronous iteration for clients.
- [x] **Generic Event Utilities (Debounce)**:
  - [x] Implement a generic **Debouncer** utility (`events.DebounceHandler(handler, window time.Duration, mergeFunc)`).
  - [x] Provide out-of-the-box noise dampening for high-frequency events (e.g., rapid FileSystem writes).
- [x] **FileWatchSource Expansion (Recursive FS)**:
  - [x] Extracted and evolved `FileWatchSource` to support deep monitoring.
  - [x] Add `WithRecursive(true)` and `WithFilter(func(path string) bool)` support.
  - [x] *Use Case*: Loam delegates directory watching and `index.lock` git-pause filtering natively without maintaining deep `fsnotify` boilerplate.
- [x] **Refinement**:
  - [x] Document "Inhibition Pattern" as a generic way to handle "Git-awareness" without hardcoding Git logic.

---

### v1.7.1 (Patch): Pathological Context Enforcement

**Focus**: Implementing hygiene rules defined by ADR-0016 to prevent zombie chains and enforce strict contextual control in child execution (borrowing lessons from Rod/Chrome DevTools patterns).

- [ ] **ADR-0016 Propagation Audit**:
  - [ ] Review all internal Gorotine spawning (`lifecycle.Go()`, Supervisors) to guarantee they are derived directly from the operation's context, never defaulting implicitly to `context.Background()` or global router contexts.
- [ ] **Chained Cancels**:
  - [ ] Document in Reference how downstream libraries (`trellis`, `loam` process runners) should enforce immediate process killing using cascaded cancellation over SIGKILL thresholds in deep hierarchies, leveraging the mechanics already provided by the *upstream* `procio` package.

---

### v1.7.2 (Patch): Context Cancellation Coverage

**Focus**: Close the testing blind spot exposed by the v1.7.1 audit. While the pathological `context.Background()` usages were fixed in `supervisor.go` and `task.go`, no test previously validated that cancelling the parent context **during a supervisor restart** would propagate and actually stop the operation. This patch adds that safety net and closes the remaining webhook detachment.

- [ ] **Supervisor Cancellation Tests**:
  - [ ] Add test: cancel parent `ctx` while a `StrategyOneForAll` restart is in progress → assert supervisor stops, children don't re-spawn.
  - [ ] Add test: cancel parent `ctx` while `Add()` is dynamically starting a child → assert child start is aborted cleanly.
- [ ] **Webhook Context Fix**:
  - [ ] In `pkg/events/webhook.go`, replace `context.WithTimeout(context.Background(), 5*time.Second)` with `context.WithTimeout(ctx, 5*time.Second)` in the server shutdown path.

---

### v1.8.0: Advanced Stability & Shared State (Minor)

**Focus**: Evolve control plane self-healing, scaling, and coordinated lifecycle guarantees.

#### **Declarative Stability (Self-Healing)**: (v1.8+)

- [ ] **Topology Spec Enforcement**: Move from reactive restarts to periodic reconciliation.
- [ ] **Dynamic Scaling**: Allow supervisors to adjust child counts based on external events.
- [ ] **Status Probing**: Add `Prober` interface to workers to go beyond simple "Process Alive" check.
- [ ] **Worker Role Grouping** (ADR-0015): Add Role-based scheduling to Supervisor for targeted control and Leader Election awareness.

#### **Coordinated Lifecycle & Shared State**: (v1.8+)

- [ ] **Dependency Gates**: Block provider shutdown until all registered consumers have reached a safe state. (Shared State Quiescence)
- [ ] **Barrier Patterns**: Primitives to ensure concurrent workers synchronize their lifecycle transitions.
- [ ] **State Bridge**: Mechanism to hand over "hot" state (e.g., open file descriptors, active sessions) across restarts without serialization.
- [ ] **Durable Event Router Extension** (ADR-0014): Capability to connect Router to durable sinks to support process pause/resume across reboots.

---

### v1.9.0: Documentation & Marketing

**Focus**: Make lifecycle discoverable, understandable, and compelling.

#### 📖 Documentation Overhaul

- [ ] **README.md Reformation**:
  - [ ] Lead with **The Problem** (concrete pain points):

    ```markdown
    ## The Problem
    **Ever had your Go CLI hang on Windows when you press Ctrl+C?**
    **Or found zombie child processes after your app crashes?**
    **Or struggled with goroutines that never shut down cleanly?**
    ```

  - [ ] Add **Quick Win** section (10-line working example)
  - [ ] Add **When to Use** decision tree
  - [ ] Add **Comparison Table** (vs cobra, errgroup, signal.Notify)
  - [ ] Move "Vision" to PRODUCT.md (keep README practical)
  - [ ] Add badges: Coverage, Go Report Card, License, "Used by"
  
- [ ] **docs/QUICKSTART.md** (new):
  - [ ] Level 1: Basic CLI (`lifecycle.Run`) — 15 lines
  - [ ] Level 2: Interactive REPL (`NewInteractiveRouter`) — 30 lines
  - [ ] Level 3: Long-Running Service (`Supervisor`) — 50 lines
  - [ ] Each level fully executable (`go run`)
  
- [ ] **docs/WINDOWS.md** (new — Marketing Differentiator):
  - [ ] Explain Windows stdin blocking problem
    - Link to Stack Overflow questions showing pain
    - Show error: `EOF` instead of graceful shutdown
  - [ ] Detail CONIN$ solution (with diagram)
  - [ ] Before/After comparison:

    ```text
    Without lifecycle: [screenshot of hang]
    With lifecycle:    [screenshot of clean exit]
    ```

  - [ ] Section on Job Objects preventing zombies
  - [ ] Emphasize: "No other Go library does this automatically"
  
- [ ] **docs/COMPARISON.md** (new):
  - [ ] **vs cobra/urfave/kong**: Complementary (runtime vs arg parsing)
  - [ ] **vs errgroup/conc**: Adds signal awareness, limits, panic recovery
  - [ ] **vs signal.Notify**: Adds state machine, platform quirks, escalation
  - [ ] Side-by-side code examples
  
- [ ] **docs/RECIPES.md** (expand):
  - [ ] Add GIF/screenshot for each recipe
  - [ ] Recipe: Migrate existing app (context.Background → lifecycle)
  - [ ] Recipe: Hot reload config without restart
  - [ ] Recipe: REPL with history and autocomplete
  - [ ] Recipe: Signal Detachment (Stop vs Cancel) - demonstrate handing over control
  
- [ ] **docs/LIMITATIONS.md** (new — Transparency):
  - [ ] Windows: Go 1.20+ required for full Job Objects support
  - [ ] macOS: No PDeathSig (zombies possible on hard crashes)
  - [ ] Router: `path.Match` is glob, not full regex
  - [ ] Performance: Measured overhead (from benchmarks)

#### 🎨 Visualization & Introspection 2.0 (Promoted)

- [ ] **Separate Topology from Status**:
  - [ ] Static Tree (Logic) vs Dynamic Status (Health)
  - [ ] Prominently highlight "Stuck" or "Zombie" nodes
  - [ ] Real-time dashboard skeleton (Web/Terminal)

#### 🧪 Validation & Social Proof

- [ ] **Benchmarks**:
  - [ ] Publish `docs/BENCHMARKS.md`:
    - `lifecycle.Go` vs `go func()` + manual WaitGroup
    - `lifecycle.Group` vs `errgroup.Group`
    - Introspection overhead (State() calls)
    - Memory footprint (Router with 100+ handlers)
  - [ ] Include graphs/charts if possible
  
- [ ] **Case Study**:
  - [ ] Write `docs/CASE_STUDY_TRELLIS.md`:
    - What problems lifecycle solved in Trellis
    - Code before/after snippets
    - Impact metrics (if available)
  - [ ] Use as "Used by" social proof
  
- [ ] **Platform Validation**:
  - [ ] Test on:
    - Windows 10, 11, Server 2022
    - macOS (Intel + Apple Silicon)
    - Linux (Ubuntu, Alpine, RHEL)
    - Docker (Alpine base image)
    - Kubernetes (graceful shutdown with SIGTERM)
  - [ ] Document compatibility matrix in README

---

### v1.10: Public Launch

**Focus**: Maximize visibility and initial adoption.

#### 🚀 Launch Preparation

- [ ] **Release Notes**:
  - [ ] Write `CHANGELOG.md` (v1.0 → v1.5 → v1.8)
  - [ ] Highlight breaking changes (v1.5 Control Plane)
  - [ ] Migration guide for early adopters (MIGRATION.md)
  - [ ] Feature showcase with code examples
  
- [ ] **Marketing Materials**:
  - [ ] Blog post: "Lifecycle 1.5: Stop Fighting Signals and Shutdowns in Go"
    - Subtitle: "Windows Support That Actually Works"
    - Sections: The Problem, The Solution, How It Works, Get Started
  - [ ] Tweet thread (5-7 tweets):
    - Hook: "Tired of Go CLIs hanging on Windows?"
    - Pain points with examples
    - Lifecycle solution
    - Call to action (GitHub link)
  - [ ] GIFs/Videos:
    - CLI with Ctrl+C on Windows (before/after)
    - REPL with suspend/resume live demo
    - Supervisor auto-restart visualization
  
- [ ] **Community Submissions**:
  - [ ] Submit PR to [awesome-go](https://github.com/avelino/awesome-go)
  - [ ] Post to [r/golang](https://reddit.com/r/golang) (with context, not just link)
  - [ ] Submit to [Go Weekly newsletter](https://golangweekly.com/)
  - [ ] Consider Hacker News (if material is strong enough)

#### 📊 Post-Launch Monitoring

- [ ] **Metrics Tracking**:
  - [ ] Monitor GitHub stars, forks, watchers
  - [ ] Track pkg.go.dev import stats (if available)
  - [ ] Watch for mentions on Reddit, Twitter, Stack Overflow
  
- [ ] **Community Engagement**:
  - [ ] Set up issue labels: `bug`, `feature`, `question`, `docs`, `good first issue`
  - [ ] Response SLA: <48h for issues, <7d for PRs
  - [ ] Monthly review of feature requests → prioritize for backlog
  
- [ ] **Success Criteria** (90 days post-launch):
  - [ ] 100+ GitHub stars
  - [ ] 5+ projects publicly using lifecycle
  - [ ] 10+ community issues/PRs
  - [ ] Listed in awesome-go

---

### v1.11+: Feature Expansion (Future)

**Focus**: Evolve based on community feedback and adoption patterns.

#### From Backlog (Prioritize Based on Demand)

- [ ] **Telemetry Integration**:
  - [ ] OpenTelemetry adapter for `pkg/metrics`
  - [ ] Standardized trace/span emission
  - [ ] Integration guide for observability stacks
  
- [ ] **Lifecycle Testkit**:
  - [ ] Helpers to simulate signals in tests
  - [ ] Event assertion utilities
  - [ ] Mock sources and handlers
  - [ ] Eliminate `time.Sleep` hacks in tests
  
- [ ] **Raw Mode Helpers**:
  - [ ] Wrapper for `x/term` Raw Mode enter/restore
  - [ ] Ensure cross-platform (Windows console modes)
  
- [ ] **Cross-Platform Control Signals**:
  - [ ] Named pipes (Windows/Linux)
  - [ ] HTTP admin endpoint (standard pattern)
  - [ ] Replace limited SIGTSTP/SIGUSR with universal triggers
  
- [ ] **Advanced Shutdown**:
  - [ ] Parallel hooks with dependency mapping
  - [ ] Priority shutdown phases (Critical → Normal → Optional)
  - [ ] **Stuck Worker Detection**: (Lesson from `suture`)
    - [ ] Heartbeat/Progress requirements for workers
    - [ ] Auto-dump stack for specific "stuck" component before global timeout

- [ ] **Worker Behaviors (Behaviors)**: (Lesson from `ergo`)
  - [ ] Standard templates for common patterns:
    - [ ] `PeriodicWorker` (Ticker-based)
    - [ ] `StreamWorker` (Producer/Consumer with backpressure)
    - [ ] `StatefulWorker` (Standard state machine template)

#### Ecosystem Integration

- [ ] **Universal Introspection**:
  - [ ] Public `Introspectable` interface (`State() any`)
  - [ ] Generic adapters for Trellis, Loam, Arbour
  
- [ ] **Visualization 2.0**:
  - [ ] Separate Topology (static) from Status (dynamic)
  - [ ] Show missing/crashed nodes prominently
  - [ ] Real-time dashboard (web UI?)

---

## Technical Debt (Concrete Items)

> [!TIP]
> **Reality Check (v1.6.4)**: Much of the early technical debt (coverage, performance unknowns) has been resolved. The focus now shifts to **Hardening** and **CI Reliability**.

### Resolved Debt (v1.5 - v1.6.2)

- [x] **Low Test Coverage**: Re-baselined and achieved >85% in core packages.
- [x] **Performance Unknowns**: Introspection, Router, and Runtime overhead measured and documented in `LIMITATIONS.md`.
- [x] **Documentation Gaps**: `MIGRATION.md` and `LIMITATIONS.md` created; `TECHNICAL.md` updated.

### Active Technical Debt

- [ ] **Test Flakiness (High Priority)**:
  - Several tests (notably `FileWatchSource` and `Supervisor`) still use `time.Sleep` for synchronization.
  - Risk: Intermittent CI failures.
  - Solution: Move to deterministic synchronization (channels/Ready hooks).
- [ ] **Webhook Resilience (Security)**:
  - `WebhookSource` lacks request body limits (OOM risk).
  - Solution: Implement `http.MaxBytesReader` (Target: v1.6.5).
- [ ] **Experimental API Audit**:
  - APIs like `suspend`, `webhook`, and `health` are functional but need explicit `// Experimental` or `// Stable` tags in code to match `LIMITATIONS.md`.
- [ ] **Cross-Platform CI**:
  - Windows/macOS specific features (CONIN$, Job Objects) are tested locally but lack automated runners in GitHub Actions.
  - Solution: Configure platform-specific CI jobs.

### Future Debt Prevention

- [ ] **Automated Quality Gates**:
  - [ ] Implement strict coverage gate in CI (fail PR if <80% on core packages).
  - [ ] Add performance regression check (run benchmarks in CI and compare with baseline).
- [ ] **Governance & Documentation**:
  - [ ] Require **ADR (Architecture Decision Records)** for any change affecting the Control Plane or Worker state machine.
  - [ ] Maintain the "Transparency First" policy in `LIMITATIONS.md` for every Minor release.
- [ ] **Release Discipline**:
  - [ ] Strict adherence to SemVer (especially distinguishing Patch vs Minor for new Watcher features).
  - [ ] Regular quarterly debt audits to re-baseline "Honest Coverage" exclusions.

---

## Development Philosophy (Maintained)

From original planning:

- **Managed Global State**: We abstract the inevitable global state (OS Signals) into clean, context-aware usage. Prefer `Context` propagation, but enjoy `DefaultRouter` convenience.
- **Leak-Free**: Every resource (goroutine, file handle) must close on shutdown.
- **Platform Agnostic**: Windows `CONIN$` handling is a first-class citizen, not an afterthought.
- **Observability**: Internal state changes must be visible via `pkg/metrics` interfaces.
- **Solo Development**: Roadmap reflects single-maintainer capacity with AI collaboration. Work is divided into achievable increments, not date-driven sprints.
