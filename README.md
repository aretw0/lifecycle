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
> **v2.x (Current)**: Introduces the **Application Control Plane**, generalizing "Signals" into "Events" (Hot Reload, Health Checks, Input Commands).
>
> **v1.x (Stable - LTS)**: Focuses strictly on **Death Management** (Graceful Shutdown, Signals, Leak Prevention).

## Installation

```bash
go get github.com/aretw0/lifecycle
```

## Features (Foundation v1)

* **SignalContext**: Differentiates between `SIGINT` (User Interrupt) and `SIGTERM` (System Shutdown).
  * **Configurable Thresholds**: Support for "Industry Standard" (Cancel on 1st), "Escalation" (Cancel on Nth), and "Unsafe" (No auto-kill) modes.
  * **SIGTERM**: Always triggers immediate graceful shutdown (context cancellation).
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
* **DX Helpers** (v1.4 & v2.0):
  * **`Run`**: One-line `main` entry point with options (`WithLogger`, `WithMetrics`).
  * **`NewInteractiveRouter`**: Pre-configured router for CLIs (Signals + Input + Commands).
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

## Quick Start (The Golden Path)

For 99% of CLI applications, you just need `lifecycle.Run`. It handles signals, context cancellation, and cleanup automatically.

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/aretw0/lifecycle"
)

func main() {
    // 1. Wrap your logic in a Job
    // 2. lifecycle.Run manages the boring "Death Management" stuff
    lifecycle.Run(lifecycle.Job(func(ctx context.Context) error {
        fmt.Println("App started. Press Ctrl+C to exit.")
        
        // 3. Use lifecycle.Go to spawn safe, tracked background tasks
        lifecycle.Go(ctx, func(ctx context.Context) error {
            // This goroutine is automatically waited for on shutdown
            // Panics are caught and logged, preventing crashes
            select {
            case <-ctx.Done():
                return nil
            case <-time.After(5 * time.Second):
                fmt.Println("Task complete")
                return nil
            }
        })

        // 4. Wait for interrupt
        <-ctx.Done()
        fmt.Println("Shutting down...")
        return nil
    }))
}
```

## Basic Patterns (CLI & Automation)

### 1. Robust Input (Windows Safe)

Reading from Stdin on Windows is tricky. `lifecycle` solves the "Ctrl+C kills the prompt" problem by automatically using `CONIN$`.

```go
// Smart Open (handles Windows CONIN$)
reader, _ := lifecycle.OpenTerminal()
defer reader.Close()

// Wrap to respect context cancellation (prevents blocked Read calls)
r := lifecycle.NewInterruptibleReader(reader, ctx.Done())

buf := make([]byte, 1024)
n, err := r.Read(buf)
if lifecycle.IsInterrupted(err) {
    return // Clean exit
}
```

### 2. Graceful Hooks

Register cleanup functions that run *after* the context is cancelled but *before* the process exits.

```go
lifecycle.OnShutdown(ctx, func() {
    db.Close()
    fmt.Println("Cleanup done locally")
})
```

## Advanced Patterns (Control Plane v2)

For complex long-running services/agents that need dynamic behavior (Hot Reload, Supervisors).

### 1. Managed Concurrency & Global Fallback

`lifecycle.Go(ctx, fn)` is designed to work within `lifecycle.Run`. However, if you use it in a standalone script, it safely falls back to a **Global Task Tracker**.

```go
func main() {
    ctx := context.Background()
    
    // Works safely even without lifecycle.Run
    lifecycle.Go(ctx, func(ctx context.Context) error {
        // ...
        return nil
    })
    
    // Explicitly wait for global tasks (required without Run)
    lifecycle.WaitForGlobal()
}
```

### 2. Supervisor & Workers

Manage long-running processes, containers, or goroutines with restarts and hygiene.

```go
// Create a Supervisor with a "OneForOne" restart strategy
sup := lifecycle.NewSupervisor("agent", lifecycle.SupervisorStrategyOneForOne,
    lifecycle.NewProcessWorker("pinger", "ping", "1.1.1.1"),
    lifecycle.NewWorkerFromFunc("metrics", metricsLoop),
)

sup.Start(ctx)
<-ctx.Done()
sup.Stop(context.Background())
```

### 3. System Introspection

Generate live architecture diagrams of your running application.

```go
// Generate Mermaid Dashboard
diagram := lifecycle.SystemDiagram(ctx.State(), supervisor.State())
fmt.Println(diagram)
```

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
