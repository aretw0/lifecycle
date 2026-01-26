# BOOTSTRAP: lifecycle

Este repositório conterá a biblioteca central de gerenciamento de ciclo de vida e sinais para o ecossistema `aretw0` (Trellis, Tobot, Fiscus).

## Objetivo

Centralizar a lógica delicada de interrupção de processos (Graceful Shutdown) e I/O interativo que hoje reside no código interno do CLI do Trellis.

## Componentes a Extrair (Do Trellis)

1. **SignalContext**: Lógica de "Duplo Sinal" (SIGINT = Pausa/Cancelamento Suave, SIGTERM = Shutdown).
    * *Origem*: `trellis/internal/cli/helpers.go`
    * *Funcionalidade*: Deve diferenciar Ctrl+C (Interrupção de fluxo) de Kill request.

2. **InterruptibleReader**: Wrapper de I/O que permite cancelar leituras de Stdin sem bloquear a goroutine indefinidamente.
    * *Origem*: `trellis/internal/cli/helpers.go`

3. **TerminalReader**: Abstração para leitura de terminal que lida com idiossincrasias de SO.
    * *Origem*: `trellis/pkg/runner/text_handler.go` (Lógica `runtime.GOOS == "windows"`).
    * *Funcionalidade*: No Windows, deve usar `CONIN$` para garantir que o handle não feche prematuramente ao receber um Signal, permitindo que o `SignalContext` processe o evento antes do EOF fatal.

## Próximos Passos

* [ ] Inicializar Go Module (`go mod init github.com/aretw0/lifecycle`).
* [ ] Criar pacote `signal` e portar `SignalContext`.
* [ ] Criar pacote `io` (ou `term`) e implementar `NewTerminalReader` e `InterruptibleReader`.
* [ ] Adicionar testes unitários robustos (concorrência é difícil de testar).
