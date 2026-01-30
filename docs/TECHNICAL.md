# Technical Architecture

## Overview

`lifecycle` manages the interaction between OS signals, Context cancellation, and blocking I/O calls.

## Patterns

### 1. Robust Signal Context (`pkg/signal`)

Our `SignalContext` manages the transition from Graceful to Forced shutdown, essential for interactive CLIs.

```mermaid
stateDiagram-v2
    [*] --> Running
    
    Running --> Graceful: SIGINT/SIGTERM (1st)
    note right of Graceful
        Context cancelled.
        App starts cleanup.
    end note

    Graceful --> ForceExit: SIGINT/SIGTERM (2nd)
    note right of ForceExit
        os.Exit(1) called.
        Immediate termination.
    end note

    ForceExit --> [*]
    Graceful --> [*]: Natural Cleanup Complete
```

**Key Features:**

* **Functional Options**: Customize if `SIGINT` cancels or the number of signals for `ForceExit`.
* **Zero Leak**: monitoring goroutine is cleaned up via `.Stop()`.
* **Lifecycle Hooks**: Supports `OnShutdown` callbacks executed in LIFO order (defer-like) upon signal reception.

#### Execution Flow

```mermaid
sequenceDiagram
    participant OS
    participant SignalContext
    participant Hook_B
    participant Hook_A
    participant App

    OS->>SignalContext: SIGTERM
    SignalContext->>App: Cancel Context (ctx.Done closed)
    
    rect rgb(30, 30, 30)
        note right of SignalContext: Async Cleanup (LIFO)
        SignalContext->>Hook_B: Execute()
        Hook_B-->>SignalContext: Return
        SignalContext->>Hook_A: Execute()
        Hook_A-->>SignalContext: Return (or Panic recovered)
    end
```

### 2. Interruptible I/O (`pkg/termio`)

Go's `io.Reader` is blocking. We cannot easily kill a thread blocked on a syscall. `InterruptibleReader` solves this by using a "Peek & Abandon" strategy.

```mermaid
sequenceDiagram
    participant Main
    participant ReaderWrapper
    participant OS_Stdin
    
    Main->>ReaderWrapper: Read(p)
    ReaderWrapper->>ReaderWrapper: Select(ctx.Done?)
    
    alt Context Active
        ReaderWrapper->>OS_Stdin: syscall.Read() [BLOCKING]
        OS_Stdin-->>ReaderWrapper: return n bytes
        ReaderWrapper->>ReaderWrapper: Select(ctx.Done?)
        ReaderWrapper-->>Main: return n, nil
    else Context Cancelled (Before)
        ReaderWrapper-->>Main: return 0, ErrInterrupted
    else Context Cancelled (During)
        note over ReaderWrapper: OS Read eventually returns (or we abandon)
        OS_Stdin-->>ReaderWrapper: return n bytes
        ReaderWrapper->>ReaderWrapper: Select(ctx.Done?) -> YES
        ReaderWrapper-->>Main: return 0, ErrInterrupted
    end
```

#### Caveats & Constraints

* **Data Loss (Peek & Abandon)**: If a read returns exactly as the context is cancelled, the bytes consumed from the OS buffer are **discarded** to prioritize the error. This is acceptable for interactive CLIs but risky for binary streams.
* **Blocking Syscalls**: Go cannot interrupt a raw `read()` syscall. The `Reader` remains blocked in the OS until data arrives, even if the user receives `ErrInterrupted`.
* **Buffer Inconsistency**: Sharing the underlying `io.Reader` between multiple `InterruptibleReader` instances leads to non-deterministic data theft.

### 3. Process Hygiene (`pkg/proc`)

Ensures that child processes do not outlive the parent (preventing "Zombies"), essential for managing Language Servers or background tools. We follow a **Fail-Closed** principle: if a process cannot be safely managed by the OS hygiene (e.g., job assignment failure), it is immediately killed to prevent leaks.

* **Linux**: Uses `SysProcAttr.Pdeathsig` to signal the child when the parent thread dies.
* **Windows**: Uses **Job Objects** (`JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`) to ensure the OS terminates the child tree when the parent handle is closed.

### 4. Runtime Management (`pkg/runtime`)

Ensures process lifecycle is deterministic.

* **Timeouts**: `BlockWithTimeout` enforces deadlines on Shutdown phases. This prevents "Cleanup Zombie" states where a CLI hangs forever waiting for a database close or log flush.

### 5. Observability (`pkg/log` & `pkg/metrics`)

The library is instrumented for production visibility without external dependencies.

* **Structured Logging**: Uses Go 1.21 `slog`. Users can inject custom loggers via `lifecycle.SetLogger`.
* **Decoupled Metrics**: A `Provider` interface allows users to bridge lifecycle events to Prometheus or OTEL.
  * **Signals**:
    * `IncSignalReceived(signal string)`: Counter of OS signals received.
  * **Processes** (`pkg/proc`):
    * `IncProcessStarted()`: Counter of child processes managed.
    * `IncProcessFailed()`: Counter of process start failures.
  * **Terminal** (`pkg/termio`):
    * `IncTerminalUpgrade(success bool)`: Counter of stdin upgrades (e.g. to CONIN$).
  * **Hooks** (`pkg/signal` - *New*):
    * `IncHookExecuted()`: Counter of successfully completed hooks.
    * `IncHookPanicked()`: Counter of hooks that panic (and recovered).
    * `ObserveHookDuration(duration)`: Histogram/Timer of hook execution time.
* **LogProvider**: Special provider that redirects metrics to debug logs for local verification.

### 6. Safety Mechanisms

* **Stalled Hook Detection**: Each hook is monitored by a timer (configurable via `WithHookTimeout`). If it exceeds the threshold (default 5s), a warning is logged.
* **Panic Recovery**: Hooks are wrapped in `recover()` to ensure a single faulty hook does not prevent others from running.

### 7. Introspection & Visualization (`pkg/signal`)

To support tooling and better stewardship, we adopt an **Introspection Pattern**: components expose their configuration state as immutable DTOs (Data Transfer Objects). This decouples the internal logic from external representation.

* **Context State**: `SignalContext` exposes a `State()` method returning its current configuration (timeouts, thresholds, etc.).
* **Mermaid Renderer**: A pure function `Mermaid(State)` in `pkg/signal` consumes this state to generate a Mermaid State Diagram, allowing runtime visualization of the application's lifecycle policy.

## Windows `CONIN$`

On **Windows**, `os.Stdin` is a wrapper around a Handle. If that Handle is in a blocking Read, standard Windows signals might not propagate correctly to the Go runtime in console applications.
Using `CONIN$` ensures we are talking directly to the Console Input buffer, which allows `Ctrl+C` events to bypass the blocking Read and trigger the `pkg/signal` handler.

### 8. Worker Protocol (`pkg/worker`)

To support the Supervisor pattern (v1.3), we define a uniform `Worker` interface for managed units of work (Processes, Goroutines, Containers).

```mermaid
sequenceDiagram
    participant Manager
    participant Worker
    
    Manager->>Worker: Start(ctx)
    activate Worker
    
    rect rgb(30, 30, 30)
        note right of Worker: Work happens...
    end

    alt Graceful Stop
        Manager->>Worker: Stop(ctx)
        Worker-->>Manager: Returns nil
        Worker->>Worker: Closes Wait() channel
    else Crash
        Worker->>Worker: Closes Wait() channel (w/ error)
    end
    deactivate Worker
```

* **Process Worker**: The `Process` implementation wraps `exec.Cmd` and enforces **Fail-Closed** hygiene using `pkg/proc` (JobObjects/PDeathSig).
