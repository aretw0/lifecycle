# Ecosystem Evolution: Trellis (Engine Abstraction Pioneer)

> **Status**: Strategic Refactoring in Progress  
> **Last Updated**: 2026-03-02

## Historical Context

**Trellis** (v0.7.1) is currently a **monolithic framework** that combines:

- **DSL**: Flow definitions, state machines, transitions (domain-specific)
- **Engine**: Graph execution, state persistence, durability (generic)

This coupling was pragmatic for initial development, but now limits reuse across domains.

## Strategic Vision: Engine Abstraction

Trellis is evolving into the **first realization** of the Abstract Execution Engine vision (see [engine_abstraction.md](./engine_abstraction.md)).

### The Transformation

```
┌─────────────────────────────────────────────────┐
│ Before: Trellis (Monolith)                      │
│ ─────────────────────────────────────────────── │
│ • DSL (flows, states, transitions)              │
│ • Engine (graph execution, persistence)         │
│ • Lifecycle integration (basic)                 │
└─────────────────────────────────────────────────┘

                      ↓ Refactor

┌─────────────────────────────────────────────────┐
│ After: Separated Architecture                   │
└─────────────────────────────────────────────────┘

┌──────────────────┐  ┌───────────────────────────┐
│ flow-dsl         │  │ trellis-engine            │
│ (Domain Syntax)  │  │ (Generic Execution)       │
│ ──────────────── │  │ ─────────────────────────│
│ • Parse flows    │→ │ • Node abstraction        │
│ • Compile to IR  │  │ • Scheduler interface     │
│ • State machine  │  │ • State store             │
│   semantics      │  │ • Lifecycle integration   │
└──────────────────┘  └───────────────────────────┘
         ↓                        ↑
    ┌────┴────────────────────────┴──────┐
    │ trellis (Facade - Backward Compat) │
    └────────────────────────────────────┘
```

### Why Separate?

**Problem**: If we want "Life as Code" or "WebScrape as Code", we must rebuild everything.

**Solution**: Extract generic execution primitives. DSLs become thin layers that compile to the same engine.

**Benefit**: Write engine once, N DSLs benefit (reuse scheduler, state store, durability, lifecycle integration).

## Current Lifecycle Integration (v0.7.1)

::: warning
**Version Discrepancy**: Trellis currently depends on **lifecycle v0.1.1** (legacy).

The patterns described below represent the **Target Architecture** for Trellis post-refactor.
Full "Suspend/Resume" and Control Plane integration will happen during Phase 2 (Engine Extraction).
:::

### Today's Integration

- Uses `lifecycle.NewSignalContext` for basic cancellation
- No suspend/resume support
- No control plane routing

### Post-Refactor Integration (Phase 2)

Trellis-engine will **deeply integrate** lifecycle:

```go
// Deep lifecycle integration in trellis-engine
type Engine struct {
    router     *lifecycle.Router      // Control plane events
    supervisor *lifecycle.Supervisor  // Worker orchestration
    store      StateStore             // Persistence
}

func (e *Engine) Run(ctx context.Context, nodes []Executable) error {
    // Integrate with lifecycle for signals
    ctx = lifecycle.Attach(ctx, e.router)
    
    // Use supervisor for node execution
    for _, node := range nodes {
        worker := e.supervisor.Add(node.ID(), node.Execute)
        worker.RestartPolicy = lifecycle.Always
    }
    
    return lifecycle.Run(ctx, e.supervisor)
}
```

**Key Features**:

1. **Suspend/Resume**: Engine responds to `SuspendEvent` for checkpointing
2. **Graceful Shutdown**: Uses `lifecycle.DoDetached` for critical state transitions
3. **Health Checks**: Exposes node health via `lifecycle.HealthCheckSource`
4. **File Watch**: Reload flow definitions via `lifecycle.FileWatchSource`

## Refactoring Roadmap (Phase 2)

### Week 1-2: Design & Prototype

- [ ] **Node Abstraction Decision** (Critical Blocker)
  - Options: Function-based, Interface-based, Hybrid
  - Recommendation: Hybrid (see [engine_abstraction.md](./engine_abstraction.md))
  - Owner: Trellis maintainer
  - **ETA**: This week (2026-03-08)

- [ ] **Package Structure**

  ```
  trellis/
  ├── engine/          ← New: Generic execution
  │   ├── node.go
  │   ├── scheduler.go
  │   ├── store.go
  │   └── lifecycle_integration.go
  ├── dsl/             ← Refactored: Flow-specific
  │   ├── parser.go
  │   ├── compiler.go
  │   └── flow_types.go
  └── trellis.go       ← Facade: Backward compat
  ```

