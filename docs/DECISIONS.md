# Architecture Decision Records (ADR)

This document logs significant architectural decisions for the `lifecycle` project.

## ADR-0001: Aggressive Default Safety (Fail-Closed)

* **Status**: Accepted
* **Context**: A common problem in Go/Docker environments is "Zombie Processes" &mdash; child processes that outlive their parents because the parent crashed or failed to signal them. This leads to resource leaks and operational headaches.
* **Decision**: `lifecycle` delegates low-level process guarantees to the **`procio`** library. We use platform-specific mechanisms (Linux PDeathSig, Windows Job Objects) to guarantee that if the parent dies, the children die.
* **Consequences**: This behavior is enabled by default in `pkg/supervisor` (via `procio/proc`). It is effectively non-negotiable for the library's identity.

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

## ADR-0004: Event-Driven Control Plane (v1.5)

* **Status**: Accepted
* **Context**: As the library evolves from "Death Management" to "Lifecycle Management", we need to handle non-terminal events (Reload, Suspend).
* **Decision**: Adopt an Event-Driven Architecture. Decouple **Sources** (Signals, Webhooks, Tickers) from **Handlers** via a standardized `Router`.
* **Consequences**: Allows for infinite extensibility without polluting the core `Run` loop.
* **Note**: Originally planned for a "v2.0" major version, this was released as v1.5 to avoid `go.mod` migration overhead. See [MIGRATION.md](MIGRATION.md) for breaking changes.

## ADR-0005: Synchronization Pattern with Helpers

