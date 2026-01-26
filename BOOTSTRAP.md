# BOOTSTRAP: lifecycle

Este repositório conterá a biblioteca central de gerenciamento de ciclo de vida e sinais para o ecossistema `aretw0` (Trellis, Tobot, Fiscus).

## Objetivo
Centralizar a lógica delicada de interrupção de processos (Graceful Shutdown) que hoje reside no código interno do CLI do Trellis.

## Componentes a Extrair (Do Trellis)
1.  **SignalContext**: Lógica de "Duplo Sinal" (SIGINT = Pausa/Cancelamento Suave, SIGTERM = Shutdown).
    *   *Origem*: `trellis/internal/cli/helpers.go`
2.  **InterruptibleReader**: Wrapper de I/O que permite cancelar leituras de Stdin sem bloquear a goroutine indefinidamente.
    *   *Origem*: `trellis/internal/cli/helpers.go`

## Próximos Passos
- [ ] Inicializar Go Module (`go mod init github.com/aretw0/lifecycle`).
- [ ] Criar pacote `signal` e portar `SignalContext`.
- [ ] Criar pacote `io` (ou `term`) e portar `InterruptibleReader`.
- [ ] Adicionar testes unitários robustos (concorrência é difícil de testar).
