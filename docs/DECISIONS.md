# Architecture Decision Records (ADR)

This document logs significant architectural decisions for the `lifecycle` project.

## ADR-0001: Aggressive Default Safety (Fail-Closed)

* **Status**: Accepted
* **Context**: A common problem in Go/Docker environments is "Zombie Processes" &mdash; child processes that outlive their parents because the parent crashed or failed to signal them. This leads to resource leaks and operational headaches.
* **Decision**: `lifecycle` enforces a "Fail-Closed" philosophy. We use platform-specific mechanisms (Linux PDeathSig, Windows Job Objects) to guarantee that if the parent dies, the children die.
* **Consequences**: This behavior is enabled by default in `pkg/proc` and `pkg/supervisor`. It is effectively non-negotiable for the library's identity as a robust infrastructure tool.

## ADR-0002: Signal Handling Strategy (Implicit vs Explicit)

* **Status**: Accepted
* **Context**: Should the library automatically handle `SIGINT` (Ctrl+C) and `SIGTERM`?
* **Decision**: **Yes, by default (Imperial Default).**
* **Rationale**:
    1. **Safety**: Prevents beginners from creating unkillable processes.
    2. **Standards**: `SIGTERM` compliance is mandatory for Kubernetes/Docker.
    3. **Expectation**: For most Services and CLIs, `SIGINT` means "Stop", not "Clear line".
* **Exception**: Interactive Shells/REPLs. In these specific cases, developers **MUST** explicitly disable global handling (`signal.WithInterrupt(false)`) and handle signals locally to avoid killing the session on `Ctrl+C`.

## ADR-0003: Managed Concurrency (Zero Config)

* **Status**: Accepted
* **Context**: Goroutine leaks occur when developers forget to `Wait()` on a `WaitGroup` or fail to propagate cancellation.
* **Decision**: `lifecycle.Go(ctx, fn)` automatically tracks goroutines embedded in the Context. `lifecycle.Run` waits for all tracked goroutines to finish before returning.
* **Consequences**: Zero configuration required for safe concurrency.

## ADR-0004: Event-Driven Control Plane (v2.0)

* **Status**: Accepted
* **Context**: As the library evolves from "Death Management" to "Lifecycle Management", we need to handle non-terminal events (Reload, Suspend).
* **Decision**: Adopt an Event-Driven Architecture. Decouple **Sources** (Signals, Webhooks, Tickers) from **Handlers** via a standardized `Router`.
* **Consequences**: Allows for infinite extensibility without polluting the core `Run` loop.
