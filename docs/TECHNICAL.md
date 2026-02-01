# Technical Architecture (v1.x)

> **Note**: This document describes the architecture of the **v1.x series** (Death Management).

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
* **Introspection**: `Reason()` allows checking *why* a context was cancelled (`Interrupt`, `Terminate`, `Timeout` or `Manual:Stop`).

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

### 2. Context-Aware I/O & Safety (`pkg/termio`)

Traditional I/O is binary: it reads or it blocks. `lifecycle` introduces **Context-Aware I/O**, allowing the application to decide how to balance **Data Preservation** vs **Responsiveness** based on the use case.

#### Strategy A: Shielded Return (Pipeline Safe)

**Use Case**: CI/CD Pipelines, Log Aggregation, Automation.
**Goal**: Never lose data. If `SIGINT` arrives *while* data is being read, we must return the data first.

* **Method**: `InterruptibleReader.Read(p)` takes a "Data First" approach.
* **Behavior**:
    1. Blocks on OS Read.
    2. If Data arrives AND Context is cancelled: Returns `n > 0, nil`.
    3. Only returns `ErrInterrupted` if *no data* was read.

#### Strategy B: Strict Discard (Interactive Safe)

**Use Case**: Interactive Prompts (Y/N), Confirmation Dialogs, Menu Selection.
**Goal**: Safety First. If the user hits `Ctrl+C` while typing, they likely *changed their mind*, and we should not act on partial input (like `y` followed by `Ctrl+C`).

* **Method**: `InterruptibleReader.ReadInteractive(p)` takes an "Error First" approach.
* **Behavior**:
    1. Blocks on OS Read.
    2. If Data arrives AND Context is cancelled: Returns `n=0, ErrInterrupted`.
    3. **Discards** the buffer to prevent accidental execution.

#### Strategy C: Regret Window (Execution Safe)

**Use Case**: Critical Operations (Deleting Resources, Deployment).
**Goal**: Give the user a "grace period" to abort *after* submitting a command.

* **Method**: `lifecycle.Sleep(ctx, duration)`.
* **Behavior**:
    1. User submits command (Input Phase complete).
    2. App pauses for N seconds.
    3. If `Ctrl+C` happens during sleep, `Sleep` returns `ctx.Err()` immediately, allowing clean abort.

```mermaid
sequenceDiagram
    participant App
    participant Reader
    participant OS_Stdin
    participant Context

    note over App: Strategy Selection

    alt Strategy A (Data First)
        App->>Reader: Read()
        OS_Stdin-->>Reader: Returns "Data"
        Context-->>Reader: Returns "Cancelled"
        Reader-->>App: Return "Data", nil
        note right of App: Process Data
    else Strategy B (Error First)
        App->>Reader: ReadInteractive()
        OS_Stdin-->>Reader: Returns "Data"
        Context-->>Reader: Returns "Cancelled"
        Reader-->>App: Return 0, ErrInterrupted
        note right of App: Abort Operation (Strict)
    else Strategy C (Regret Window)
        App->>App: Input Accepted
        App->>lifecycle: Sleep(ctx, 3s)
        Context-->>lifecycle: Cancelled (User Regret)
        lifecycle-->>App: Return ctx.Err()
        note right of App: Abort Execution
    end
```

#### Caveats & Constraints

* **Blocking Syscalls**: Go cannot interrupt a raw `read()` syscall. The `Reader` remains blocked in the OS until data arrives, even if the user receives `ErrInterrupted`.
* **Buffer Inconsistency**: Sharing the underlying `io.Reader` between multiple `InterruptibleReader` instances leads to non-deterministic data theft.

#### Windows `CONIN$`

On **Windows**, `os.Stdin` is a wrapper around a Handle. If that Handle is in a blocking Read, standard Windows signals might not propagate correctly to the Go runtime in console applications.
Using `CONIN$` ensures we are talking directly to the Console Input buffer, which allows `Ctrl+C` events to bypass the blocking Read and trigger the `pkg/signal` handler.

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
  * **Hooks** (`pkg/signal`):
    * `IncHookExecuted()`: Counter of successfully completed hooks.
    * `IncHookPanicked()`: Counter of hooks that panic (and recovered).
    * `ObserveHookDuration(duration)`: Histogram/Timer of hook execution time.
  * **Workers** (`pkg/worker`):
    * `IncWorkerStarted(type)`: Counter of workers started.
    * `IncWorkerStopped(type)`: Counter of workers stopped gracefully.
    * `IncWorkerFailed(type)`: Counter of worker failures.
    * `ObserveWorkerDuration(type, duration)`: Histogram of worker lifespan.
