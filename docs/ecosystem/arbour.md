# Ecosystem Evolution: Arbour (Community Hub for Flow Library)

> **Status**: Clarified — Community Hub (2026+)  
> **Last Updated**: 2026-03-02

## Historical Context

**Arbour** originally started as a **WhatsApp ChatOps Assistant** (proof-of-concept) to demonstrate Trellis integration. However, deeper analysis revealed a far greater purpose: **The Package Manager and Community Hub for Flow Library**.

## Strategic Clarity

After detailed ecosystem analysis, **Arbour's True Purpose emerged**:

**Arbour = Community Hub for Flow Library & Ecosystem**

- **Package Manager** — `arbour install`, `arbour publish`, versioning, dependencies
- **Registry** — Marketplace of reusable flows (public, private, organizational)
- **CLI Tool** — Discovery, installation, composition, execution
- **Standard Library** — Core flows (templates, utilities) maintained by community
- **Execution Adapters** — WhatsApp, Telegram, Element, HTTP/CLI as optional plugins

## Lifecycle Integration (v1.5+)

- **Dependency**: Indirect (via `github.com/aretw0/trellis`)
- **Signal Handling**: Delegated to Trellis Runner
- **Benefit**: Zero-config robustness (WhatsApp reconnection, graceful shutdown)

### The Validation

Arbour validates the architectural decision to keep `lifecycle` as a foundational, transitive dependency:

- **Zero Config**: Arbour developers don't need to configure `lifecycle`. It "just works" because Trellis handles the signal context.
- **Version Propagation**: When Trellis upgrades to use `lifecycle` v1.5+, Arbour will automatically gain "Suspend/Resume" capabilities for WhatsApp connections without code changes.

## Strategic Value

Arbour is a testament to the "Core" philosophy: Solve the problem once in the foundation (Lifecycle/Trellis), and it's solved everywhere (Arbour, etc).

## Arbour's True Purpose: The Community Hub

Após análise profunda do Trellis (Platform Stack de 6 layers), ficou claro que Arbour tem um propósito MUITO MAIOR do que ser apenas um "WhatsApp bot".

### A Visão Original (Redescoberta)

Você tinha vislumbrado uma **comunidade criando fluxos e compartilhando-os**:

```
┌─────────────────────────────────────────────────────┐
│ Arbour: The Flows Community Hub                     │
├─────────────────────────────────────────────────────┤
│                                                      │
│  Package Manager       Registry/Marketplace         │
│  ↓                     ↓                             │
│  arbour install        arbour search                │
│  arbour publish        arbour browse                │
│                                                      │
│                  ↓                                   │
│  Shared Flow Ecosystem (npm for state machines)     │
│                                                      │
│  • Conversational flows (chatbots)                  │
│  • Business workflows (approvals, onboarding)       │
│  • Life automation (health, finance, learning)      │
│  • DevOps pipelines (deploy, monitor, rollback)    │
│  • Home automation (IoT, scheduling)                │
│                                                      │
│                  ↓                                   │
│         Execution Adapters (Plugins)                │
│  ┌──────────┬──────────┬──────────┬──────────┐     │
│  │ WhatsApp │ Telegram │ Element  │ HTTP CLI │     │
│  │ (plugin) │ (plugin) │ (plugin) │ (plugin) │     │
│  └──────────┴──────────┴──────────┴──────────┘     │
└─────────────────────────────────────────────────────┘
```

### O Que Arbour Realmente É

**Arbour = Community Hub for Flow Library & Ecosystem**

**Core Components**:

1. **Package Manager** — `arbour install <flow>`, `arbour publish`, versioning, dependencies
2. **Registry** — Marketplace of flows (public, private, organizational)
3. **CLI Tool** — Discovery, installation, composition, execution
4. **Standard Library** — Core flows (templates, utilities) maintained by community
5. **Execution Adapters** — WhatsApp, Telegram, Element, HTTP, CLI as **plugins**

### Por Que Isso Faz Sentido

