# Ecosystem Status: Objetivos de Longo Prazo

> **Última Atualização**: 2026-03-02  
> **Propósito**: Resumo executivo do que foi deixado em cada projeto para alcançar a visão "Everything as Code"

---

## 🎯 Visão de Longo Prazo (2026+)

**Objetivo Central**: Múltiplos DSLs compilando para um **Abstract Execution Engine** compartilhado, todos construídos sobre a fundação robusta do `lifecycle`.

```
┌────────────────────────────────────────────────────────┐
│ DSL Layer (Domain-Specific Syntax)                     │
│ • flow-dsl  (Workflows, state machines)                │
│ • life-dsl  (Habits, routines, energy management)      │
│ • scrape-dsl (Web automation)                          │
│ • deploy-dsl (Infrastructure)                          │
└─────────────────┬──────────────────────────────────────┘
                  │ Compile to IR
┌─────────────────▼──────────────────────────────────────┐
│ trellis-engine (Abstract Execution)                    │
│ • Node abstraction • Scheduler • State store           │
└─────────────────┬──────────────────────────────────────┘
                  │ Built on
┌─────────────────▼──────────────────────────────────────┐
│ lifecycle (Foundation)                                 │
│ • Signal handling • I/O resilience • Supervision       │
└────────────────────────────────────────────────────────┘
```

---

## 📊 Status por Projeto

### 1️⃣ **lifecycle** (Este Projeto)

**Papel**: Foundation Layer — Infrastructure robustness  
**Versão**: v1.7.2 (stable), v1.8.0 (in progress)  
**Status**: ✅ **Foundation Stabilization Complete**

#### O Que Está Pronto

- ✅ Control Plane (Router + Event Sources + Handlers)
- ✅ Managed Concurrency (Go, Supervisor, Group)
- ✅ Signal Differentiation (SIGINT ≠ SIGTERM)
- ✅ Platform-Specific I/O (Windows CONIN$ via procio)
- ✅ Durability Primitives (DoDetached, Grace Period)
- ✅ Observability (Metrics, Introspection, Mermaid diagrams)

#### Roadmap Documentado

- **v1.8**: Hardening (Test stability, Windows validation)
- **v1.9**: Community prep (Docs polish, Benchmarks, Case studies)
- **v1.10**: Public launch (Marketing, awesome-go submission)
- **v1.11+**: Feature expansion (OpenTelemetry, Testkit, Advanced behaviors)

#### Objetivos de Longo Prazo

1. Permanecer **domain-agnostic** (sem conhecer "flows", "tasks", "selectors")
2. Fornecer **primitives universais** para todos os DSLs
3. Manter **backward compatibility** (SemVer estrito)
4. Windows como **first-class citizen**

**Próxima Ação**: Aguardar adoção pelo Trellis (Phase 2)

---

### 2️⃣ **introspection**

**Papel**: Visualization Primitives (Mermaid diagrams)  
**Versão**: v0.1.3 (stable)  
**Status**: ✅ **Universal Primitive Ready**

#### O Que Foi Deixado

- ✅ Generic diagram builders (Tree, StateMachine, Component)
- ✅ Customization hooks (NodeStyler, NodeLabeler, PrimaryStyler)
- ✅ Type-safe configuration API
- ✅ Usado por lifecycle para `SystemDiagram()`

#### Objetivos de Longo Prazo

1. Servir como **visualization layer** para todo o ecossistema
2. Trellis, Arbour, Life-DSL reutilizam automaticamente
3. Manter **separation of concerns** (domain logic ≠ presentation)

**Próxima Ação**: Aguardar v0.2.0 com features adicionais (deprecar metadata bridge)

---

### 3️⃣ **procio**

**Papel**: Process Hygiene Primitives  
**Versão**: v0.1.2 (stable)  
**Status**: ✅ **Platform-Specific Foundation Complete**

#### O Que Foi Deixado

- ✅ PDeathSig (Linux/macOS) — Kill child if parent dies
- ✅ Job Objects (Windows) — Prevent zombie processes
- ✅ CONIN$ I/O (Windows) — Interactive terminal fixes
- ✅ Cross-platform terminal handling

#### Objetivos de Longo Prazo

1. Resolver **platform-specific quirks** uma vez
2. Lifecycle e outros projetos herdam automaticamente
3. Evitar reimplementação de raw syscalls

**Próxima Ação**: Stable — Aguardar feedback de adoção

---

### 4️⃣ **loam**

**Papel**: Data Layer (Parsing & Configuration)  
**Versão**: v0.10+ (stable)  
**Status**: ✅ **Integrated with lifecycle v1.7.2**

#### O Que Foi Deixado

