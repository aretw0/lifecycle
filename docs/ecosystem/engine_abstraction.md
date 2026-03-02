# Ecosystem Architecture: Abstract Execution Engine

> **Vision**: A universal execution engine that enables "Everything as Code" by separating domain-specific syntax (DSL) from execution primitives.

## The Problem: Coupling DSL to Engine

Today, **Trellis** is a monolithic framework that couples:

- **DSL** (Flows, States, Transitions) ← Domain-specific syntax
- **Engine** (Graph execution, State management, Durability) ← Generic execution primitives

This prevents reuse. If we want "Life as Code" or "WebScrape as Code", we must rebuild everything from scratch.

## The Solution: DSL ↔ Engine Separation

```
┌──────────────────────────────────────────────────────────┐
│ Abstract Execution Engine (Domain Agnostic)              │
│ ──────────────────────────────────────────────────────── │
│ • Node abstraction (generic task unit)                   │
│ • Graph representation (DAG/FSM agnostic)                │
│ • Scheduler interface (cron, event, state-transition)    │
│ • State store (persistence interface)                    │
│ • Context propagation (lifecycle integration)            │
│ • Durability primitives (checkpoint/resume)              │
└──────────────────────────────────────────────────────────┘
         ↑                  ↑                  ↑
         │ Compiles to IR   │ Compiles to IR   │ Compiles to IR
         │                  │                  │
    ┌────┴─────┐      ┌─────┴──────┐      ┌────┴──────┐
    │ flow-dsl │      │ life-dsl   │      │scrape-dsl │
    │ (Trellis)│      │(Workers)   │      │(Selectors)│
    └──────────┘      └────────────┘      └───────────┘
         ↓                  ↓                   ↓
     flows.yaml         life.yaml          scrape.yaml
```

**Key Insight**: All DSLs compile to the same **Intermediate Representation (IR)** that the engine executes.

---

## Naming Strategy

### Option 1: Keep "Trellis" as Engine Name

- **trellis** (generic engine) ← Becomes abstract
- **trellis-flow-dsl** (original syntax) ← Specific to workflows
- **trellis-life-dsl** (new) ← Life management
- **trellis-scrape-dsl** (future) ← Web automation

**Pros**: "Trellis" keeps brand recognition.  
**Cons**: Confusion — people might think it's still workflow-specific.

### Option 2: Generic Engine Name + Specific DSLs

- **conductor** (or **orchestrator**, **executor**) ← New name for engine
- **flow-dsl** ← Trellis syntax migrates here
- **life-dsl** ← Life management
- **scrape-dsl** ← Web automation

**Pros**: Crystal clear separation.  
**Cons**: "Trellis" loses meaning (becomes a legacy alias).

### Option 3: Hybrid Branding

- **trellis-engine** (core) ← Generic execution
- **trellis** (facade) ← Backward compat wrapper (engine + flow-dsl)
- **flow-dsl** ← Original Trellis syntax
- **life-dsl**, **scrape-dsl**, etc. ← New domains

**Pros**: Evolution without breaking existing users.  
**Cons**: Slightly verbose.

### 🎯 Recommendation: **Option 3 (Hybrid Branding)**

- Existing Trellis users keep working (`import "github.com/aretw0/trellis"`).
- New users can choose: Full Trellis or just engine (`import "github.com/aretw0/trellis/engine"`).
- DSLs are explicit libraries (`import "github.com/aretw0/flow-dsl"`).

---

## Node Abstraction: Design Options

### Option A: Function-Based (Simplest)

```go
// Pure functions — no interfaces
type Node struct {
    ID       string
    Execute  func(ctx context.Context) error
    Schedule func(ctx context.Context) (*Node, error)
    Metadata map[string]any
}

// Example: Life worker
func NewWorkoutWorker() *Node {
    return &Node{
        ID: "workout",
        Execute: func(ctx context.Context) error {
            return exec.Command("notify-send", "Time to workout!").Run()
        },
        Schedule: func(ctx context.Context) (*Node, error) {
            // Cron logic: "Every day 7am"
            next := time.Now().Add(24 * time.Hour)
            return nil, scheduler.WaitUntil(ctx, next)
        },
    }
}
```

**Pros**:

- Simple, no interfaces.
- Easy to test (inject functions).
- Minimal boilerplate.

**Cons**:

- No type safety for node-specific behavior.
- Hard to extend with lifecycle hooks (`OnStart`, `OnFailure`).
- Metadata is stringly-typed.

---

### Option B: Interface-Based (Most Extensible)