* **LogProvider**: Special provider that redirects metrics to debug logs for local verification.

### 6. Safety Mechanisms

* **Stalled Hook Detection**: Each hook is monitored by a timer (configurable via `WithHookTimeout`). If it exceeds the threshold (default 5s), a warning is logged.
* **Panic Recovery**: Hooks are wrapped in `recover()` to ensure a single faulty hook does not prevent others from running.

### 7. Introspection & Visualization (`pkg/signal` & `pkg/worker`)

To support tooling and better stewardship, we adopt an **Introspection Pattern**: components expose their configuration state as immutable DTOs (Data Transfer Objects). This decouples the internal logic from external representation.

* **Signal Context State**: `SignalContext` exposes a `State()` method returning its current configuration.
  * **MermaidRenderer**: `MermaidState(State)` generates a **State Diagram** (FSM) of the lifecycle policy.
* **Worker State**: All workers implement `State()` returning a recursive snapshot.
  * **MermaidTree(State)**: Generates a **Toplogical/Tree** diagram (`graph TD`) of the supervision tree.
  * **MermaidState(State)**: Generates a **Lifecycle/FSM** diagram (`stateDiagram-v2`) for individual workers.

These are exposed at the top-level via `lifecycle.SignalStateDiagram`, `lifecycle.WorkerTreeDiagram`, `lifecycle.WorkerStateDiagram`, and the unified `lifecycle.SystemDiagram`.

* **System Diagram (Synthesis)**: A specialized visualizer that joins the `SignalContext` (Control Plane) and the `Worker` Tree (Data Plane). It highlights the logical relationship where the signal handler cancels the root of the supervision tree.

#### Visualization Standards

We maintain a consistent visual language across all generated diagrams to reduce cognitive load:

| Element | Diagram Type | Intent |
| :--- | :--- | :--- |
| **Logic/FSM** | `stateDiagram-v2` | Show "How it behaves" (Transitions, Hooks, Timeouts). |
| **Structure** | `graph TD` | Show "How it's built" (Parents, Children, PIDs). |

**Status Color Palette:**

We use a standard bootstrap-like palette for status coloring:

* 🟡 **Pending**: `#fff3cd` (Yellow) - Defined, but not yet active.
* 🔵 **Running**: `#d1ecf1` (Blue) - Active, healthy, or in-progress.
* 🟢 **Stopped**: `#d4edda` (Green) - Successfully terminated (Exit 0).
* 🔴 **Failed**: `#f8d7da` (Red) - Terminated with error or crashed.

**Diagnostic Symbolism (v1.3.1):**

To provide rapid infrastructure awareness, tree diagrams use type-specific shapes and icons based on `Metadata` detection:

| Shape | Icon | Identity | Rule |
| :--- | :--- | :--- | :--- |
| `[[Node]]` | 📦 | **Container** | Metadata includes `image` key. |
| `[Node]` | ⚙️ | **OS Process** | Metadata includes `path` key. |
| `([Node])` | 🧬 | **Goroutine** | Default/Ephemeral unit of work. |

Labels automatically enrich with **Diagnostic Snapshots** (e.g., 🌐 IP addresses, Ports) when detected in metadata.

> [!TIP]
> When implementing `State()` for new components, ensure the fields captured are sufficient to reconstruct these diagrams without needing a reference to the live object.

#### Windows & UTF-8 Encoding

If you see mangled characters in Mermaid diagrams (e.g., `ÔÜÖ´©Å` instead of `⚙️`), it is likely due to the terminal encoding. Windows terminals (CMD, PowerShell) often default to CP850.

To fix this, run:

```bash
chcp 65001
```

This sets the code page to UTF-8. Using the **Windows Terminal** is also recommended as it supports UTF-8 by default.

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

* **Process Worker**: The `ProcessWorker` implementation wraps `exec.Cmd` and enforces **Fail-Closed** hygiene using `pkg/proc` (JobObjects/PDeathSig).

### 9. Supervisor Pattern (`pkg/supervisor`)