- ✅ Embedded reactive & transactional engine
- ✅ YAML/JSON/Markdown parsing
- ✅ Git audit trail integration
- ✅ Obsidian compatibility
- ✅ **lifecycle.Run() integration** in CLI entrypoint

#### Integração Atual

**Loam já usa lifecycle**:

1. Direct dependency: `lifecycle v1.7.2` in go.mod
2. CLI entry: `lifecycle.Run(lifecycle.Job(...))` in cmd/loam/main.go
3. Graceful shutdown: Signals handled by lifecycle
4. Automatic watcher cleanup on shutdown

#### Próximos Passos Opcionais

**Oportunidades futuras** (low priority):

- Deeper Router integration (file watch events → Router)
- Supervisor for long-running watchers
- Suspend/Resume for interactive mode

**Prioridade**: Baixa (já funciona bem com básico lifecycle)

#### Objetivos de Longo Prazo

1. **ADR 013/014**: Demand-Driven Adapters (autonomia + sinergias)
2. Trellis como **primary stakeholder** (consume loam para parsing)
3. Continue benefiting from lifecycle improvements automatically

**Próxima Ação**: Monitor lifecycle upgrades, adopt new features as needed

---

### 5️⃣ **trellis**

**Papel**: Platform Stack (6 Layers) → Evoluir para Abstract Engine  
**Versão**: v0.7.1 (usando lifecycle v0.1.1 — outdated)  
**Status**: 🚧 **Strategic Refactoring in Progress**

#### O Que Foi Deixado

**10 Critical Insights Preserved** (ver `trellis_refactoring_handoff.md`):

1. Hypertext State Machine (Markdown links = transitions)
2. File-System Routing (repo/about.md → /about node)
3. MCP Server (AI-first orchestration)
4. Chat Web UI + SSE Delta Patching
5. Polyglot Tool Registry (.sh, .py, .js, .ps1)
6. Session Management & SAGA patterns
7. Schema Validation (context_schema)
8. Type-Safe Builders (Go DSL)
9. Signal Differentiation (lifecycle v1.5+)
10. Entrypoint Fallback (start → main → index)

**Refactoring Strategy Documented** (3 Phases):

- **Phase 2a**: Internal Restructuring (2-3 weeks)  
  → Organize layers `engine/`, `flow/`, `protocols/`, `persistence/`, `tooling/`, `ui/`

- **Phase 2b**: Life-DSL POC (2-3 weeks)  
  → Validate engine abstraction with second DSL

- **Phase 2c**: Package Extraction (2-3 weeks)  
  → Extract `trellis-engine`, `flow-dsl`, `trellis-protocols`

#### Blocker Crítico

**Node Abstraction Design Decision** (Aguardando mantenedor Trellis):

- Option A: Function-based (simplest)
- Option B: Interface-based (extensible)
- Option C: Hybrid (recommended in `engine_abstraction.md`)

**Recommendation**: **Hybrid** (functions for simple cases, interfaces for advanced)

#### Objetivos de Longo Prazo

1. Separar **DSL syntax** (flow-specific) de **Engine execution** (generic)
2. Upgrade para **lifecycle v1.7+** (Control Plane + Suspend/Resume)
3. Servir como **first realization** do Abstract Engine vision
4. Validar arquitetura com **life-dsl** (Phase 2b)

**Próxima Ação**: **🔴 URGENT** — Mantenedor Trellis decidir Node abstraction (esta semana)

---

### 6️⃣ **arbour**

**Papel**: Community Hub & Package Manager ("npm for flows")  
**Versão**: v0.7.x (indirect via Trellis)  
**Status**: ⏳ **Strategic Clarity Achieved — Awaiting Phase 3**

#### O Que Foi Deixado

**Strategic Vision Clarified**:

```
┌───────────────────────────────────────────────────┐
│ Arbour = Package Manager + Registry + CLI        │
├───────────────────────────────────────────────────┤
│ • arbour install <flow>                           │
│ • arbour publish (community flows)                │
│ • arbour search                                   │
│ • Registry/Marketplace (public/private/org)      │
│ • Standard Library (core templates)              │
│ • Execution Adapters (WhatsApp, Telegram, etc.)  │
└───────────────────────────────────────────────────┘
```

**Core Components Defined**:

1. Package Manager (versioning, dependencies)
2. Registry (marketplace of flows)
3. CLI Tool (discovery, installation, execution)
4. Standard Library (community-maintained)
5. Execution Adapters as **plugins** (WhatsApp, Telegram, Element, HTTP)

#### Integration Path

- **Current**: Indirect lifecycle via Trellis
- **Future**: Arbour CLI uses `trellis-engine` + `lifecycle` directly

#### Objetivos de Longo Prazo