```go
// Type-safe, polymorphic
type Node interface {
    ID() string
    Execute(ctx context.Context) error
    Scheduler() Scheduler
    OnStart(ctx context.Context) error   // Hook
    OnFailure(ctx context.Context, err error) error
}

type Scheduler interface {
    Next(ctx context.Context) (Node, error)
}

// Example: Life worker (struct implements Node)
type WorkoutWorker struct {
    id       string
    schedule string // Cron expression
}

func (w *WorkoutWorker) ID() string { return w.id }

func (w *WorkoutWorker) Execute(ctx context.Context) error {
    return exec.Command("notify-send", "Time to workout!").Run()
}

func (w *WorkoutWorker) Scheduler() Scheduler {
    return &CronScheduler{Expression: w.schedule}
}

func (w *WorkoutWorker) OnStart(ctx context.Context) error {
    log.Info("Starting workout worker")
    return nil
}

func (w *WorkoutWorker) OnFailure(ctx context.Context, err error) error {
    log.Error("Workout failed", "error", err)
    return EscalateToPriority(ctx, "health")
}
```

**Pros**:

- Type-safe.
- Extensible (can add more methods to interface).
- Clear lifecycle hooks.
- IDE autocomplete for node-specific behavior.

**Cons**:

- More boilerplate (struct + methods).
- Harder to mock in tests (need mock implementations).
- Less flexible than functions (can't swap logic on-the-fly).

---

### Option C: Hybrid (Best of Both Worlds)

```go
// Base struct with function fields (flexibility)
// + Interface for lifecycle hooks (type safety)

type Node struct {
    id       string
    execute  func(ctx context.Context) error
    schedule Scheduler
    
    // Optional: Lifecycle hooks
    onStart   func(ctx context.Context) error
    onFailure func(ctx context.Context, err error) error
    
    // Metadata for introspection
    metadata *NodeMetadata
}

// Interface for engine compatibility
type Executable interface {
    ID() string
    Execute(ctx context.Context) error
    Scheduler() Scheduler
}

func (n *Node) ID() string { return n.id }
func (n *Node) Execute(ctx context.Context) error {
    if n.onStart != nil {
        if err := n.onStart(ctx); err != nil {
            return err
        }
    }
    
    err := n.execute(ctx)
    
    if err != nil && n.onFailure != nil {
        return n.onFailure(ctx, err)
    }
    return err
}

func (n *Node) Scheduler() Scheduler { return n.schedule }

// Builder pattern for ergonomics
func NewNode(id string) *NodeBuilder {
    return &NodeBuilder{node: &Node{id: id}}
}

type NodeBuilder struct {
    node *Node
}

func (b *NodeBuilder) Execute(fn func(ctx context.Context) error) *NodeBuilder {
    b.node.execute = fn
    return b
}

func (b *NodeBuilder) Schedule(s Scheduler) *NodeBuilder {
    b.node.schedule = s
    return b
}

func (b *NodeBuilder) OnStart(fn func(ctx context.Context) error) *NodeBuilder {
    b.node.onStart = fn
    return b
}

func (b *NodeBuilder) OnFailure(fn func(ctx context.Context, err error) error) *NodeBuilder {
    b.node.onFailure = fn
    return b
}

func (b *NodeBuilder) Metadata(m *NodeMetadata) *NodeBuilder {
    b.node.metadata = m
    return b
}

func (b *NodeBuilder) Build() Executable {
    return b.node
}

// Example: Life worker (fluent API)
func NewWorkoutWorker() Executable {
    return NewNode("workout").
        Execute(func(ctx context.Context) error {
            return exec.Command("notify-send", "Time to workout!").Run()
        }).
        Schedule(CronScheduler("0 7 * * *")).
        OnStart(func(ctx context.Context) error {
            log.Info("Starting workout worker")
            return nil
        }).
        OnFailure(func(ctx context.Context, err error) error {
            log.Error("Workout failed", "error", err)
            return EscalateToPriority(ctx, "health")
        }).
        Metadata(&NodeMetadata{
            Category: "health",
            Priority: 1,
            EnergyBudget: 20,
        }).
        Build()
}
```

**Pros**:

- **Flexibility**: Functions for logic (easy to test, inject).
- **Type Safety**: Interface for engine contract.
- **Ergonomics**: Builder pattern = clean, readable.
- **Extensible**: Can add hooks without breaking existing nodes.
- **Metadata**: Structured data for introspection/observability.

**Cons**:

- Slightly more complex implementation.
- Builder adds verbosity for simple nodes (can provide helper constructors).

---

### 🎯 Recommendation: **Option C (Hybrid)**

Reasons:

1. **Composability**: Functions are easier to compose than interfaces.
2. **Safety**: Interface guarantees engine compatibility.
3. **DX**: Builder pattern is idiomatic Go and highly readable.
4. **Evolution**: Easy to add hooks (`OnPause`, `OnResume`) later without breaking changes.

---

## Scheduler Interface

```go
type Scheduler interface {
    // Next returns the next node to execute
    // Returns (nil, nil) when done
    // Returns (nil, err) on error
    // Returns (node, nil) to schedule next
    Next(ctx context.Context, current Node) (Node, error)
}

// Implementations:
type CronScheduler struct { ... }          // Time-based (life-dsl)
type StateMachineScheduler struct { ... }  // Transition-based (flow-dsl)
type SelectorScheduler struct { ... }      // DOM traversal (scrape-dsl)
type OnceScheduler struct { ... }          // Execute once, then done
type LoopScheduler struct { ... }          // Infinite loop with delay
```

---

## State Store Interface

```go
type StateStore interface {
    Get(ctx context.Context, nodeID string) (*NodeState, error)
    Set(ctx context.Context, nodeID string, state *NodeState) error
    List(ctx context.Context) ([]NodeState, error)
    
    // Checkpoint for durability
    Checkpoint(ctx context.Context) error
    Restore(ctx context.Context) error
}

// Implementations:
type MemoryStore struct { ... }     // For testing
type SQLiteStore struct { ... }     // For local persistence
type RedisStore struct { ... }      // For distributed systems
type LoamStore struct { ... }       // For filesystem (Obsidian vaults)
```

---

## DSL Compiler Interface

```go
type Compiler interface {
    // Parse takes raw config (YAML/JSON/Markdown)
    // Returns AST (abstract syntax tree)
    Parse(ctx context.Context, data []byte) (AST, error)
    
    // Compile converts AST to engine IR (Nodes)
    Compile(ctx context.Context, ast AST) ([]Executable, error)
}

// Example: flow-dsl
type FlowCompiler struct { ... }

func (c *FlowCompiler) Parse(ctx context.Context, data []byte) (AST, error) {
    var flow Flow
    if err := yaml.Unmarshal(data, &flow); err != nil {
        return nil, err
    }
    return &FlowAST{Flow: flow}, nil
}

func (c *FlowCompiler) Compile(ctx context.Context, ast AST) ([]Executable, error) {
    flowAST := ast.(*FlowAST)
    nodes := make([]Executable, 0, len(flowAST.States))
    
    for _, state := range flowAST.States {
        node := NewNode(state.ID).
            Execute(state.ActionFunc()).
            Schedule(StateMachineScheduler(state.Transitions)).
            Build()
        nodes = append(nodes, node)
    }
    return nodes, nil
}
```

---

## Execution Engine Core

```go
type Engine struct {
    store     StateStore
    scheduler Scheduler
    lifecycle *lifecycle.Router
}

func (e *Engine) Run(ctx context.Context, nodes []Executable) error {
    // Integrate with lifecycle for signals
    ctx = lifecycle.Attach(ctx, e.lifecycle)
    
    for {
        select {
        case <-ctx.Done():
            return e.gracefulShutdown(ctx)
        default:
            node, err := e.scheduler.Next(ctx, currentNode)
            if err != nil {
                return err
            }
            if node == nil {
                return nil // Done
            }
            
            // Execute node
            if err := node.Execute(ctx); err != nil {
                if handler := node.OnFailure; handler != nil {
                    if err := handler(ctx, err); err != nil {
                        return err
                    }
                }
            }
            
            // Checkpoint state
            if err := e.store.Set(ctx, node.ID(), &NodeState{
                LastRun: time.Now(),
                Status:  "completed",
            }); err != nil {
                return err
            }
        }
    }
}

func (e *Engine) gracefulShutdown(ctx context.Context) error {
    // Checkpoint current state
    return e.store.Checkpoint(ctx)
}
```

---

## DSL Examples: Same Engine, Different Syntax

### flow-dsl (Trellis)

```yaml
# workflow.yaml
flow:
  name: "Approve Payment"
  states:
    - id: pending_approval
      on_success: execute_payment
      on_failure: notify_admin
    
    - id: execute_payment
      action: call_api
      url: "https://bank.com/api/pay"
      on_success: done
      on_failure: rollback
```

**Compiles to**: `StateMachineScheduler` + `HTTPExecutor` nodes

---

### life-dsl (Life Management)

```yaml
# life.yaml
workers:
  - id: workout
    schedule: "0 7 * * *"  # Every day 7am
    actions:
      - type: notify
        message: "Time to workout!"
      - type: cli
        command: ["open", "fitness-app"]
    health_check:
      metric: consecutive_completions
      threshold: 3
      on_failure: escalate_to_supervisor
```

**Compiles to**: `CronScheduler` + `NotifyExecutor` + `CLIExecutor` nodes

---

### scrape-dsl (Web Automation)

```yaml
# scrape.yaml
selectors:
  - id: extract_price
    url: "https://shop.com/product"
    steps:
      - wait_for: "#price"
      - extract: ".value"
      - transform: parse_float
      - store: "product_price"
```

**Compiles to**: `SelectorScheduler` + `BrowserExecutor` nodes

---

## Ecosystem Convergence Timeline

### Phase 1: Foundation Stabilization (Current)

- **lifecycle** v1.5 ✅ (Control Plane stable)
- **loam** v0.10 ✅ (YAML/JSON/Markdown parser stable)
- **procio** v0.1 ✅ (Process primitives extracted)

### Phase 2: Engine Extraction (Next 4-6 weeks)

- **trellis-engine** v0.1.0
  - Extract generic execution primitives from Trellis
  - Node, Scheduler, StateStore interfaces
  - Lifecycle integration
  - Basic schedulers (Cron, StateMachine, Once)
  
- **trellis** v0.8.0 (refactored to use trellis-engine)
  - Backward compatible facade
  - Migrate to use trellis-engine internally
  - `flow-dsl` remains embedded for compatibility

### Phase 3: First Alternative DSL (6-8 weeks from now)

- **life-dsl** v0.1.0
  - Parser for `life.yaml`
  - Compiler to `trellis-engine` IR
  - Executors: CLI, HTTP, Notify, Browser
  - Lifecycle integration
  
- **POC**: Run `life.yaml` using trellis-engine

### Phase 4: DSL Ecosystem (3+ months)

- **flow-dsl** v1.0.0 (extracted from Trellis)
- **scrape-dsl** v0.1.0 (web automation)
- **deploy-dsl** v0.1.0 (infrastructure deployment)
- Community DSLs emerge

---

## Design Principles

### 1. Composition Over Inheritance

DSLs compose with engine primitives rather than subclassing.

### 2. Separation of Concerns

- **DSL**: Syntax, parsing, domain concepts
- **Engine**: Execution, scheduling, durability
- **Lifecycle**: Signals, I/O, control plane

### 3. Dependency Direction

```
DSL (flow-dsl, life-dsl, scrape-dsl)
  ↓ depends on
Engine (trellis-engine)
  ↓ depends on
Foundation (lifecycle, loam, procio)
```

### 4. Backward Compatibility

`trellis` package remains as facade for existing users.

### 5. Local-First

All DSLs support file-based configuration (YAML/JSON/Markdown).

---

## Integration Contracts

### For Each DSL to Implement

```go
// Required interfaces:
type Compiler interface {
    Parse(ctx context.Context, data []byte) (AST, error)
    Compile(ctx context.Context, ast AST) ([]Executable, error)
}

type Executable interface {
    ID() string
    Execute(ctx context.Context) error
    Scheduler() Scheduler
}

// Recommended patterns:
// - Builder API for node creation
// - Metadata for introspection
// - Lifecycle hooks (OnStart, OnFailure)
// - Error wrapping for observability
```

### For Engine to Provide

```go
// Core abstractions:
type Engine interface {
    Run(ctx context.Context, nodes []Executable) error
    Pause(ctx context.Context) error
    Resume(ctx context.Context) error
    Checkpoint(ctx context.Context) error
}

type Scheduler interface { ... }
type StateStore interface { ... }

// Utilities:
func WithLifecycle(router *lifecycle.Router) EngineOption
func WithStore(store StateStore) EngineOption
func WithMetrics(observer metrics.Observer) EngineOption
```

---

## Cross-Project Alignment

Each project in the ecosystem maintains a `docs/ECOSYSTEM_INTEGRATION.md` that documents:

1. **Current State**: What version, what dependencies.
2. **Integration Points**: How it uses/provides to other projects.
3. **Next Steps**: What must happen before deeper integration.
4. **Blockers**: What's waiting on another project.

This ensures continuity across chat sessions and development cycles.

---

## Next Steps

### Immediate (This Week)

- [ ] Document hybrid Node design in Trellis
- [ ] Prototype `trellis-engine` package structure
- [ ] Update Trellis PLANNING.md with extraction roadmap

### Short-Term (2-4 Weeks)

- [ ] Extract `trellis-engine` as sub-package
- [ ] Refactor Trellis to use engine internally
- [ ] Validate backward compatibility

### Medium-Term (1-2 Months)

- [ ] Create `life-dsl` POC
- [ ] Compile `life.yaml` → engine nodes
- [ ] Run POC with trellis-engine + lifecycle

### Long-Term (3+ Months)

- [ ] Promote engine to standalone repo (`trellis-engine`)
- [ ] Extract `flow-dsl` from Trellis
- [ ] Launch `scrape-dsl`
- [ ] Community DSLs

---

## Conclusion

By separating DSL from Engine, we unlock:

- **Reusability**: Write engine once, N DSLs benefit.
- **Composability**: DSLs can invoke each other (homogeneous IR).
- **Innovation**: New domains = new DSL, not new engine.
- **Philosophy**: True "Everything as Code" with shared foundation.

This is the evolution from **"Trellis for workflows"** to **"Engine for everything"**.
