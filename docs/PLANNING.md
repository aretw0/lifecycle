# Planning: Lifecycle Roadmap

> **Latest Release**: [v1.7.2](https://github.com/aretw0/lifecycle/releases/tag/v1.7.2)  
> **Current Focus**: v1.7.2 → v1.8.0 (In Progress)
>
> For completed releases (v1.5-v1.7), see [GitHub Releases](https://github.com/aretw0/lifecycle/releases).

---

### v1.8.0: Advanced Stability & Status Probing (Minor)

**Focus**: Evolve control plane self-healing, scaling, and coordinated lifecycle guarantees.

> [!NOTE]
> **Status**: Partially implemented. Some features are complete (promoted fields, health icons), while topology enforcement and dependency gates are planned for future iterations.

#### **Declarative Stability (Self-Healing)**: (v1.8+)

- [ ] **Topology Spec Enforcement**: Move from reactive restarts to periodic reconciliation. (v1.9+)
- [ ] **Dynamic Scaling**: Allow supervisors to adjust child counts based on external events. (v1.9+)
- [x] **Status Probing**: Add `Prober` interface to workers to go beyond simple "Process Alive" check.
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
  - [ ] Write comprehensive v1.10 release notes
  - [ ] Highlight breaking changes
  - [ ] Migration guide for early adopters (update MIGRATION.md)
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

#### Ecosystem Integration & Cleanup

- [ ] **Universal Introspection**:
  - [ ] Public `Introspectable` interface (`State() any`)
  - [ ] Generic adapters for Trellis, Loam, Arbour
- [ ] **Metadata Bridge Deprecation**:
  - [ ] Once `introspection` v0.2.0 is released and adopted, move to remove redundant data from `Metadata`.
  - [ ] Phase 1: Mark `worker.MetadataHealth` and friends as `@Deprecated` (v1.9).
  - [ ] Phase 2: Remove mirroring from `supervisor.go` (v2.0).
  
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

---

## Ecosystem Vision: Everything as Code (2026+)

> **Strategic Shift (2026-03-02)**: Lifecycle is now positioned as the **foundation layer** for a broader "Everything as Code" ecosystem.

### The Vision

Enable multiple domain-specific languages (DSLs) to compile to a **shared abstract execution engine**, all built on lifecycle's robust infrastructure.

```
┌───────────────────────────────────────────────────┐
│ DSL Layer (Domain-Specific Syntax)               │
│ ─────────────────────────────────────────────────│
│ • flow-dsl  (Workflows, state machines)          │
│ • life-dsl  (Habits, routines, energy mgmt)      │
│ • scrape-dsl (Web automation, selectors)         │
│ • deploy-dsl (Infrastructure orchestration)      │
└────────────────┬──────────────────────────────────┘
                 │ Compiles to IR
┌────────────────▼──────────────────────────────────┐
│ Abstract Execution Engine (Domain Agnostic)      │
│ ─────────────────────────────────────────────────│
│ • Node abstraction (generic task unit)           │
│ • Scheduler interface (cron, event, transition)  │
│ • State store (persistence)                      │
│ • Context propagation (lifecycle integration)    │
└────────────────┬──────────────────────────────────┘
                 │ Built on
┌────────────────▼──────────────────────────────────┐
│ Foundation (Infra Robustness)                    │
│ ─────────────────────────────────────────────────│
│ • lifecycle (this project)                       │
│ • loam (parsing)                                 │
│ • procio (process primitives)                    │
│ • introspection (visualization)                  │
└───────────────────────────────────────────────────┘
```

### Lifecycle's Role (Unchanging)

Lifecycle remains **domain-agnostic**. It provides:

- Signal handling & control plane
- I/O resilience & managed concurrency
- Durability primitives

**What lifecycle does NOT do**:

- ❌ Know about "flows", "workers", or "selectors"
- ❌ Provide DSL parsing or compilation
- ❌ Implement domain-specific schedulers

**Philosophy**: Solve infrastructure once. All DSLs benefit automatically.

### Ecosystem Phases (2026)

#### Phase 1: Foundation Stabilization ✅ (Complete)

- **Status**: Done
- **Milestone**: lifecycle v1.5 (Control Plane stable)
- **Outcome**: Ready for ecosystem integration

#### Phase 2: Engine Extraction (4-6 weeks)

- **Goal**: Separate Trellis DSL from generic execution engine
- **Deliverables**:
  - `trellis-engine` package (Node, Scheduler, StateStore interfaces)
  - Trellis refactored to use engine
  - Deep lifecycle integration (Router, Supervisor, Context)
- **Blocker**: Node abstraction design decision (Function/Interface/Hybrid)
- **Status**: Blocked (waiting for Trellis maintainer decision)

#### Phase 3: First Alternative DSL (6-8 weeks from now)

- **Goal**: Validate engine abstraction with second DSL
- **Deliverables**:
  - `life-dsl` v0.1.0 (life.yaml parser + compiler)
  - POC running on trellis-engine
  - Executors: CLI, HTTP, Browser, Notify
- **Dependencies**: Phase 2 must complete
- **Status**: Planned

#### Phase 4: DSL Ecosystem (3+ months)

- **Goal**: Enable community DSLs
- **Deliverables**:
  - `flow-dsl` extracted from Trellis
  - `scrape-dsl` for web automation
  - Plugin architecture stabilized
- **Status**: Future

### Key Documents for Ecosystem

- **[ECOSYSTEM_INTEGRATION.md](ECOSYSTEM_INTEGRATION.md)**: Lifecycle's integration status with all projects (use as template for other projects)
- **[ecosystem/engine_abstraction.md](ecosystem/engine_abstraction.md)**: Abstract engine vision & design
- **[ecosystem/arbour.md](ecosystem/arbour.md)**: Arbour's strategic reassessment

### Success Criteria

- [ ] 3+ DSLs compile to shared engine
- [ ] Engine deeply integrates lifecycle (Router, Supervisor, Context)
- [ ] All DSLs benefit from lifecycle improvements automatically
- [ ] Community can build custom DSLs with minimal friction

### Next Ecosystem Milestone

**Trellis Node Abstraction Decision** (This Week)

- **Owner**: Trellis maintainer
- **Options**: Function-based, Interface-based, Hybrid
- **Recommendation**: Hybrid (see `ecosystem/engine_abstraction.md`)
- **Impact**: Unblocks all of Phase 2
- **ETA**: 2026-03-08

---

## Related Ecosystem Projects

| Project | Role | Status | Next Milestone |
|---------|------|--------|----------------|
| **lifecycle** | Foundation | ✅ Stable (v1.5) | Maintain, support ecosystem |
| **trellis** | Engine + flow-dsl | ⏳ Needs refactor | Extract engine (Phase 2) |
| **arbour** | ChatOps/Plugins | ⏳ Strategic hold | Clarify role (Phase 3) |
| **loam** | Parser | ✅ Stable (v0.10) | Optional lifecycle adoption |
| **fiscus** | Reference impl | 🔮 Greenfield | Lifecycle-native architecture |
| **life-dsl** | Life as Code | 🔮 Not started | POC in Phase 3 |

---

## For Chat Session Continuity

When resuming development in a new chat session:

1. **Read First**: `docs/ECOSYSTEM_INTEGRATION.md`
2. **Check Blockers**: See "Integration Status" section
3. **Current Phase**: Foundation complete, waiting for Phase 2
4. **Active Work**: Trellis node abstraction decision
5. **Next Steps**: Support trellis-engine extraction once unblocked

---

**Last Updated**: 2026-03-02  
**Next Review**: After trellis-engine v0.1.0 or lifecycle v1.9 planning