1. **Problema a Resolver**: "comunidade criando fluxos e compartilhando"
2. Flows reutilizáveis cross-domain (life, automation, DevOps, etc.)
3. Plugin architecture para execution adapters
4. Versioning semântico para flows

**Próxima Ação**: Aguardar Phase 3 (após trellis-engine extraction)

---

### 7️⃣ **fiscus**

**Papel**: Reference Implementation (Lifecycle-First Architecture)  
**Versão**: v0.0.x (stub/greenfield)  
**Status**: 🆕 **Perfect Opportunity for Native Integration**

#### O Que Foi Deixado

**Blueprint Defined**:

1. Start com `lifecycle.Run()` desde commit #1
2. Domain events implementam `lifecycle.Introspectable`
3. Strict usage de `lifecycle.Context` para DB/API calls
4. Showcase de best practices desde o início

**Diferencial**:

- **Loam**: Retroactive adoption (já existe)
- **Arbour**: Indirect (via Trellis)
- **Fiscus**: **Native** (lifecycle-first desde dia 1)

#### Objetivos de Longo Prazo

1. Demonstrar como construir **sovereign application** corretamente
2. Servir como **reference implementation** em exemplos
3. Validar APIs do lifecycle em contexto real

**Próxima Ação**: Aguardar início do desenvolvimento

---

### 8️⃣ **life-dsl** (Futuro)

**Papel**: First Alternative DSL (Validation of Engine Abstraction)  
**Versão**: v0.0.x (não existe ainda)  
**Status**: 📅 **Phase 3 (6-8 weeks from now)**

#### O Que Foi Documentado

**Vision**:

```yaml
# life.yaml
workers:
  - id: workout
    schedule: "0 7 * * *"  # Daily 7am
    tool: notify
    args: ["Time to workout!"]
    
  - id: rest_reminder
    schedule: "0 22 * * *"  # Daily 10pm
    tool: notify
    args: ["Wind down for sleep"]
```

**Purpose**: Validar que `trellis-engine` é verdadeiramente **domain-agnostic**.

**Success Criteria**:

- life-dsl compila para mesma IR que flow-dsl
- Reusa scheduler, state store, persistence do engine
- Integração profunda com lifecycle (Router, Supervisor, Context)

**Blocker**: Aguardar trellis-engine extraction (Phase 2)

#### Objetivos de Longo Prazo

1. Provar viabilidade de **múltiplos DSLs** → shared engine
2. Validar design decisions (Node abstraction, IR)
3. Showcase de "Everything as Code" vision

**Próxima Ação**: POC em `examples/life-dsl/` após Phase 2 completar

---

### 9️⃣ **trellis-engine** (Futuro)

**Papel**: Abstract Execution Engine (Generic Primitives)  
**Versão**: v0.0.x (não existe — será extraído)  
**Status**: 📅 **Phase 2 (4-6 weeks) — BLOCKED**

#### O Que Foi Documentado

**Core Interfaces** (ver `engine_abstraction.md`):

```go
// Abstract Node representation
type Executable interface {
    ID() string
    Execute(ctx context.Context) error
}

// Scheduler interface (generic)
type Scheduler interface {
    Schedule(node Executable, trigger Trigger) error
    Next(ctx context.Context) (Executable, error)
}

// State Store (persistence)
type StateStore interface {
    Save(id string, data any) error
    Load(id string) (any, error)
}

// Engine integra lifecycle profundamente
type Engine struct {
    router     *lifecycle.Router
    supervisor *lifecycle.Supervisor
    store      StateStore
}
```

**Naming Strategy**: **Hybrid Branding**

- `trellis-engine` (core generic)
- `trellis` (backward compat facade)
- `flow-dsl` (original Trellis syntax)
- `life-dsl`, `scrape-dsl` (new domains)

**Recommendation**: **Hybrid Node Abstraction** (functions + interfaces)

#### Integration Layers

1. **Router Integration**: Control events (reload, suspend, health)
2. **Supervisor Integration**: Node orchestration (auto-restart)
3. **Context Propagation**: Cancellation signals
4. **Durability**: Checkpoint/resume via lifecycle primitives

#### Blocker Crítico

**🔴 DECISION NEEDED**: Node Abstraction Design (Function vs Interface vs Hybrid)

**Owner**: Trellis maintainer  
**ETA**: Esta semana  
**Impact**: Unblocks todo Phase 2

#### Objetivos de Longo Prazo

1. Fornecer **universal primitives** para N DSLs
2. Integração profunda com lifecycle (Router, Supervisor, Context)
3. Todos DSLs **herdam robustness** automaticamente
4. Community pode criar custom DSLs com minimal friction

**Próxima Ação**: **🔴 URGENT** — Trellis maintainer tomar decisão

