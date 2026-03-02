# The Lifecycle Ecosystem: Everything as Code

> **Vision**: A foundation where applications differentiate between "User Interrupts" and "System Terminations", with robust infrastructure signaling and interactive I/O. Built as a modular, composable ecosystem.
>
> **Pattern**: [Serve Sozinho → Converge Emergentemente](pattern.md) — Universal primitives that work standalone but synergize naturally.

---

## 🌳 The Big Picture

```
┌─────────────────────────────────────────────────────────────────┐
│ Arbour: Community Hub & Package Manager                         │
│ (Fluxos compartilhados, descoberta, versioning)                │
└─────────────────────────────────────────────────────────────────┘
                    ↓ Executa via
┌─────────────────────────────────────────────────────────────────┐
│ Trellis: State Machine Platform (6 Layers)                      │
│ Layer 5: UI | Layer 4: Protocols | Layer 3: DX | Layer 2: DSLs │
│ Layer 1: Persistence | Layer 0: Engine                          │
└─────────────────────────────────────────────────────────────────┘
                    ↓ Usa
┌─────────────────────────────────────────────────────────────────┐
│ Loam: Data Parsing & Configuration                              │
│ (YAML/JSON/Markdown, Obsidian integration)                      │
└─────────────────────────────────────────────────────────────────┘
                    ↓ Usa
┌─────────────────────────────────────────────────────────────────┐
│ Lifecycle: Foundation Layer                                     │
│ (Signal handling, Terminal I/O, Graceful shutdown)             │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📚 Ecosystem Documentation

### **Foundation**

- [**lifecycle**](../../README.md) — Signal handling, I/O resilience, supervision primitives
- [**TECHNICAL.md**](../TECHNICAL.md) — Architecture deep-dive

### **Ecosystem Analysis**

- [**ECOSYSTEM_README.md**](README.md) ← You are here
- [**ecosystem_status.md**](ecosystem_status.md) — Executive summary: long-term goals per project
- [**ecosystem/**](.) — Detailed analysis per component

### **Components**

#### 🌲 **Loam** (Data Layer)

- **Purpose**: Parse YAML/JSON/Markdown into structured data
- **Pattern**: [Universal data primitive](pattern.md) — serves standalone, converges naturally
- **Docs**: [loam.md](loam.md) — Component overview & integration
- **Repo**: `github.com/aretw0/loam` (v0.10+)
- **Status**: ✅ Mature

#### 🏗️ **Trellis** (Logic Layer)

- **Purpose**: State machine engine with 6-layer platform stack
- **Key Insight**: Platform, not just engine — preserve all insights during refactoring
- **Docs**: [Refactoring Guide](https://github.com/aretw0/trellis/blob/main/docs/REFACTORING_GUIDE.md) — Refactoring roadmap with 10 preserved insights
- **Roadmap**: [Phases 2a/2b/2c](https://github.com/aretw0/trellis/blob/main/docs/PLANNING.md) — Incremental refactoring strategy
- **Repo**: `github.com/aretw0/trellis` (v0.7.15+)
- **Status**: ✅ In Production (Arbour)

#### 🌳 **Arbour** (Community Hub)

- **Purpose**: Package manager, registry, CLI for sharing Trellis-based flows
- **Key Insight**: NOT a "WhatsApp bot" — a community hub for flow library
- **Docs**: [ecosystem/arbour.md](arbour.md) — Vision, phases 1-4, roadmap
- **Repo**: `github.com/aretw0/arbour` (v0.1+)
- **Status**: 🧪 Strategic pivot completed, Phase 1 (Package Manager MVP) ready to start

#### 🧬 **Life-DSL** (Future)

- **Purpose**: "Life as Code" — Personal automation (workers, schedules, health)
- **Docs**: [ecosystem/PLANNING.md](../PLANNING.md) (Phase 2b — Life-DSL POC)
- **Status**: 🔮 Planned (Phase 2b: 2-3 weeks)
- **Blocker**: Trellis Phase 2a must complete first

#### 🧪 **Fiscus** (Experimental)

- **Purpose**: Financial/legal agreement workflows (Colang-inspired)
- **Docs**: [fiscus.md](fiscus.md)
- **Status**: 📋 Experimental, awaiting engine abstraction

#### ⚙️ **Procio** (OS Mechanics)

- **Purpose**: Leak-free process lifecycle + robust terminal I/O primitives
- **Pattern**: [Universal process primitive](pattern.md) — serves standalone, converges via lifecycle
- **Docs**: Covered in [TECHNICAL.md](../TECHNICAL.md#6-process-hygiene-powered-by-procio)
- **Repo**: `github.com/aretw0/procio` (v0.4+)
- **Status**: ✅ Stable, integrated by lifecycle

#### 🔍 **Introspection** (Observability)

- **Purpose**: Domain-agnostic state visualization, Mermaid diagrams, monitoring primitives
- **Pattern**: [Universal observability primitive](pattern.md) — every project wants to “see what’s happening”
- **Docs**: [introspection.md](introspection.md) — Component overview & adoption
- **Repo**: `github.com/aretw0/introspection` (v0.1.3)
- **Status**: ✅ Integrated with lifecycle, ready for Trellis/Arbour adoption

---

## 🎯 Key Decisions (2026-03-02)

### ✅ Trellis is a Platform, Not Just an Engine

- 6 layers: UI → Protocols → DX → DSLs → Persistence → Core Engine
- 11 valuable insights cataloged for preservation during refactoring
- Strategy: **Refactor → Validate → Extract** (not extract first)

### ✅ Arbour is the Community Hub

- Package manager for flows, not a "WhatsApp bot"
- Registry, discovery, installation, composition, execution
- WhatsApp/Telegram/Element are optional protocol adapters (plugins)

### ✅ Incremental Architecture Evolution

- **Phase 2a** (2-3 weeks): Trellis internal restructuring
- **Phase 2b** (2-3 weeks): Life-DSL POC validates what's generic
- **Phase 2c** (1-2 weeks): Extract only proven components (protocols, persistence, tooling)
- **Phase 3** (2-3 weeks): Life-DSL standalone repo
- **Phase 4** (Future): Extract Trellis-Engine (only after 2+ DSLs validated)

---

## 🗂️ Documentation Structure

### By Component

```
ecosystem/
├── pattern.md                        (Universal primitive pattern)
├── engine_abstraction.md             (Abstract engine vision)
├── ecosystem_status.md               (Executive summary: long-term goals)

