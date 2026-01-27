# Planning: Lifecycle

## Roadmap

### v0.x: Foundation (Current)

- [x] **Signal Layer**: `pkg/signal` with `SignalContext` (Dual Signal support).
- [x] **I/O Layer**: `pkg/termio` with `InterruptibleReader` and Windows `CONIN$` support.
- [x] **Testing**: Unit tests for core logic.
- [x] **Release Automation**: GoReleaser configuration.

### v1.0: Stability (Release Candidate)

- [x] **Integration**: Adopted by `trellis` v0.8+.

### v1.1: Stewardship & Robustness

Foco: Garantir que o ciclo de vida seja resiliente, debugável e seguro.

- [x] **Process Hygiene**: Abstração para `PDeathSig` (Linux) e `JobObjects` (Windows) para garantir que filhos morram com o pai (evitar zumbis - "Fail-Closed").
- [x] **Shutdown Timeouts**: Add helpers for `Shutdown(ctx, timeout)` to prevent hangs.
- [x] **Observability**: `SetLogger` e `MetricsProvider` para monitoramento do ciclo de vida sem dependências externas.
- [ ] **Lifecycle Hooks**: A standard `OnShutdown(func())` registry to simplify consumer cleanup logic.
- [ ] **TermIO Automation Spike**: Design a specific plan/POC for testing blocking reads (complex scenario).

### v1.2: Ecosystem Convergence (The Supervisor Pattern)

Foco: Gerenciamento robusto de processos filhos e suporte a "Agents" (Trellis/Arbour-like).

- [ ] **Supervisor**: Implementação de um padrão Supervisor para gerenciar grupos de `exec.Cmd` (Restart policies, Group Shutdown).
- [ ] **Container Interface**: Definição de interfaces para orquestração de containers (sem dependência direta de Docker SDK).

### v1.3: Portability

- [ ] **BSD/Solaris Support**: Verify `termio.Open()` behavior on other Unixes.

## Backlog

- **Non-destructive I/O**: Research a "Buffered Peek" for `InterruptibleReader` to avoid data loss when context is cancelled during a read.
- **Raw Mode Helpers**: Consider wrapping `x/term` Raw Mode enter/restore logic if it becomes repetitive across projects?

## Technical Debt

- [x] **Test Coverage**: Cobertura robusta para `pkg/signal` e `pkg/metrics` usando mocks internos.
- [x] **Refactoring**: Simplified `IsInterrupted` em `pkg/termio` e refatoração "Sober" de `pkg/signal` para legibilidade.
