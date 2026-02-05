# Planning: Lifecycle

> [!IMPORTANT]
> **Strategy Clarification (2026-02-03)**:
>
> - **v1.x is deprecated** — No users to protect. v2.0 will launch as "the lifecycle" (no confusing versioning).
> - **Roadmap is work-driven, not date-driven** — Organized by iterative releases (v2.1, v2.2) instead of fixed timelines.
> - **Working solo with AI partners** — Planning reflects single-maintainer capacity with agent collaboration.

---

## Current State: v2.0 Stabilization (BETA) 🛠️

> [!IMPORTANT]
> **v2.0 Stabilization Phase**: Based on the critical analysis (2026-02-05), the architectural split and baseline test coverage are now prioritized as **blockers for the stable v2.0 launch** to ensure a robust foundation and prevent immediate breaking changes in v2.x.

### Completed Work (v2.0 Core)

- [x] **Event Sources (Inputs)**: Generalized interface for lifecycle changes.
  - [x] `OSSignalSource` (SIGINT, SIGTERM)
  - [x] `WebhookSource` (Admin HTTP endpoints - Skeleton)
  - [x] `HealthCheckSource` (Promoted from Backlog - Skeleton)
  - [x] `FileWatchSource` (Integration with Loam? - Skeleton)
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

## Quality Baseline (Current Status)

### Test Coverage Snapshot (2026-02-05 - Final Update)

**Current Coverage by Package**:

```log
lifecycle (root)       : 15.4%  ⚠️ BELOW TARGET
pkg/core/signal        : 85.9%  ✨ EXCELLENT (PASSED TARGET!)
pkg/core/runtime       : 89.3%  ✨ EXCELLENT
pkg/core/supervisor    : 84.4%  ⚠️ NEAR TARGET (84.4% → need 0.6% more)
pkg/events             : 52.5%  🟡 IMPROVED (37.5% → 52.5%)
pkg/core/termio        : 51.4%  ⚠️ NEEDS IMPROVEMENT
pkg/core/proc          : 55.9%  ⚠️ NEEDS IMPROVEMENT
pkg/core/metrics       : 20.4%  🔴 CRITICAL GAP
pkg/core/container     : 0.0%   🔴 CRITICAL GAP
pkg/core/introspection : 0.0%   🔴 CRITICAL GAP
pkg/core/log           : 0.0%   🔴 CRITICAL GAP
```

**Overall Coverage (Tested Packages)**: 72.7%

**Target**: ≥80% average coverage across all packages before v2.1 launch

**Progress Summary (2026-02-05)**:
- Added tests for `pkg/core/signal`: WithResetTimeout, IsUnsafe, ForceExitThreshold → +3.2%
- Added tests for `pkg/core/runtime`: improved TestRun_LeakWarning → validation
- Added tests for `pkg/events`: EventType String methods → +2.1%
- Added tests for `pkg/events`: Router State, Routes methods → coverage improvement

**Action Items**:

- [ ] Generate full coverage report: `go test -coverprofile=coverage.out ./...`
- [ ] Identify untested critical paths (signal handling, Windows I/O, supervisor restarts)
- [ ] Prioritize coverage for foundation packages (`signal`, `runtime`, `supervisor`)

---

### v2.0 Refinement (Stable Blockers)

#### 🏗️ Architecture Refinement

> [!NOTE]
> **Architectural Reflection Required**:
> Before implementing `core/events` split, we need to carefully consider:
>
> - Import cycles between packages
> - Facade API design (`lifecycle.go` as unified entry point)
> - Migration path for existing examples
> - Performance implications of modular boundaries
> - Documentation structure (separate docs for core vs events?)

- [x] **Analyze Current Package Structure**:
  - [x] Map dependency graph of `pkg/*` packages
  - [x] Identify coupling between foundation (signal, I/O) and control plane (router, handlers)
  - [x] Document current import patterns and potential cycles
  
- [x] **Design Core/Events Split**:
  - [x] Architectural analysis performed during v2.0 refinement.
  - [x] Validated with existing examples (successfully refactored).
  
- [x] **Implement Modular Architecture**:
  - [x] Create `core/` package structure.
  - [x] Create `events/` package structure.
  - [x] Refactor `lifecycle.go` as facade.
  - [x] Update all examples to use new structure.
  - [x] Ensure zero breaking changes for `import "github.com/aretw0/lifecycle"` users.

#### 🧪 Test Coverage Sanity

- [ ] **Foundation Packages** (Target: 85%+):
  - [ ] `pkg/signal`: Full state machine coverage (all transitions, edge cases)
  - [ ] `pkg/runtime`: lifecycle.Go, Do, Group (panic recovery, wait groups)
  - [ ] `pkg/supervisor`: All restart policies, circuit breaker, handover protocol
  - [ ] `pkg/termio`: Windows CONIN$, Unix stdin, interrupt handling
  