#### ✅ Problema: Flow Libraries Precisam Existir

Você mencionou:
> "uma comunidade criando fluxos e compartilhando e instalando o fluxo dos outros para criar maiores"

**Trellis** resolve: *Como executar um fluxo (State Machine)*  
**Life-DSL** resolve: *Como descrever vida em código*  
**Arbour** resolve: **Como comunidades compartilham, descobrem e reutilizam fluxos**

#### ✅ Nome Forte Faz Sentido

"Arbour" (Abrigo/Árvore) é **perfeito** para um Package Manager:

- Lugar onde fluxos **crescem**
- Raízes conectadas (dependências)
- Comunidade se reúne

#### ✅ Adapters Como Plugins, Não Core

- **WhatsApp adapter** — Plugin opcional (não core do Arbour)
- **Telegram adapter** — Outro projeto (`telegraf`) pode ter seu próprio
- **Element adapter** — `element-executor` (third-party)
- **HTTP adapter** — Built-in (reference implementation)

**Cada projeto especializado mantém seu adapter**. Arbour não precisa de nomes como "arbour-whatsapp-plugin".

---

## O Que Sobrou para Arbour Ser

### ✅ Phase 1: Foundation (2026-03 → 2026-04)

**Objetivo**: Estabelecer Arbour como Package Manager + CLI Tool.

**Tasks**:

- [ ] **Flow Manifest Format** — Arquivo `flow.yaml` com metadados, dependências, entrada/saída esperada

  ```yaml
  name: "daily-health-check"
  description: "Asks about water intake, exercise, sleep"
  version: "0.1.0"
  dependencies:
    - "health/sleep-tracker:*"
    - "health/water-reminder:^1.0"
  input: { user_id: string }
  output: { health_score: int }
  ```

- [ ] **Package Registry** — Local registry + remote (GitHub-backed ou similar)
  - `~/.arbour/registry.yaml` (local cache)
  - `arbour remote add github https://github.com/aretw0/flows`

- [ ] **CLI Tools**:
  - `arbour search <query>` — Buscar fluxos
  - `arbour info <flow>` — Detalhes (versão, deps, autor)
  - `arbour install <flow[@version]>` — Baixar + cache
  - `arbour list` — Fluxos instalados localmente
  - `arbour run <flow> [--input '{}']` — Executar fluxo
  - `arbour publish <path>` — Publicar fluxo (push para registry)

- [ ] **Composition** — Permitir que fluxos usem outros fluxos

  ```yaml
  # sleep.yaml
  nodes:
    - id: check_duration
      type: tool
      do: "health/sleep-tracker"  # Referência a outro fluxo
      then: analyze_quality
  ```

**Deliverable**: `arbour init`, `arbour search`, `arbour install`, `arbour run` funcionando.

---

### 🔧 Phase 2: Standard Library (2026-04 → 2026-06)

**Objetivo**: Criar core set de flows reutilizáveis.

**Categories**:

- **Health** — Sleep tracking, water intake, exercise reminders
- **Productivity** — Time blocking, task breakdown, progress tracking
- **Finance** — Expense tracking, budget alerts, investment tracking
- **Learning** — Reading schedule, spaced repetition, project portfolio
- **DevOps** — Deployment workflows, monitoring, incident response
- **Home** — Temperature control, security checks, maintenance reminders

**Each flow**:

- ✅ Documentado (PRODUCT.md)
- ✅ Testado (unit + integration)
- ✅ Versionado (semantic versioning)
- ✅ Exemplo de uso (examples/)

**Deliverable**: 20-30 core flows forming a "Standard Library".

---

### 🚀 Phase 3: Execution Adapters (2026-06+)

**Objetivo**: Permitir que fluxos rodem em múltiplos canais.

**Adapter Pattern** (Standard Interface):

```go
// Qualquer protocolo implementa isso
type ExecutionAdapter interface {
    // Render node for this protocol
    Render(node, state) Response
    
    // Handle input for this protocol
    Navigate(state, input) (newState, nextNode)
    
    // Notify on completion/error
    Notify(status, message)
}
```