### Week 3-4: Implementation

- [ ] Extract `engine/` package with:
  - `Node` interface (or hybrid struct)
  - `Scheduler` interface (StateMachineScheduler, CronScheduler, etc.)
  - `StateStore` interface (SQLite, Redis, Memory, Loam)
  - Deep lifecycle integration (Router, Supervisor, Context)

- [ ] Refactor Trellis to use `engine/` internally
- [ ] Maintain 100% backward compatibility via facade

### Week 5: Validation

- [ ] Run existing Trellis tests (should pass unchanged)
- [ ] Run Arbour (indirect consumer) to validate
- [ ] Performance benchmarks (ensure no regression)

### Week 6: Lifecycle Upgrade

- [ ] Upgrade from lifecycle v0.1.1 → v1.5
- [ ] Implement suspend/resume handlers
- [ ] Add control plane examples

**Total ETA**: 4-6 weeks from node decision

## Integration with Other DSLs (Phase 3+)

Once trellis-engine exists, new DSLs can emerge:

### life-dsl (Life as Code)

```go
import "github.com/aretw0/trellis/engine"

// Life workers compile to same engine
func (w *LifeWorker) Compile() engine.Executable {
    return engine.NewNode(w.ID).
        Execute(w.ActionFunc()).
        Schedule(engine.CronScheduler(w.Schedule)).
        Build()
}
```

### scrape-dsl (Web Automation)

```go
import "github.com/aretw0/trellis/engine"

// Scrape selectors compile to same engine
func (s *Selector) Compile() engine.Executable {
    return engine.NewNode(s.ID).
        Execute(s.BrowserFunc()).
        Schedule(engine.SelectorScheduler(s.Steps)).
        Build()
}
```

**All use the same**:

- State persistence
- Lifecycle integration
- Durability primitives
- Error handling

## Key Design Principles

### 1. Backward Compatibility First

Existing Trellis users must not break. Facade pattern ensures:

```go
import "github.com/aretw0/trellis"

// Still works
engine := trellis.New(path)
engine.Run(ctx)
```

### 2. Progressive Migration

Users can opt-in to advanced features:

```go
import "github.com/aretw0/trellis/engine"

// Advanced: Direct engine usage
eng := engine.New(
    engine.WithLifecycle(router),
    engine.WithStore(sqlite),
)
```

### 3. Separation of Concerns

- **DSL**: Syntax, domain concepts, compilation
- **Engine**: Execution, scheduling, persistence
- **Lifecycle**: Signals, I/O, control plane

No circular dependencies.

### 4. Composition Over Inheritance

DSLs **compose** with engine primitives, not subclass them.

## Blockers & Dependencies

### Critical Blocker

**Node Abstraction Design** (Waiting on Decision)

- **Impact**: Blocks all of Phase 2
- **Owner**: Trellis maintainer
- **Options**: See [engine_abstraction.md](./engine_abstraction.md)
- **Recommendation**: Hybrid (function + interface)
- **ETA**: This week

### Dependencies

- ✅ Lifecycle v1.5 (stable, ready)
- ✅ Loam v0.10 (stable, ready)
- ⏳ Node design decision (blocking)

## Success Criteria

- [ ] `trellis-engine` package extracts successfully
- [ ] Existing Trellis tests pass unchanged
- [ ] Arbour continues working (indirect consumer)
- [ ] `life-dsl` POC compiles to engine (Phase 3 validation)
- [ ] Performance within 5% of monolithic version

## For Chat Session Continuity

When resuming Trellis development:

1. **Check Blocker**: Node abstraction decided?
2. **Current Phase**: Week 1 (Design) blocked
3. **Next Steps**: Once unblocked, create `engine/` package structure
4. **Documentation**: Update `trellis/docs/ECOSYSTEM_INTEGRATION.md` (needs creation)

---

**Last Updated**: 2026-03-02  
**Next Review**: After node abstraction decision (this week)

---

## Related Documents

- [engine_abstraction.md](./engine_abstraction.md) - Full vision & design options
- [Trellis Refactoring Guide](https://github.com/aretw0/trellis/blob/main/docs/REFACTORING_GUIDE.md) - Refactoring roadmap & preserved insights
- [README.md](./README.md) - Lifecycle ecosystem status and component index
- [trellis/docs/ECOSYSTEM_INTEGRATION.md](https://github.com/aretw0/trellis/blob/main/docs/ECOSYSTEM_INTEGRATION.md) - Trellis-side integration map
