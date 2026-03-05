# Ecosystem Integration: Lifecycle

> **Purpose**: This document ensures continuity across development sessions by tracking lifecycle's role in the broader "Everything as Code" ecosystem.

## Project Identity

- **Name**: lifecycle
- **Version**: v1.5.x (stable)
- **Role**: Foundation Layer (Infra Robustness)
- **Repository**: [github.com/aretw0/lifecycle](https://github.com/aretw0/lifecycle)

## What Lifecycle Provides

Lifecycle is the **foundational infrastructure layer** for all "as code" projects in the ecosystem.

### Core Capabilities

1. **Signal Management**
   - Differentiate User Interrupts (SIGINT) from System Terminations (SIGTERM)
   - Escalation policies (first signal = graceful, second = force)
   - Cross-platform consistency (Windows CONIN$, Linux pipes)

2. **Control Plane (v1.5+)**
   - Event-driven router (`lifecycle.Router`)
   - Sources: OS signals, webhooks, file watches, health checks, input
   - Handlers: Shutdown, reload, suspend/resume
   - Middleware: Logging, recovery, metrics

3. **I/O Resilience**
   - Context-aware input (`lifecycle.InputSource`)
   - Non-blocking readers (prevents signal delivery issues)
   - Platform-specific handling (via `procio`)

4. **Managed Concurrency**
   - `lifecycle.Go` for tracked goroutines
   - `lifecycle.Supervisor` for service orchestration
   - Automatic cleanup on shutdown

5. **Durability Primitives**
   - `lifecycle.DoDetached` for critical sections
   - Grace period enforcement
   - Checkpoint hooks

### Integration Points

Lifecycle is designed to be **imported** by:

- Application frameworks (Trellis, Arbour)
- CLI tools (Loam, Fiscus)
- Any Go application needing robust lifecycle management

## Dependencies

### Direct Dependencies

- **procio** (v0.1.2): Process hygiene primitives (PDeathSig, Job Objects, termio)
- **introspection** (v0.1.3): Mermaid diagram generation

### No Dependencies On

- ❌ Trellis (higher-level framework)
- ❌ Loam (config parsing)
- ❌ Any DSL-specific logic

**Direction**: Lifecycle is at the **bottom** of the dependency tree.

```
┌──────────────────────────────────────┐
│ Applications & DSLs                  │
│ (trellis, arbour, life-dsl, etc.)    │
└────────────┬─────────────────────────┘
             │ depends on
┌────────────▼─────────────────────────┐
│ Foundation                           │
│ - lifecycle (this project)           │
│ - loam (parsing)                     │
│ - procio (process primitives)        │
└──────────────────────────────────────┘

┌──────────────────────────────────────┐
│ Browser Domain (TS/JS)               │
│ - refarm (concept transfer only)     │
│ - Future: Daemon uses lifecycle      │
└──────────────────────────────────────┘
```

## Integration Status (Per Project)

### ✅ Trellis (v0.7.1)

- **Status**: Legacy integration (uses lifecycle v0.1.1)
- **Usage**: `lifecycle.NewSignalContext` for basic cancellation
- **Next Steps**: Upgrade to v1.5 Control Plane (suspend/resume)
- **Blocker**: None (can upgrade anytime)
- **ETA**: 2-4 weeks

### ⏳ Arbour (v0.7.x)

- **Status**: Indirect (via Trellis)
- **Usage**: Inherits signal handling from Trellis
- **Next Steps**: Wait for strategic clarity (see `ecosystem/arbour.md`)
- **Blocker**: Engine abstraction design must complete first
- **ETA**: TBD (deferred until Phase 3)

### ✅ Loam (v0.10.x)

- **Status**: Independent (no dependency)
- **Usage**: None currently
- **Next Steps**: Retroactive adoption for CLI robustness
- **Blocker**: None (optional enhancement)
- **ETA**: Low priority (works fine as-is)

### 🆕 Fiscus (v0.0.x)

- **Status**: Greenfield (perfect for native integration)
- **Usage**: Planned lifecycle-first architecture
- **Next Steps**: Use as reference implementation
- **Blocker**: None
- **ETA**: When development starts

### 🌾 Refarm (v0.0.x)

- **Status**: Concept transfer (first non-Go project)
- **Usage**: Architectural patterns from lifecycle ecosystem inform TS/JS design
- **Integration**: Future Refarm Daemon (Go) will use `lifecycle.Run()` for OS-level access
- **Key Decision**: [ADR-008](https://github.com/aretw0/refarm/blob/main/specs/ADRs/ADR-008-ecosystem-technology-boundary.md) — TS for browser, Go for OS daemon
- **Blocker**: None (concept transfer is active, code integration is future)
- **ETA**: Daemon development TBD (after Refarm v0.1.0)
- **Docs**: [ecosystem/refarm.md](ecosystem/refarm.md)

### 🔮 Life-DSL (v0.0.x)

- **Status**: Not yet created (Phase 3)
- **Usage**: Will be **primary integration showcase**
- **Next Steps**: POC in `examples/life-dsl/`
- **Blocker**: Waiting for `trellis-engine` extraction (Phase 2)
- **ETA**: 6-8 weeks from now

### 🔮 Trellis-Engine (v0.0.x)

- **Status**: Not yet extracted (Phase 2)
- **Usage**: Will **heavily depend** on lifecycle
  - Router integration for control plane
  - Supervisor for worker orchestration
  - Context propagation throughout engine
- **Next Steps**: Extract from Trellis monolith
- **Blocker**: Design decisions (Node abstraction, Scheduler interface)
- **ETA**: 4-6 weeks

## Current Phase: Foundation Stabilization ✅

**Status**: Complete

Lifecycle v1.5 is **stable** and ready for production. All planned control plane features are implemented.

### What's Ready Now

- [x] Event sources (OS signals, webhooks, file watch, health checks, input)
- [x] Lifecycle handlers (shutdown, reload, suspend/resume)
- [x] Managed concurrency (Go, Supervisor)
- [x] Control router with middleware
- [x] Platform-specific I/O (Windows CONIN$)
- [x] Documentation & examples

### What's Not Ready

- ❌ Adoption by Trellis (still on v0.1.1)
- ❌ Engine abstraction design
- ❌ Life-DSL

## Next Phase: Engine Extraction (4-6 weeks)

**Goal**: Separate Trellis DSL from generic execution engine.

### Prerequisites (Lifecycle Side)

- ✅ Control plane stable
- ✅ Supervisor API finalized
- ✅ Examples demonstrate patterns

### Deliverables (Other Projects)

- [ ] `trellis-engine` package extracted
- [ ] Trellis refactored to use engine
- [ ] Engine integrates lifecycle deeply:
  - Uses `lifecycle.Router` for control events
  - Uses `lifecycle.Supervisor` for node orchestration
  - Respects `lifecycle.Context` cancellation

### Blockers

- **Design decision needed**: Node abstraction (Function vs Interface vs Hybrid)
  - **Recommendation**: Hybrid (see `ecosystem/engine_abstraction.md`)
  - **Owner**: Trellis maintainer to decide
  - **ETA**: This week

## Long-Term Vision: Everything as Code

### The Goal

Enable multiple DSLs (flow-dsl, life-dsl, scrape-dsl) to compile to a shared execution engine, all built on lifecycle's robust foundation.

### Lifecycle's Role (Unchanging)

- Provide signal handling
- Provide control plane
- Provide I/O resilience
- Provide concurrency primitives

**Philosophy**: Lifecycle stays generic. It does not know about "flows", "workers", or "selectors". It only knows about "events", "handlers", and "goroutines".

### Success Criteria

- [ ] 3+ DSLs compile to shared engine
- [ ] Engine uses lifecycle for robustness
- [ ] All DSLs benefit from lifecycle improvements automatically

## Design Principles

### 1. Lifecycle is a Library, Not a Framework

Applications **use** lifecycle; they don't **run on** it. There's no `lifecycle.Main()` or magic globals.

### 2. Zero Opinions on Domain Logic

Lifecycle doesn't care if you're managing "life tasks" or "web scrapes". It only cares about shutdown, signals, and I/O.

### 3. Backward Compatibility

Breaking changes are minimized. When unavoidable, follow SemVer strictly (v2.0 if needed).

### 4. Windows is a First-Class Citizen

All features must work on Windows (CONIN$, Job Objects, etc), not just Unix.

### 5. Observability by Default

Metrics, introspection, and debugging hooks are core, not afterthoughts.

## Communication Protocol (Cross-Project)

### For Chat Continuity

Each project maintains an `ECOSYSTEM_INTEGRATION.md` that answers:

1. **What does this project provide?**
2. **What does this project consume?**
3. **What's blocking us?**
4. **What's the next milestone?**

This allows future chat sessions to resume context instantly.

### For Dependency Updates

When lifecycle releases a new version:

1. Update `CHANGELOG.md` with breaking changes
2. Notify dependent projects via GitHub issues
3. Provide migration guides in `docs/`

### For Design Discussions

Major architectural changes (like engine abstraction) are documented in `docs/ecosystem/` **before** implementation.

## Action Items (Next 2 Weeks)

### For Lifecycle Maintainers

- [x] Document engine abstraction vision
- [x] Update ecosystem docs (arbour.md, this file)
- [ ] Create `examples/life-dsl/` POC (if trellis-engine design finalizes)

### For Trellis Maintainers

- [ ] Decide on Node abstraction (Function/Interface/Hybrid)
- [ ] Create `trellis/docs/ECOSYSTEM_INTEGRATION.md`
- [ ] Plan extraction roadmap in `PLANNING.md`

### For Arbour Maintainers

- [ ] Create `arbour/docs/ECOSYSTEM_INTEGRATION.md`
- [ ] Document strategic options (plugin hub vs life-dsl merge)
- [ ] Defer active development until Phase 3

### For Loam Maintainers

- [ ] Create `loam/docs/ECOSYSTEM_INTEGRATION.md`
- [ ] Document optional lifecycle adoption path

## Conclusion

Lifecycle is **stable** and **ready**. The ecosystem now waits for:

1. **Trellis-engine** extraction (Phase 2)
2. **Life-DSL** POC (Phase 3)
3. **Clear plugin architecture** emerges

This document will be updated as the ecosystem evolves.

---

**Last Updated**: 2026-03-05  
**Next Review**: After trellis-engine v0.1.0 release or Refarm Daemon development begins