**Adapters** (Community-Driven):

- **HTTP Adapter** (Arbour Core) — REST API
- **WhatsApp Adapter** (by Arbour Team)— `arbour-whatsapp`
- **Telegram Adapter** (Community) — `telegraf-arbour` ou similar
- **Element Adapter** (Community) — `element-arbour-executor`
- **Discord Adapter** (Community) — `discord-flows`
- **Slack Adapter** (Community) — `slack-arbour`

**Key Point**: Adapters são **SEPARATE PROJECTS**, não parte do Arbour core. Arbour é agnóstico.

---

### 🌱 Phase 4: Community & Monetization (2027+)

**Objetivo**: Viabilizar uma comunidade auto-sustentável.

- [ ] **Arbour Registry (Web)** — Website para descobrir flows
- [ ] **Author Profiles** — Reconhecimento de criadores
- [ ] **Social Features** — Stars, forks, discussions
- [ ] **Monetization** (Optional):
  - Premium flows (author decides)
  - Support/training
  - Enterprise registry hosting
- [ ] **Governance** — Community standards, code of conduct

---

## Matriz de Responsabilidades (Reaprovisionada)

| Componente | Owner | Responsabilidade |
|-----------|-------|------------------|
| **Lifecycle** | aretw0 | Foundation (signals, I/O, supervision) |
| **Loam** | aretw0 | Data parsing (YAML/Markdown/JSON) |
| **Trellis** | aretw0 | State machine engine + HTTP/MCP adapters |
| **Life-DSL** | aretw0 | Life as Code (workers, schedules) |
| **Arbour** | aretw0 + Community | **Package manager, registry, CLI, standard library** |
| **arbour-whatsapp** | aretw0 | Reference WhatsApp adapter |
| **telegraf-arbour** | Community | Telegram adapter (third-party) |
| **element-executor** | Community | Matrix/Element adapter (third-party) |

---

## Por Que Isso Ganha

1. ✅ **Aligned with Vision** — Comunidade compartilhando fluxos (sua ideia original)
2. ✅ **Leverage Strong Name** — "Arbour" como hub de comunidade faz sentido perfeito
3. ✅ **Separation of Concerns** — Adapters são projetos independentes (WhatsApp é detail)
4. ✅ **Network Effects** — Mais fluxos publicados = mais valor para todos
5. ✅ **Sustainability** — Community-driven, potencial de monetização responsável
6. ✅ **Precedent** — npm (Node), pip (Python), cargo (Rust), Kubernetes (Helm charts)

---

## Next Steps (Immediate)

### **Esta Semana** (2026-03-02 → 2026-03-09)

1. ✅ **Vision Reframed** — Arbour como Community Hub (este documento)
2. ⏭️ **Flow Manifest Design** — Definir `flow.yaml` schema
3. ⏭️ **Registry Design** — Como armazenar/versionar fluxos
4. ⏭️ **CLI Prototype** — `arbour search`, `arbour install`

### **Próximas 4-6 Semanas** (Phase 1)

- Arbour Package Manager MVP
- Local registry + remote support
- `arbour install/run` funcionando

### **Próximas 12+ Semanas** (Phase 2)

- Standard Library de 20-30 flows
- Exemplos documentados
- Community contributions

---

## Conclusão: Arbour é o NPM de Fluxos

**Não é um "WhatsApp bot".**  
**É a plataforma que permite comunidades criarem uma biblioteca rich de fluxos reutilizáveis.**

WhatsApp é apenas um caso de uso. HTTP é outro. Telegram é um terceiro.

Arbour é o que conecta todos eles — **The Community Hub**. 🌳

---

**Última Atualização**: 2026-03-02 (Rediscovered)  
**Próxima Revisão**: Após Phase 1 (Package Manager MVP)  
**Tracking**: See [arbour/docs/PLANNING.md](https://github.com/aretw0/arbour/blob/main/docs/PLANNING.md) for detailed roadmap.
