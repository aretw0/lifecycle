# Refarm: Ecosystem Synergy Analysis

> **Purpose**: Document the relationship between lifecycle's Go ecosystem and Refarm's TypeScript/browser ecosystem.
>
> **Key Insight**: Concept export, not code export — architecture transfers, binaries don't.

---

## Project Identity

- **Name**: refarm
- **Version**: v0.0.x (active R&D)
- **Stack**: TypeScript, Astro, SQLite/OPFS, WASM Component Model, Nostr
- **Repository**: [github.com/aretw0/refarm](https://github.com/aretw0/refarm)
- **Role**: Personal Operating System for Sovereign Data (browser-first)

## Relationship to Lifecycle Ecosystem

Refarm is the first project in the ecosystem that **does not run on the OS**. It operates entirely in the browser. This creates a clear technology boundary:

```
┌───────────────────────────────────────────────────┐
│ Browser Domain (TypeScript)                       │
│                                                   │
│ Refarm Kernel ← Plugin orchestration              │
│ Refarm Studio ← Web IDE (Astro)                   │
│ @refarm/storage-sqlite ← OPFS/SQLite              │
│ @refarm/sync-crdt ← Real-time sync                │
│ @refarm/identity-nostr ← Sovereign identity       │
└───────────────────────┬───────────────────────────┘
                        │ WebSocket / Local HTTP (future)
┌───────────────────────▼───────────────────────────┐
│ OS Domain (Go)                                    │
│                                                   │
│ lifecycle ← Signal handling, graceful shutdown    │
│ procio ← Process hygiene, terminal I/O            │
│ Refarm Daemon (future) ← Local file indexing,     │
│   OS automation, native integrations              │
└───────────────────────────────────────────────────┘
```

### What Doesn't Transfer (Code)

- ❌ Go compiled to WASM in browser (heavy runtime, immature WIT support, goroutine conflicts)
- ❌ Direct library imports (different language, different runtime)
- ❌ procio/lifecycle in browser (no OS signals, no processes, no terminal I/O)

### What Transfers (Concepts)

| Go Source | Concept | Refarm Destination |
|---|---|---|
| **trellis** (engine) | DAG execution, worker state machines, plugin orchestration | `apps/kernel` — Plugin host design |
| **introspection** | State visualization, Mermaid diagrams, observer patterns | ADR-007 — Observer plugin interface |
| **lifecycle** (control plane) | Event-driven router, graceful shutdown, suspend/resume | Future Refarm Daemon (Go binary) |
| **procio** | Leak-free process management, platform hygiene | Future Refarm Daemon (Go binary) |
| **loam** | Reactive data stores, parser composition, Git audit trails | `packages/storage-sqlite` design patterns |

---

## Integration Points

### Current: Concept Transfer (Active)

The Go ecosystem serves as an **architecture proving ground**:

1. **Engine patterns**: Trellis validated DAG execution and state machine rigor. Refarm's Kernel benefits from these patterns without importing Go code.
2. **Observability**: Introspection proved that domain-agnostic visualization works. Refarm's ADR-007 adopts the same philosophy (primitives + pluggable observers).
3. **Configuration philosophy**: Loam's reactive embedded engine and Git audit trails inform how Refarm thinks about local-first data management.

### Future: Refarm Daemon (Planned)

When Refarm needs to access local OS resources (index PDFs, watch file changes, interact with native apps), the **Go ecosystem provides the foundation**:

- Built with `lifecycle.Run()` — leak-free, signal-aware
- Uses `procio` for cross-platform process management
- Exposes local API for browser Kernel to consume
- Follows lifecycle's design principles: library (not framework), zero domain opinions, Windows first-class

**This is where lifecycle and Refarm converge operationally**, not just conceptually.

### Anti-Pattern: Do NOT Force Go into Browser

The decision to **not** compile Go to WASM for browser execution was deliberate ([Refarm ADR-008](https://github.com/aretw0/refarm/blob/main/specs/ADRs/ADR-008-ecosystem-technology-boundary.md)):

- Go WASM binaries carry 5-15MB runtime overhead
- WASM Component Model support for Go is immature
- Goroutines conflict with browser's Event Loop
- The friction exceeds the benefit for Refarm's P&D stage

---

## What Lifecycle Provides to Refarm

| Capability | Usage | Status |
|---|---|---|
| **Signal handling** | Future Daemon: graceful shutdown | 🔮 Planned |
| **I/O resilience** | Future Daemon: non-blocking local file access | 🔮 Planned |
| **Supervisor** | Future Daemon: orchestrate background indexers | 🔮 Planned |
| **Router** | Future Daemon: control plane for local events | 🔮 Planned |
| **Architectural patterns** | Kernel design, ADR-007 observability | ✅ Active |

## What Refarm Provides to Lifecycle Ecosystem

| Capability | Value | Status |
|---|---|---|
| **Browser-first validation** | Proves ecosystem concepts work beyond Go/OS | ✅ Active |
| **WASM plugin model** | Informs trellis-engine plugin architecture | 🔮 Future |
| **Offline-first patterns** | Novel territory for the ecosystem (OPFS, CRDTs) | ✅ Active |
| **JSON-LD data model** | Richer semantic model than current YAML/Markdown | 🧪 R&D |

---

## Key Decisions

1. **Technology boundary is a feature**: TS for browser, Go for OS — each excels where it matters
2. **Concept export, not code export**: Architecture transfers, compiled artifacts don't
3. **Daemon is optional**: Refarm works 100% in browser; Go daemon unlocks OS-level superpowers
4. **Cross-reference documentation**: Both repos maintain awareness of each other via ADRs and ecosystem docs

---

**Last Updated**: 2026-03-05  
**Related ADR**: [Refarm ADR-008](https://github.com/aretw0/refarm/blob/main/specs/ADRs/ADR-008-ecosystem-technology-boundary.md)  
**Next Review**: When Refarm Daemon development begins