The Supervisor manages a set of child Workers (Processes or other Supervisors), forming a **Supervision Tree**. It is responsible for starting, monitoring, and restarting children based on failures.

#### Restart Strategies

* **OneForOne**: If a child dies, only that child is restarted.
* **OneForAll**: If a child dies, all other children are stopped, and then all are restarted. (Useful for tightly coupled dependencies).

#### Handover Protocol

The Supervisor maintains a persistent **Resume ID** for each worker spec. This ID remains constant across restarts, allowing the worker to identify its session in durable storage or logs.

When a worker is restarted due to failure, the Supervisor injects the following via the `EnvInjector` (typically environment variables):

* **`LIFECYCLE_RESUME_ID`**: The stable UUID for the worker.
* **`LIFECYCLE_PREV_EXIT`**: The exit code (or `-1` on crash) of the previous execution.

```mermaid
sequenceDiagram
    participant Sup as Supervisor
    participant W as Worker (Instance 1)
    participant W2 as Worker (Instance 2)
    
    Sup->>W: Start (Injected: RESUME_ID=ABC, PREV_EXIT=0)
    W-->>Sup: Crash!
    
    note over Sup: Strategy OneForOne
    
    Sup->>W2: Start (Injected: RESUME_ID=ABC, PREV_EXIT=-1)
    note right of W2: Worker resumes work for session 'ABC'
```

#### Backoff Strategy (v1.3.1)

To prevent "Tight Loops" where a broken child restarts immediately and repeatedly (burning CPU), the Supervisor implements an **Exponential Backoff** strategy.

* **Configuration**: Each `Spec` can define `Backoff` parameters (InitialInterval, MaxInterval, Multiplier, ResetDuration).
* **Reset Logic**: If a child runs successfully for `ResetDuration`, the backoff interval resets to the initial value.
* **Jitter**: A 10% randomization is added to intervals to prevent thundering herd restarts.

#### Dynamic Topology (v1.3.1)

The Supervisor topology is not static. It supports runtime modifications:

* **Add(Spec)**: Dynamically starts a new worker under supervision.
* **Remove(Name)**: Gracefully stops a worker and removes it from the supervision tree.

This allows the Supervisor to act as a "Connection Manager" or "Session Host" where workers come and go based on external events.

### 10. Ecosystem Interfaces & Containers (`pkg/container`)

To support broader infrastructure management, `lifecycle` provides a generic `Container` interface. This allows the Supervisor to manage containerized workloads (Docker, Podman, Mock) without a direct compile-time dependency on third-party SDKs.

* **Decoupling**: Applications depend on `container.Container`, and the concrete implementation is injected at runtime.
* **ContainerWorker**: A bridge worker that adapts the `Container` interface to the standard `Worker` contract.

```mermaid
graph TD
    Sup[Supervisor] --> CW[ContainerWorker]
    CW --> C["Container (Interface)"]
    C --> D[Docker Impl]
    C --> M[Mock/Shell Impl]
    
    style CW fill:#ccf,stroke:#333
    style C fill:#f9f,stroke:#333
```

### 11. Reliability Primitives (`internal/reliability`) (v1.4)

To support **Durable Execution** engines (like Trellis), we provide primitives that shield critical operations from interruptions.

#### Critical Sections (`lifecycle.Do`)

`lifecycle.Do(ctx, fn)` allows executing a function that *cannot be cancelled* by the parent context until it completes.

* **Shielding**: The provided `fn` receives a new context that is **detached** from the parent's cancellation.
* **Deferral**: If the parent context is cancelled during execution, `Do` waits for `fn` to return before returning the parent's error.

```mermaid
sequenceDiagram
    participant P as Parent Context
    participant D as lifecycle.Do
    participant F as Function
    
    P->>D: Call Do(ctx, fn)
    D->>F: Run fn(shieldedCtx)
    
    note right of P: User hits Ctrl+C
    P--xP: Cancelled!
    
    note over D: Do detects cancellation<br/>but WAITS for fn
    
    F->>F: Complete Critical Work
    F-->>D: Return
    
    D-->>P: Return ctx.Err() (Canceled)
```

> [!WARNING]
> **Blocking Shutdown**: Since `Do` waits for the function to complete, a long-running or hung function *inside* a `Do` block will prevent the application from shutting down, effectively overriding `SIGINT`. Use strict internal timeouts within the shielded function.
