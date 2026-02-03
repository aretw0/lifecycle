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
* **Exception**: Interactive Shells/REPLs. In these specific cases, developers **MUST** explicitly disable global handling (`signal.WithForceExit(0)`) and handle signals locally to avoid killing the session on `Ctrl+C`.

## ADR-0003: Managed Concurrency (Zero Config)

* **Status**: Accepted
* **Context**: Goroutine leaks occur when developers forget to `Wait()` on a `WaitGroup` or fail to propagate cancellation.
* **Decision**: `lifecycle.Go(ctx, fn)` automatically tracks goroutines. `lifecycle.Run` waits for all tracked goroutines to finish before returning.
* **Implementation Note**: Since ADR-0006, this is powered by context value discovery, ensuring it works even when the context is wrapped by telemetry/middle-tier providers.
* **Consequences**: Zero configuration required for safe concurrency.

## ADR-0004: Event-Driven Control Plane (v2.0)

* **Status**: Accepted
* **Context**: As the library evolves from "Death Management" to "Lifecycle Management", we need to handle non-terminal events (Reload, Suspend).
* **Decision**: Adopt an Event-Driven Architecture. Decouple **Sources** (Signals, Webhooks, Tickers) from **Handlers** via a standardized `Router`.
* **Consequences**: Allows for infinite extensibility without polluting the core `Run` loop.

## ADR-0005: Interactive Router Preset

* **Status**: Accepted
* **Context**: Setting up a robust interactive CLI (Standard signals + detached Stdin reader + common commands) requires significant boilerplate (~50 lines of wiring).
* **Decision**: Provide a `NewInteractiveRouter` preset that encapsulates standard source wiring (OS Signals, Input) and standard command routing (q/quit/suspend/resume).
* **Rationale**: Drastically improves Developer Experience (DX) and ensures consistency across tools in the ecosystem without sacrificing flexibility (configurable via options).

## ADR-0006: Context-Aware Signal Discovery (Pattern)

* **Status**: Accepted
* **Context**: Application contexts are often wrapped by middle-tier providers (e.g., Task Tracking, Tracing). Simple type assertions to `*signal.Context` fail in these scenarios, breaking core library features like `OnShutdown`.
* **Decision**: Implement a **Value-Based Discovery Path**. Use a private context key to store and retrieve the `signal.Context` pointer. Provide a robust `FromContext(ctx)` helper that handles both direct pointers and wrapped values.
* **Consequences**: Ensures library resilience when integrated with other heavy-weight frameworks or complex diagnostic wrappers.

## ADR-0007: Standardized Observation Metadata

* **Status**: Accepted
* **Context**: Introspection (Diagrams, Metrics, Logs) needs consistent keys (e.g., `restarts`, `circuit_breaker`) to provide a unified "Single Pane of Glass" view. Hardcoded strings across packages lead to drift and broken diagrams.
* **Decision**: Standardize metadata keys as **typed constants in `pkg/worker`**. All components (Supervisor, Diagram Engine, Metrics) must use these constants instead of literal strings.
* **Consequences**: Centralizes the introspection "schema", making it trivial to update the visual representation across all interfaces.

## ADR-0008: Programmatic Shutdown Facade

* **Status**: Accepted
* **Context**: Handlers and Jobs often need to trigger the same graceful termination sequence as an OS Signal (e.g., a "quit" command in a REPL).
* **Decision**: Provide an explicit `lifecycle.Shutdown(ctx)` facade.
* **Rationale**: This abstracts the complex context discovery and cancellation logic, providing a high-level API for internal application control that mirrors external signals.
