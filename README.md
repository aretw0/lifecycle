# lifecycle

[![Go Report Card](https://goreportcard.com/badge/github.com/aretw0/lifecycle)](https://goreportcard.com/report/github.com/aretw0/lifecycle)
[![Go Reference](https://pkg.go.dev/badge/github.com/aretw0/lifecycle.svg)](https://pkg.go.dev/github.com/aretw0/lifecycle)
[![License](https://img.shields.io/github/license/aretw0/lifecycle.svg?color=red)](LICENSE.txt)
[![Release](https://img.shields.io/github/release/aretw0/lifecycle.svg?branch=main)](https://github.com/aretw0/lifecycle/releases)

**lifecycle** is a Go library for managing application shutdown signals and interactive terminal I/O robustly. It centralizes the "Dual Signal" logic and "Interruptible I/O" patterns originally extracted from [Trellis](https://github.com/aretw0/trellis) and designed for any tool needing robust signal handling.

## Vision

To be the **standard Control Plane** for Infrastructure-Aware Applications (Services, Agents, CLIs).

* **v1 (Foundation)**: Solves "Death Management" (Signals, Blocking I/O, Zombies).
* **v2 (Evolution)**: Solves "Life Management" (Events, Reactions, Hot Reloading).

## Project Status & Versioning

> [!IMPORTANT]
> **v1.x (Current - LTS)**: Focuses strictly on **Death Management** (Graceful Shutdown, Signals, Leak Prevention).
> This branch is in **Maintenance Mode**. New features target v2.0.
>
> **v2.x (Upcoming)**: Will introduce the **Application Control Plane**, generalizing "Signals" into "Events" (Hot Reload, Health Checks, etc).

## Installation

```bash
go get github.com/aretw0/lifecycle
```

## Features (Foundation v1)

* **SignalContext**: Differentiates between `SIGINT` (User Interrupt) and `SIGTERM` (System Shutdown).
  * **SIGINT**: Captured but doesn't cancel context immediately (allows "Wait, are you sure?" logic).
  * **SIGTERM**: Cancels context immediately (standard graceful shutdown).
* **TermIO**:
  * **InterruptibleReader**: Wraps `io.Reader` to allow `Read()` calls to be abandoned when a context is cancelled (avoids goroutine leaks).
  * **Platform Aware**: Automatically uses `CONIN$` on Windows.
    * **Why?** On Windows, standard `os.Stdin` closes immediately upon receiving a signal (like Ctrl+C), causing a fatal `EOF` before the application can gracefully handle the signal. `lifecycle` switches to `CONIN$`, which keeps the handle open, allowing the `SignalContext` to process the event.
  * **UpgradeTerminal**: Helper to upgrade an arbitrary existing `io.Reader` (if it identifies as a terminal) to the safe platform-specific reader.
* **Observability & Introspection**:
  * **Unified Dashboard**: `SystemDiagram` synthesizes Signal and Worker states into a single Mermaid visualization.
  * **Rich Metrics**: Built-in providers for tracking shutdown health, data loss, and shutdown latency.
  * **Stall Detection**: Automatically detects and warns if a shutdown hook is stalled (runs > 5s).
* **Reliability Primitives** (v1.4):
  * **Critical Sections**: `lifecycle.Do(ctx, fn)` shields atomic operations from cancellation and returns any error from the protected function.
  * **Introspection**: `SignalContext.Reason()` to differentiate between "Manual Stop", "Interrupt", or "Timeout".
* **Worker & Supervisor** (v1.3):
  * **Unified Interface**: Standard `Start`, `Stop`, `Wait` contract for Processes, Goroutines, and Containers.
  * **Supervision Tree**: `Supervisor` manages hierarchical worker clusters with restart policies (`OneForOne`, `OneForAll`).
  * **Dynamic Topology**: Add or remove workers at runtime.
  * **Functional Workers**: Turn any Go function into a managed Worker.
  * **Process Hygiene**: Automatic cleanup of child processes if the parent dies (Job Objects/PDeathSig).
  * **Handover Protocol**: Standardized environment variables (`LIFECYCLE_RESUME_ID`, `LIFECYCLE_PREV_EXIT`) to pass context across restarts.
  * **Container Abstraction**: Generic interface to manage containerized workloads without direct SDK dependencies.
* **DX Helpers** (v1.4):
  * **`Run`**: One-line `main` entry point (Context + Signal Handling + Cleanup).
  * **`Sleep`**: Context-aware sleep (returns immediately on cancel).
  * **`OnShutdown`**: Type-safe hook registration without casting.

## Roadmap (Control Plane v2)

* **Event Router**: Generalize `Signals` into `Events` (Webhook, FileWatch, HealthCheck).
* **Managed Concurrency**: `lifecycle.Go(ctx, fn)` for non-leaking goroutines.
* **Reactions**: `Reload`, `Suspend`, `Scale` alongside `Shutdown`.

### Managed Concurrency (v2.0 Preview)

`lifecycle` now provides primitives to manage goroutines safely, ensuring they respect shutdown signals and provide visibility.

```go
lifecycle.Run(func(ctx context.Context) error {
    // Fire-and-forget but tracked and panic-safe
    lifecycle.Go(ctx, func(ctx context.Context) error {
        // ...
        return nil
    })
    return nil
})
```

## Usage

### Signal Context

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/aretw0/lifecycle"
)

func main() {
    // lifecycle.Run handles context creation, signal listening, and cleanup.
    // It automatically waits for hooks if a signal is received.
    lifecycle.Run(lifecycle.Job(runApp))
}

func runApp(ctx context.Context) error {
    // 1. Frictionless Hook Registration
    lifecycle.OnShutdown(ctx, func() {
        fmt.Println("Cleanup: Database closed")
    })

    // 2. Safe Sleep (Regret Window)
    // Returns immediately if Ctrl+C is pressed.
    if err := lifecycle.Sleep(ctx, 10*time.Second); err != nil {
        return err
    }
    
    return nil
}
```

### Interruptible I/O

```go
package main

import (
    "context"
    "fmt"
    "github.com/aretw0/lifecycle"
)

func main() {
    ctx := context.Background() // or SignalContext
    
    // Smart Open (handles Windows CONIN$)
    reader, _ := lifecycle.OpenTerminal()
    
    // Wrap to respect context cancellation
    r := lifecycle.NewInterruptibleReader(reader, ctx.Done())

    buf := make([]byte, 1024)
    n, err := r.Read(buf)
    if lifecycle.IsInterrupted(err) {
        fmt.Println("Read cancelled!")
        return
    }
    fmt.Printf("Read: %s\n", buf[:n])
}
```

### Worker Protocol (v1.3)

Manage long-running processes, containers, or goroutines with a uniform interface, hygiene, and handover support.

```go
package main

import (
    "context"
    "fmt"
    "github.com/aretw0/lifecycle"
)

func main() {
    ctx := lifecycle.NewSignalContext(context.Background())
    defer ctx.Stop()

    // 1. Process Worker (Fail-Closed hygiene automatically applied)
    worker := lifecycle.NewProcessWorker("pinger", "ping", "127.0.0.1")

    // 2. Handover Protocol (Access resume info in child process via env)
    // resumeID := os.Getenv(lifecycle.EnvResumeID)

    // Async Start
    worker.Start(ctx)

    // Wait for shutdown or worker exit
    select {
    case <-ctx.Done():
        worker.Stop(context.Background()) // Graceful stop
    case <-worker.Wait():
        fmt.Println("Worker finished!")
    }
}
```

### System Introspection (v1.3)

Generate live architecture diagrams of your running application.

```go
// Get current snapshots
sigState := ctx.State()
workState := supervisor.State()

// Generate Mermaid "Unified Dashboard"
diagram := lifecycle.SystemDiagram(sigState, workState)
fmt.Println(diagram)
```

> [!NOTE]
> We use **state diagrams** (`stateDiagram-v2`) for behavior/FSM and **flowcharts** (`graph TD`) for topology/trees.

## Metrics Palette

The library uses a consistent color palette for all generated diagrams:

* 🟡 **Pending**: Defined but not yet active.
* 🔵 **Running**: Active and healthy.
* 🟢 **Stopped**: Successfully terminated.
* 🔴 **Failed**: Crashed or terminated with error.

## I/O Safety

The library implements **Context-Aware I/O** to balance data preservation and responsiveness:

* **`Read()` (Pipeline Safe)**: Uses a **Shielded Return** strategy. If data arrives simultaneously with a cancellation signal, it returns the *data* (nil error). This guarantees no data loss in pipelines or logs.
* **`ReadInteractive()` (Interactive Safe)**: Uses a **Strict Discard** strategy. If the user hits Ctrl+C while typing, any partial input is discarded to prevent accidental execution of commands.

## Documentation

* [**PRODUCT**](docs/PRODUCT.md): Vision & Mission.
* [**TECHNICAL**](docs/TECHNICAL.md): Architecture & Design (with Diagrams).
* [**PLANNING**](docs/PLANNING.md): Roadmap & Backlog.