---

## 📋 Ecosystem Phases Overview

### ✅ Phase 1: Foundation Stabilization (Complete)

- lifecycle v1.5+ stable
- Control Plane ready
- Documentation complete
- Examples demonstrate patterns

### 🚧 Phase 2: Engine Extraction (4-6 weeks) — BLOCKED

**Goal**: Separar Trellis DSL de generic engine

**Blocker**: Node abstraction design decision

**Deliverables**:

- [ ] `trellis-engine` package extracted
- [ ] Trellis refactored to use engine
- [ ] Deep lifecycle integration (Router + Supervisor + Context)

**Owner**: Trellis maintainer

### 📅 Phase 3: First Alternative DSL (6-8 weeks)

**Goal**: Validar engine abstraction com life-dsl

**Dependencies**: Phase 2 must complete

**Deliverables**:

- [ ] `life-dsl` v0.1.0 (parser + compiler)
- [ ] POC running on trellis-engine
- [ ] Executors: CLI, HTTP, Browser, Notify

### 🔮 Phase 4: DSL Ecosystem (3+ months)

**Goal**: Enable community DSLs

**Deliverables**:

- [ ] `flow-dsl` extracted from Trellis
- [ ] `scrape-dsl` for web automation
- [ ] Plugin architecture stabilized
- [ ] Arbour as community hub operational

---

## 🎯 Success Criteria (Long-Term)

- [ ] **3+ DSLs** compile to shared engine
- [ ] Engine **deeply integrates** lifecycle (Router, Supervisor, Context)
- [ ] All DSLs **benefit from lifecycle improvements** automatically
- [ ] **Community can build** custom DSLs with minimal friction
- [ ] **Arbour marketplace** has 50+ community flows
- [ ] **100+ GitHub stars** on lifecycle
- [ ] **5+ projects** publicly using lifecycle

---

## 🚨 Critical Path (Next 2 Weeks)

### Immediate Blockers

1. **🔴 URGENT**: Trellis Node Abstraction Decision
   - Owner: Trellis maintainer
   - Recommendation: Hybrid (see `engine_abstraction.md`)
   - ETA: This week
   - Impact: Unblocks all of Phase 2

### Parallel Tracks (Non-Blocking)

2. **lifecycle v1.8 Hardening**:
   - Test stability (eliminate time.Sleep)
   - Windows platform validation
   - CI improvements

3. **Documentation Maintenance**:
   - Keep `ECOSYSTEM_INTEGRATION.md` updated
   - Transfer `trellis_refactoring_handoff.md` to Trellis repo

---

## 📞 Communication Protocol

### Cross-Project ECOSYSTEM_INTEGRATION.md

Cada projeto deve manter este documento respondendo:

1. O que este projeto fornece?
2. O que este projeto consome?
3. O que está nos bloqueando?
4. Qual o próximo milestone?

**Template**: Use `lifecycle/docs/ECOSYSTEM_INTEGRATION.md` como base.

### Dependency Updates

Quando lifecycle lançar nova versão:

1. Atualizar CHANGELOG.md (breaking changes)
2. Notificar projetos dependentes via GitHub issues
3. Fornecer migration guides em `docs/`

### Design Discussions

Mudanças arquiteturais maiores documentadas em `docs/ecosystem/` **antes** de implementação.

---

## 🎓 Key Documents

### Foundation Layer

- [lifecycle/AGENTS.md](../../AGENTS.md) — Project overview
- [lifecycle/docs/TECHNICAL.md](../TECHNICAL.md) — Architecture deep-dive
- [lifecycle/docs/PLANNING.md](../PLANNING.md) — Roadmap v1.8+

### Ecosystem Vision

- [ECOSYSTEM_INTEGRATION.md](../ECOSYSTEM_INTEGRATION.md) — Integration status per project
- [engine_abstraction.md](./engine_abstraction.md) — Abstract engine architecture
- [pattern.md](./pattern.md) — Universal primitive pattern ("Serve Sozinho → Converge")

### Component Analysis

- [introspection.md](./introspection.md) — Visualization primitives
- [loam.md](./loam.md) — Data layer (retroactive adoption)
- [trellis.md](./trellis.md) — Platform evolution
- [arbour.md](./arbour.md) — Community hub vision
- [fiscus.md](./fiscus.md) — Reference implementation blueprint

### Related

- [Trellis Refactoring Guide](https://github.com/aretw0/trellis/blob/main/docs/REFACTORING_GUIDE.md) — 10 critical insights (now in Trellis repo)

---

**Last Updated**: 2026-03-02  
**Next Review**: After Trellis Node Abstraction decision  
**Maintainer**: lifecycle team (use as coordination hub)