* **Status**: Accepted
* **Context**: Manual use of locks in workers generated risks of double unlocks, deadlocks, and repetitive code.
* **Decision**: Standardize the use of the `withLock` and `withLockResult` helpers for all concurrent state manipulation in workers.
* **Exception**: Methods that already perform locking internally (e.g., `ExportState`) should not be wrapped by these helpers.
* **Consequences**: Safer, more readable, and easier-to-maintain code. Reduction of concurrency bugs.
* **Reference**: Details and examples in [TECHNICAL.md](TECHNICAL.md#d-synchronization-with-mutex).

## ADR-0006: Interactive Router Preset

* **Status**: Accepted
* **Context**: Setting up a robust interactive CLI (Standard signals + detached Stdin reader + common commands) requires significant boilerplate (~50 lines of wiring).
* **Decision**: Provide a `NewInteractiveRouter` preset that encapsulates standard source wiring (OS Signals, Input) and standard command routing (q/quit/suspend/resume).
* **Rationale**: Drastically improves Developer Experience (DX) and ensures consistency across tools in the ecosystem without sacrificing flexibility (configurable via options).

## ADR-0007: Context-Aware Signal Discovery (Pattern)

* **Status**: Accepted
* **Context**: Application contexts are often wrapped by middle-tier providers (e.g., Task Tracking, Tracing). Simple type assertions to `*signal.Context` fail in these scenarios, breaking core library features like `OnShutdown`.
* **Decision**: Implement a **Value-Based Discovery Path**. Use a private context key to store and retrieve the `signal.Context` pointer. Provide a robust `FromContext(ctx)` helper that handles both direct pointers and wrapped values.
* **Consequences**: Ensures library resilience when integrated with other heavy-weight frameworks or complex diagnostic wrappers.

## ADR-0008: Standardized Observation Metadata

* **Status**: Accepted
* **Context**: Introspection (Diagrams, Metrics, Logs) needs consistent keys (e.g., `restarts`, `circuit_breaker`) to provide a unified "Single Pane of Glass" view. Hardcoded strings across packages lead to drift and broken diagrams.
* **Decision**: Standardize metadata keys as **typed constants in `pkg/worker`**. All components (Supervisor, Diagram Engine, Metrics) must use these constants instead of literal strings.
* **Consequences**: Centralizes the introspection "schema", making it trivial to update the visual representation across all interfaces.

## ADR-0009: Programmatic Shutdown Facade

* **Status**: Accepted
* **Context**: Handlers and Jobs often need to trigger the same graceful termination sequence as an OS Signal (e.g., a "quit" command in a REPL).
* **Decision**: Provide an explicit `lifecycle.Shutdown(ctx)` facade.
* **Rationale**: This abstracts the complex context discovery and cancellation logic, providing a high-level API for internal application control that mirrors external signals.

## ADR-0010: Sequential Control Plane Hooks (FIFO)

* **Status**: Accepted
* **Context**: Complex state transitions (like `Suspend`) often involve multiple actors: workers pausing, state being persisted, and UIs reporting progress.
* **Decision**: `SuspendHandler` (and related control plane actors) must execute hooks **Sequentially** and in **FIFO** order.
* **Rationale**: This enables a "Final State" reporting pattern. By registering functional components (supervisors, workers) *before* UI reporting hooks, we guarantee that UI messages like "SYSTEM SUSPENDED" only appear *after* the heavy components have successfully blocked and confirmed their state.
* **Consequences**: Developers must be mindful of registration order for UI accuracy. Functional work comes first; reporting comes last.

## ADR-0011: Internal Decoupling & Primitive Promotion

* **Status**: **Completed (2026-02-13)** via `github.com/aretw0/procio`
* **Context**: The `lifecycle` library evolved into a comprehensive control plane, but its core primitives (Process hygiene, I/O) are valuable optimization layers for any Go program.
* **Decision**: We extracted `proc`, `termio`, and `scan` into **`procio`** (Process I/O), a standalone library with zero dependencies. `lifecycle` now consumes `procio` to provide its high-level guarantees.
* **Rationale**:
    1. **Adoption**: `procio` solves universal Go problems (Zombie processes, Windows Stdin) without the framework weight of `lifecycle`.
    2. **Separation of Concerns**: `procio` handles "OS Mechanics"; `lifecycle` handles "Application Policies".
* **Consequences**: `pkg/core/proc` and `pkg/core/termio` logic now lives in `procio`. `lifecycle` acts as the policy engine driving these primitives.

## ADR-0012: Visualization Decoupling & Primitive Promotion

* **Status**: **Completed (2026-02-15)** via `github.com/aretw0/introspection`
* **Context**: The `lifecycle` library provides runtime introspection via `State()` methods and visualizes topology using Mermaid diagrams. Originally, each package (`signal`, `worker`, `supervisor`) contained custom Mermaid string concatenation logic, leading to redundancy, rigidity, and increased testing burden.
* **Decision**: We extracted generic diagram rendering primitives into **`introspection`**, a standalone library. `lifecycle` now provides **domain-specific styling logic** (`NodeStyler`, `PrimaryStyler`) and delegates **structural rendering** (Mermaid syntax, graph traversal) to `introspection`.
* **Rationale**:
    1. **DRY Principle**: Rendering logic is centralized, not duplicated across multiple packages.
    2. **Reusability**: Other projects (e.g., `trellis`, `arbour`) can use `introspection` for their own topologies.
    3. **Separation of Concerns**: `introspection` handles generic graph rendering; `lifecycle` handles domain semantics (status colors, labels).
    4. **Maintainability**: Visual improvements or Mermaid syntax changes happen in one place.
* **Consequences**:
  * Removed `pkg/core/introspection` package (~1500 lines).
  * Introduced `diagram_config.go` (centralized configuration adapter).
  * Simplified `signal/diagram.go` and `worker/diagram.go` by removing manual fragment rendering functions.
  * `lifecycle` now depends on `github.com/aretw0/introspection` v0.1.2+.

## ADR-0013: Delegation over Source Bloating

* **Status**: **Accepted (v1.7.0)**
* **Context**: Sources like `FileWatchSource` needed to support features like "Debouncing", "Project Awareness" (ignoring `.git`), and "Synchronous Data Extraction" (Pushing to Go channels instead of relying purely on Router callbacks).
* **Decision**: We keep `Sources` structurally dumb and generic, pushing business logic (filtering, debouncing) into the **Control Plane** via `Options`, `Middleware`, and `Bridges`.
* **Rationale**:
    1. **Composability**: A `DebounceHandler` can be used to throttle *any* rapid event (like `WebhookSource` bursts), not just file events. If we baked debouncing into `FileWatchSource`, we'd have to rewrite it for everything else.
    2. **Idiomatic Go**: Instead of forcing applications to invert their control flow (callbacks only), `events.Notify(ch)` acts as a bridge, allowing consumers to use traditional `select` or `for range` loops over standard `channels` when dealing with the lifecycle router.
* **Consequences**:
  * Users are responsible for "snapping together" pieces (e.g., combining `WithFilter` and `DebounceHandler`).
  * `lifecycle` remains a toolkit of orthogonal primitives rather than a rigid framework.

## ADR-0014: Durable Extension for the Event Router (Planned)

* **Status**: Proposed (Target: v1.8+)
* **Context**: The `lifecycle` Event Router currently handles transient signals and memory-based callbacks. To serve as a robust Event Broker for ecosystem projects (like `loam` or `trellis`), it must support events that survive reboots.
* **Decision**: Add extension points to the `Router` to support "Durable Sinks" without polluting the core API. The engine remains simple but allows state resumption from a persisted event stream.
* **Rationale**: Inspired by distributed workflow engines, this allows temporal decoupling.
* **Consequences**: Enables `lifecycle` to back orchestrators that require pause/resume semantics for long-running processes over days/weeks.

## ADR-0015: Worker Role Grouping (Planned)

* **Status**: Proposed (Target: v1.8+)
* **Context**: Scaling workers currently treats all workers equally. In distributed environments, leader election or targeted scale-down requires identifying workers by their function.
* **Decision**: Incorporate a concept of "Roles" into the `Supervisor`. e.g., `supervisor.AddWorker(w, role="background-sync")`.
* **Rationale**: Required for declarative stability and leader election, allowing nodes to dynamically enable/disable specific roles based on cluster consensus.
* **Consequences**: `lifecycle` steps closer to being a distributed control plane primitive rather than just a local process manager.

## ADR-0016: Chained Contexts & Orphan Prevention

**Date**: 2026-02-22

* **Status**: Accepted
* **Context**: Reviewing highly robust automation drivers like `go-rod/rod` reinforces the importance of "Chained Contexts" (contexts that intrinsically carry their own cancellation timeout/deadline specific to an action, without bleeding into parent lifecycles) and rigorous "Zombie Process Prevention" (cleaning up browser instances).
* **Decision**:
    1. **Strict Context Propagation**: `lifecycle` will enforce that all long-running or external processes MUST accept a context derived from the specific Action/Worker, rather than relying solely on the global `Router` context.
    2. **Deep Process Hygiene (`procio` validation)**: We reaffirm ADR-0001, but extend it: `procio` integration MUST be routinely audited against "browser-like" or "daemon-like" child processes that actively try to detach. We will use `lifecycle` as the control plane to ensure even detached children created by WebDrivers or sub-shells are aggressively reaped upon lifecycle termination.
* **Rationale**: Validating our architecture against community standards (like `go-rod`'s process management) proves our abstraction (`procio` + `lifecycle`) is correct, but requires explicit documentation that *context chaining* is the preferred pattern for fine-grained timeout control inside workers.
* **Consequences**:
  * No breaking changes. This serves as an architectural reinforcement.
  * Future enhancements to `lifecycle.Go` or `procio` process execution may introduce explicit nested timeout helpers similar to `rod`'s `Timeout()` wrappers.