- [ ] **Control Plane Packages** (Target: 75%+):
  - [ ] `pkg/events`: Router, Sources and Handlers consolidated.
  
- [ ] **Integration Tests**:
  - [ ] Windows-specific tests (CONIN$, Job Objects)
  - [ ] Signal escalation scenarios (1st SIGINT, 2nd SIGINT, SIGTERM)
  - [ ] Supervisor recovery (child crashes, circuit breaker triggers)
  - [ ] Shutdown timeout diagnostics (goroutine dump)
  
- [ ] **CI/CD**:
  - [ ] Add coverage badge to README
  - [ ] Fail CI if coverage drops below 80%
  - [ ] Run tests on Windows, macOS, Linux

---

## Roadmap: Iterative Releases

### v2.1: Advanced Stability & Ecosystem

**Focus**: Evolve the control plane functionality and self-healing capabilities.

#### **Declarative Stability (Self-Healing)**: (v2.1+)

- [ ] **Topology Spec Enforcement**: Move from reactive restarts to periodic reconciliation.
- [ ] **Dynamic Scaling**: Allow supervisors to adjust child counts based on external events.
- [ ] **Status Probing**: Add `Prober` interface to workers to go beyond simple "Process Alive" check.

#### 🐛 Technical Debt Resolution

- [ ] **Debt Mapping**:
  - [ ] Scan codebase for `TODO`, `FIXME`, `HACK` comments
  - [ ] Identify unstable/experimental APIs (mark with `// Experimental:`)
  - [ ] Document known limitations clearly:
    - macOS: No PDeathSig (best-effort zombie prevention)
    - `path.Match`: Glob only, not full regex
    - Performance: ~5-10μs overhead per `lifecycle.Go` call
  
- [ ] **Update Technical Debt Section** (below) with concrete items

#### 📚 Migration API

- [ ] **Hybrid Setup Pattern**:
  - [ ] Implement `lifecycle.Context()` for manual setup:

    ```go
    func main() {
        ctx := lifecycle.Context() // returns signal context
        defer lifecycle.Shutdown(ctx)
        
        myExistingApp(ctx) // just swap context.Background() → lifecycle.Context()
    }
    ```

  - [ ] Document in `docs/MIGRATION.md`:
    - Pattern A: `lifecycle.Run` (zero config, recommended)
    - Pattern B: `lifecycle.Context` (manual, for gradual migration)
    - Trade-offs of each approach
  - [ ] Create example for both patterns

---

### v2.2: Documentation & Marketing

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

### v2.3: Public Launch

**Focus**: Maximize visibility and initial adoption.

#### 🚀 Launch Preparation

- [ ] **Release Notes**:
  - [ ] Write `CHANGELOG.md` (v1.x → v2.0 → v2.3)
  - [ ] Highlight breaking changes (if any)
  - [ ] Migration guide for early adopters (if applicable)
  - [ ] Feature showcase with code examples
  
- [ ] **Marketing Materials**:
  - [ ] Blog post: "Lifecycle 2.0: Stop Fighting Signals and Shutdowns in Go"
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

### v2.4+: Feature Expansion (Future)

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

> [!WARNING]
> **Technical Debt**: Non-blocking, but address before claiming "production-ready".

### Identified Debt (To Be Expanded After v2.1 Audit)

- [ ] **Low Test Coverage**:
  - Root package: 15.2% (target: 80%+)
  - `pkg/metrics`: 21.0% (target: 70%+)
  - `pkg/termio`: 51.0% (target: 80%+)
  
- [ ] **Missing Platform Tests**:
  - No CI for Windows-specific behavior (CONIN$, Job Objects)
  - macOS PDeathSig failure mode untested
  
- [ ] **API Stability**:
  - Some experimental features not marked (need audit)
  - No deprecation policy documented
  
- [ ] **Documentation Gaps**:
  - No migration guide for existing apps
  - Windows behavior underdocumented despite being differentiator
  - No benchmark data published
  
- [ ] **Performance Unknowns**:
  - Introspection overhead unmeasured
  - Router throughput (events/sec) not benchmarked
  - Memory footprint of large supervision trees unknown

### Future Debt Prevention

- [ ] Establish coverage gate in CI (80% minimum)
- [ ] Require ADR for architectural changes
- [ ] Document experimental APIs with `// Experimental:` comments
- [ ] Regular quarterly debt audits

---

## Development Philosophy (Maintained)

From original planning:

- **Managed Global State**: We abstract the inevitable global state (OS Signals) into clean, context-aware usage. Prefer `Context` propagation, but enjoy `DefaultRouter` convenience.
- **Leak-Free**: Every resource (goroutine, file handle) must close on shutdown.
- **Platform Agnostic**: Windows `CONIN$` handling is a first-class citizen, not an afterthought.
- **Observability**: Internal state changes must be visible via `pkg/metrics` interfaces.
- **Solo Development**: Roadmap reflects single-maintainer capacity with AI collaboration. Work is divided into achievable increments, not date-driven sprints.