├── loam.md                           (Data layer)
├── trellis.md                        (State machines)
├── arbour.md                         (ChatOps/Plugins)
├── fiscus.md                         (Experimental)
└── introspection.md                  (Observability)
```

### By Topic

```
docs/
├── PRODUCT.md                        (Vision: Everything as Code)
├── TECHNICAL.md                      (Lifecycle core)
├── PLANNING.md                       (Roadmap)
├── DECISIONS.md                      (Design decisions)
├── CONFIGURATION.md                  (Config philosophy)
├── RECIPES.md                        (Usage patterns)
├── TESTING.md                        (Testing philosophy)
└── ecosystem/
    ├── README.md                     (This file)
    └── ...
```

---

## 🚀 Getting Started

### For End Users

1. Start with [**lifecycle/README.md**](../../README.md) — understand the foundation
2. Check [**trellis/**](https://github.com/aretw0/trellis/blob/main/README.md) — use state machines to build flows
3. Join **Arbour** community — discover & share flows (coming in Phase 1)

### For Contributors

1. Review [**trellis_refactoring_handoff.md**](trellis_refactoring_handoff.md) — refactoring roadmap & preserved insights
2. Follow **Phase 2a** roadmap if contributing to Trellis
3. Use [**ECOSYSTEM_INTEGRATION.md**](../ECOSYSTEM_INTEGRATION.md) as template for new projects

### For Integrators

1. Read [**trellis/ECOSYSTEM_INTEGRATION.md**](https://github.com/aretw0/trellis/blob/main/docs/ECOSYSTEM_INTEGRATION.md) — dependencies & consumers
2. Check [**arbour.md**](arbour.md) — how Arbour consumes Trellis

---

## 💡 The Philosophy

### 1. **"Do One Thing Well"** (Unix Way)

- Lifecycle: Signals & I/O
- Loam: Parsing & Data
- Trellis: State Machines
- Arbour: Community & Package Management

### 2. **"Measure Twice, Cut Once"**

- Refactor before extracting
- Validate with empirical examples (life-dsl POC)
- Avoid premature abstraction

### 3. **"Everything as Code"**

- Infrastructure as code (workflows)
- Configuration as code (YAML/Markdown)
- Life as code (automation)

---

## 📊 Status Overview

| Component | Version | Status | Phase |
|-----------|---------|--------|-------|
| **Lifecycle** | 1.5+ | ✅ Production | Foundation |
| **Loam** | 0.10+ | ✅ Production | Data |
| **Trellis** | 0.7.15+ | ✅ Production | Phase 1 ✓ |
| **Trellis Phase 2a** | — | 🔧 Next | Internal Restructuring |
| **Arbour** | 0.1+ | 🧪 Pivot | Phase 1 Package Mgr |
| **Life-DSL** | — | 🔮 Planned | Phase 2b |
| **Fiscus** | — | 📋 Experimental | — |
| **Procio** | 0.4+ | ✅ Stable | OS Mechanics |
| **Introspection** | 0.1.3 | ✅ Integrated | — |

---

## 🎓 Learning Path

### Beginner

1. [lifecycle/README.md](../../README.md) — foundation
2. [trellis/examples/basic](https://github.com/aretw0/trellis/tree/main/examples/basic) — simple state machine

### Intermediate

1. [Trellis Refactoring Guide](https://github.com/aretw0/trellis/blob/main/docs/REFACTORING_GUIDE.md) — Refactoring strategy & 10 preserved insights
2. [engine_abstraction.md](./engine_abstraction.md) — Abstract engine vision
3. [introspection.md](./introspection.md) — observability patterns

### Advanced

1. [PLANNING.md](../PLANNING.md) — ecosystem roadmap (Phases 2a-4)
2. [trellis/ECOSYSTEM_INTEGRATION.md](https://github.com/aretw0/trellis/blob/main/docs/ECOSYSTEM_INTEGRATION.md) — component integration
3. Contribute to Phase 2a (Trellis restructuring) or Phase 1 (Arbour package manager)

---

## ❓ FAQ

**Q: Should I use Lifecycle, Loam, or Trellis?**  
A: Likely all three:

- Lifecycle = foundation (always transitive)
- Loam = if you parse YAML/Markdown
- Trellis = if you need state machines

**Q: Can I use Trellis without Loam?**  
A: Yes! Trellis has `MemoryLoader` for in-memory graphs. Loam is optional.

**Q: What about Arbour — is it required?**  
A: No. Arbour is for community flow sharing. You can use Trellis standalone.

**Q: When is Life-DSL coming?**  
A: After Trellis Phase 2a (internal restructuring) completes — likely April 2026.

**Q: Is Fiscus ready?**  
A: No, it's experimental. Wait for engine abstraction clarity (Phase 2c).

**Q: What is the "Serve Sozinho → Converge Emergentemente" pattern?**  
A: Ecosystem primitives (introspection, loam, procio) work standalone with zero coupling, but naturally synergize when combined in larger applications. See [pattern.md](pattern.md).

**Q: Do I need to use all primitives?**  
A: No. Pick what you need: lifecycle (always), loam (if parsing data), introspection (if visualizing state), procio (usually via lifecycle).

---

## 🔗 Quick Links

- **Lifecycle Repo**: [github.com/aretw0/lifecycle](https://github.com/aretw0/lifecycle)
- **Trellis Repo**: [github.com/aretw0/trellis](https://github.com/aretw0/trellis)
- **Loam Repo**: [github.com/aretw0/loam](https://github.com/aretw0/loam)
- **Procio Repo**: [github.com/aretw0/procio](https://github.com/aretw0/procio)
- **Arbour Repo**: [github.com/aretw0/arbour](https://github.com/aretw0/arbour)
- **Introspection Repo**: [github.com/aretw0/introspection](https://github.com/aretw0/introspection)

---

**Last Updated**: 2026-03-02  
**Curated By**: Ecosystem Analysis Session (2026-03-02)  
**Primary Maintainer**: @aretw0

---

**Next Steps**:

1. ✅ Correct defasações (done)
2. ⏭️ Create `arbour/docs/VISION.md` — sync with Arbour repo
3. ⏭️ Start Phase 2a (Trellis internal restructuring)
