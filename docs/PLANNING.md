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

- [x] **Windows CI**: Enable `windows-latest` runner in GitHub Actions to verify `CONIN$` support. (Critical)
- [ ] **Process Hygiene**: Abstração para `PDeathSig` (Linux) e `JobObjects` (Windows) para garantir que filhos morram com o pai (evitar zumbis).
- [x] **Shutdown Timeouts**: Add helpers for `Shutdown(ctx, timeout)` to prevent hangs.
- [ ] **Lifecycle Hooks**: A standard `OnShutdown(func())` registry to simplify consumer cleanup logic.
- [ ] **TermIO Automation Spike**: Design a specific plan/POC for testing blocking reads (complex scenario). Requires dedicated focus.

### v1.2: Ecosystem Convergence (The Supervisor Pattern)

Foco: Gerenciamento robusto de processos filhos e suporte a "Agents" (Trellis/Arbour-like).

- [ ] **Supervisor**: Implementação de um padrão Supervisor para gerenciar grupos de `exec.Cmd` (Restart policies, Group Shutdown).
- [ ] **Container Interface**: Definição de interfaces para orquestração de containers (sem dependência direta de Docker SDK).

### v1.3: Portability

- [ ] **BSD/Solaris Support**: Verify `termio.Open()` behavior on other Unixes.

## Backlog

- **Raw Mode Helpers**: Consider wrapping `x/term` Raw Mode enter/restore logic if it becomes repetitive across projects?
- **Observability Hooks**: Tracing/Logging interfaces for lifecycle events.

## Technical Debt

- **Test Coverage**: (Promoted to v1.1) `pkg/termio` relies heavily on manual verification for the "blocking read" scenarios.
- [x] **Refactoring**: Simplified `IsInterrupted` in `pkg/termio` to remove O(N^2) recursion and rely on standard `errors.Is`.
