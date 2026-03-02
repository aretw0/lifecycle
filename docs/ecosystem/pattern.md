# Universal Primitive Pattern: Serve Sozinho → Converge Emergentemente

## O Padrão

Este ecossistema é construído sobre **primitivos universais** que seguem um padrão consistente:

```
┌─────────────────────────────────────────────────────┐
│  Serve Sozinho (Standalone Value)                   │
│  ↓                                                  │
│  Converge Emergentemente (Ecosystem Synergy)        │
└─────────────────────────────────────────────────────┘
```

### Serve Sozinho

Cada primitivo é uma biblioteca **standalone** com valor independente:

- API clara e bem documentada
- Zero dependências no ecossistema
- Casos de uso externos são válidos e encorajados

### Converge Emergentemente

Quando primitivos se encontram no contexto de aplicações maiores, surgem sinergias naturais **sem acoplamento forçado**:

- lifecycle quer observabilidade → adota introspection
- trellis quer persistência → adota loam
- lifecycle quer process hygiene → adota procio

**Nenhum primitivo "conhece" os outros.** A convergência acontece nas camadas de aplicação.

## Primitivos do Ecossistema

### 1. **introspection** (Observability Primitive)

- **Standalone**: Generic Mermaid diagram builders para qualquer Go project
- **Convergência**: lifecycle, trellis, arbour querem visualizar topologias → adotam introspection naturalmente
- **Exemplo**: `lifecycle.SystemDiagram()` usa `introspection.ComponentDiagram()` sem forçar coupling

### 2. **loam** (Data Primitive)  

- **Standalone**: Embedded reactive & transactional engine com Git audit trail
- **Convergência**: trellis/arbour/Life-DSL querem "Everything as Code" → loam como data layer natural
- **Exemplo**: Trellis usa Loam para persistir estados de agentes (ADR 014), mas Loam continua útil fora do ecossistema

### 3. **procio** (Process/I-O Primitive)

- **Standalone**: Process hygiene (PDeathSig, Job Objects) + interactive I/O (CONIN$ on Windows)
- **Convergência**: lifecycle consuming tasks querem subprocess cleanup → procio como foundation
- **Exemplo**: lifecycle workers executando processos filhos usam procio, mas CLIs simples podem usar procio diretamente

### 4. **lifecycle** (Foundation)

- **Diferença**: lifecycle é o **integration point**, não um primitivo
- Consome primitivos (procio, introspection) e orquestra o control plane
- Outros apps (trellis, arbour) usam lifecycle como foundation, não como library

## Por Que Este Padrão?

### ✅ Vantagens

1. **Reuso Real**: Primitivos úteis fora do ecossistema = validação de design
2. **Baixo Acoplamento**: Mudanças em um primitivo não quebram outros
3. **Composição Natural**: Apps escolhem quais primitivos adotar
4. **Testabilidade**: Cada primitivo testável em isolamento

### ⚠️ Trade-offs

1. **Coordenação Manual**: Converging patterns não são automatizados (deliberado)
2. **Documentação Distribuída**: Cada primitivo documenta seu próprio uso
3. **Descoberta**: Devs precisam conhecer os primitivos disponíveis

## Quando Usar Este Padrão

✅ **Use quando**:

- Você quer criar algo reutilizável além do projeto atual
- O conceito tem valor standalone claro
- Acoplamento forçado traria mais custo que benefício

❌ **Evite quando**:

- O código é específico demais para um único app
- A abstração seria "premature optimization"
- A API standalone não faria sentido para outsiders

## Exemplos de Convergência

### Exemplo 1: lifecycle + introspection

```go
// introspection: generic diagram builder (standalone)
func ComponentDiagram(primary, secondary any, cfg *DiagramConfig) string

// lifecycle: domain adapter (convergence)
func SystemDiagram(sig SignalState, work WorkerState) string {
    return introspection.ComponentDiagram(sig, work, LifecycleDiagramConfig())
}
```

**Resultado**: introspection permanece generic, lifecycle ganha observability, sem coupling direto.

### Exemplo 2: trellis + loam

```go
// loam: transactional file storage (standalone)
type Engine interface { Read, Write, Watch, Transact }

// trellis: agent persistence (convergence)
type AgentStore struct { loam *Engine }
func (s *AgentStore) SaveState(agent Agent) { s.loam.Transact(...) }
```

**Resultado**: loam continua útil para CLIs, CMSs, config tools. Trellis ganha audit trail grátis.

## Anti-Pattern: Coupling Forçado

❌ **Evite**:

```go
// introspection depende de lifecycle
import "github.com/aretw0/lifecycle"
func Diagram() string { /* assume lifecycle running */ }
```

✅ **Prefira**:

```go
// introspection recebe apenas interfaces genéricas
func Diagram(primary any, cfg DiagramConfig) string
```

## Navegação

- [Introspection Analysis](introspection.md)
- [Loam Analysis](loam.md)
- [Procio Coverage](../TECHNICAL.md#6-process-hygiene-powered-by-procio)
- [Ecosystem README](README.md)
