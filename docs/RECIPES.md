# Lifecycle Recipes 📖

This document contains common architectural patterns and "recipes" for building robust Go applications with `lifecycle`.

---

## 🏗️ 1. The Interactive Service (CLI + Worker)

**Problem**: You want a long-running worker (e.g., consumer, processor) that can be controlled interactively via a CLI (Ctrl+C, Commands) without losing data.

**Solution**: Use the `NewInteractiveRouter` preset to handle standard boilerplate, and `SuspendHandler` to manage worker state.

```go
package main

import (
    "context"
    "fmt"
    "github.com/aretw0/lifecycle"
)

func main() {
    // 1. Setup Suspend Logic
    suspendHandler := lifecycle.NewSuspendHandler()
    suspendHandler.OnSuspend(func(ctx context.Context) error {
        fmt.Println("Washing in-flight data...")
        return nil
    })

    // 2. Setup Interactive Router (The Easy Way)
    router := lifecycle.NewInteractiveRouter(
        lifecycle.WithSuspendOnInterrupt(suspendHandler),
        lifecycle.WithShutdown(func() {
            fmt.Println("Cleaning up before exit...")
        }),
    )

    // 3. Run
    // Important: Disable default Ctrl+C cancellation so the Router can handle it (Suspend)
    lifecycle.Run(router, lifecycle.WithCancelOnInterrupt(false))
}
```

---

## 🏗️ 1.1 Manual Router Setup (Advanced)

If you need full control over every source and middleware, you can still wire everything manually.

```go
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/aretw0/lifecycle"
)

func main() {
    // 1. Setup Router & Sources
    router := lifecycle.NewRouter()
    
    // Listen for OS Signals
    router.AddSource(lifecycle.NewOSSignalSource(os.Interrupt))
    
    // Listen for Interactive Commands (s=Suspend, r=Resume, q=Quit)
    router.AddSource(lifecycle.NewInputSource())

    // 2. Setup Handlers
    suspendHandler := lifecycle.NewSuspendHandler()
    router.Handle("lifecycle/suspend", suspendHandler)
    router.Handle("lifecycle/resume", suspendHandler)
    router.Handle("command/quit", lifecycle.HandlerFunc(func(ctx context.Context, _ lifecycle.Event) error {
        fmt.Println("Shutting down...")
        lifecycle.Shutdown(ctx)
        return nil 
    }))

    // 3. Run
    lifecycle.Run(router)
}
```

### 💡 The "Quiescent Worker" Pattern

To safely suspend a worker without losing "in-flight" data, the worker must support **Quiescence** (Paused State).

1. **Check Pause BEFORE work**: Before taking an item from a queue/channel, check if a pause was requested.
2. **Wait**: If paused, block until resumed (use `SuspendGate` or `sync.Cond`).
3. **Finish In-Flight**: If a pause request comes *during* work, finish the current item, *then* pause.

#### 💡 Tip: Accurate Quiescence UI

When building UIs that react to suspension, always register your **UI hooks AFTER your functional components**.

Since `SuspendHandler` executes hooks in FIFO order and blocks until they finish, registering components first ensures the UI message accurately reflects a fully quiesced system.

```go
// 1. Manage workers/supervisors first (blocking calls)
suspendHandler.Manage(worker)

// 2. Register UI notifications last (reports after workers stop)
suspendHandler.OnSuspend(func(ctx context.Context) error {
    slog.Info("🛑 ALL WORKERS QUIESCED")
    return nil
})
```

> [!NOTE]
> **Concurrency and Safety:**
> When implementing custom workers (especially with mutable internal state), always use the locking pattern (`withLock`/`withLockResult` or equivalent helpers) to ensure concurrency safety and avoid race conditions. See [docs/LIMITATIONS.md](LIMITATIONS.md) for exceptions, limitations, and examples of safe usage.
**Note:** This pattern relies on locking (mutexes) to ensure the consistency of the worker's internal state. For details about exceptions, limitations, and safe concurrency usage recommendations, see [LIMITATIONS.md](LIMITATIONS.md).

---

## 🔄 2. Hot Reloading

**Problem**: You want to update configuration without restarting the process.

**Solution**: Use `FileWatchSource` or `Signal(SIGHUP)` mapped to a `ReloadHandler`.

```go
// ... setup router ...
router.AddSource(sources.NewOSSignalSource(syscall.SIGHUP))

router.Handle("Signal(hangup)", lifecycle.NewReloadHandler(func(ctx context.Context) error {
    slog.Info("Reloading configuration...")
    return loadConfig()
}))
```

---

## ⏱️ 3. Headless Progress

**Problem**: You want to update a UI/Progress Bar based on lifecycle events or periodic ticks.

**Solution**: Use `TickerSource` injecting events into the Router.

```go
router.AddSource(lifecycle.NewTickerSource(100 * time.Millisecond))
router.Handle("source/tick", lifecycle.HandlerFunc(func(_ context.Context, e lifecycle.Event) error {
    // Update UI
    return nil
}))
```

---

## 🧠 4. Smart Signal Handling (State-Aware Ctrl+C)

**Problem**: You want `Ctrl+C` to have context-aware behavior: Suspend on the first press, Quit on the second (or if already suspended).

**Solution**: Use `Events.Escalator` to compose a "Double-Tap" strategy.

```go
// 1. Define Primary Action (e.g. Suspend)
// We wrap it with StateCheck so if it's already suspended, it returns ErrNotHandled,
// causing the Escalator to trigger the fallback (Quit).
primary := events.WithStateCheck(suspendHandler, suspendHandler)

// 2. Define Fallback Action (Quit)
quit := events.HandlerFunc(func(ctx context.Context, _ events.Event) error {
    close(quitCh)
    return nil
})

// 3. Compose Escalator
// First Signal -> Try Primary (Suspend)
// Second Signal (or if Suspended) -> Fallback (Quit)
smartHandler := events.NewEscalator(primary, quit)

router.Handle("Signal(interrupt)", smartHandler)
```

---

## 🛡️ 5. The Safe Shutdown Pattern (Idempotency)

**Problem**: Your shutdown logic (e.g., closing a channel, stopping a global resource) is not idempotent and panics if called more than once.

**Solution**: Use `control.Once(handler)` to wrap your shutdown logic.

If you are using `NewInteractiveRouter`, this is handled **automatically** for the `WithShutdown` option. However, if you are building a custom router, you should use the wrapper explicitly.

```go
// 1. Defining a non-idempotent quit channel
quitCh := make(chan struct{})

// 2. Wrap the handler to be safe against double-invocations
// (e.g., SIGINT after typing 'q' in a custom terminal)
quitHandler := control.Once(control.HandlerFunc(func(ctx context.Context, _ control.Event) error {
    slog.Info("Shutting down...")
    close(quitCh) // PANIC if called twice! control.Once prevents this.
    return nil
}))

router.Handle("command/quit", quitHandler)
```

---

## 🛡️ 6. The Safety Net Pattern (Interactive Robustness)

**Problem**: You are disabling default signal handling (`WithInterrupt(false)`) to use `Ctrl+C` for custom logic (like Suspend), but you are afraid of creating an "unkillable" zombie process if your custom logic fails.

**Solution**: Use `WithForceExit(N)` as a "Deadman Switch".

* **1st Ctrl+C**: Custom Logic (e.g., Suspend).
* **2nd Ctrl+C**: Custom Logic (or Ignored).
* **3rd Ctrl+C**: **FORCE EXIT** (Runtime Kill Switch).

```go
func main() {
    // ... setup router and custom handlers ...
    
    // We use a "Deadman Switch" configuration.
    // Setting ForceExit > 1 has two effects:
    // 1. DISABLES default context cancellation on the 1st Ctrl+C (allowing manual handling).
    // 2. ENABLES runtime Force Exit on the Nth Ctrl+C (Safety Net).
    lifecycle.Run(router, 
        // If the user mashes Ctrl+C 3 times, we assume our custom logic is broken
        // and we force-kill the process.
        lifecycle.WithForceExit(3),
        lifecycle.WithCancelOnInterrupt(false),
    )
}
```

> [!TIP]
> This pattern breaks the "Zombie Process" fear. Even if your `SmartHandler` deadlocks or fails to valid state, the user always has a panic button (Mash Ctrl+C).

---

## 🔌 7. Raw Input Streams (JSON/binary)

**Problem**: You want to read structured input (like JSON lines from another process) instead of interactive commands, but still want to use the `lifecycle` event loop.

**Solution**: Use `sources.NewInputSource()` with the `WithRawInput` option. This bypasses the default command parser and gives you the raw strings.

```go
// 1. Create a Source that reads from Stdin
source := lifecycle.NewInputSource(
    lifecycle.WithInputReader(os.Stdin),
    // 2. Register a Raw Input Handler
    lifecycle.WithRawInput(func(line string) {
        var msg MyMessage
        if err := json.Unmarshal([]byte(line), &msg); err != nil {
            slog.Error("Invalid JSON", "error", err)
            return
        }
        // Process message or dispatch event
        router.Dispatch(ctx, "custom/event", msg)
    }),
)

router.AddSource(source)
```

> [!NOTE]
> When `WithRawInput` is used, the default "command/quit" and "command/suspend" parsing is **disabled**. You must handle all input logic yourself.

---

## 🔌 8. Observability Bridge (lifecycle + procio)

**Problem**: You use both `lifecycle` workers and `procio` processes and want unified telemetry without duplicating observer setup.

**Solution**: Create a single `ObserverBridge` adapter implementing both `lifecycle.Observer` (7 methods) and `procio.Observer` (5 methods). Since `lifecycle.Observer` is a superset of `procio.Observer` (adds `LogInfo` and `OnGoroutinePanicked`), a single struct satisfies both.

```go
import (
    "log/slog"
    
    "github.com/aretw0/lifecycle"
    "github.com/aretw0/procio"
)

bridge := &ObserverBridge{
    Logger:   slog.Default(),
    Provider: myMetricsProvider,
}

lifecycle.SetObserver(bridge)
procio.SetObserver(bridge)
```

> [!TIP]
> For the full `ObserverBridge` type definition with compile-time interface checks and metric calls, see **[Global Overrides — Observer Bridge](CONFIGURATION.md#observer-bridge-lifecycle--procio)**.

---

## 🧯 9. Panic Observability (Observer + Stack Capture)

**Problem**: You want to route goroutine panics to your telemetry backend with optional stack traces.

**Solution**: Install a custom `Observer` and use `WithStackCapture` to control stack collection.
See [docs/TECHNICAL.md](TECHNICAL.md#14-observability) for behavior details and
[docs/CONFIGURATION.md](CONFIGURATION.md#observer-bridge-lifecycle--procio) for adapter examples.

---

## 🧩 10. Hybrid Migration (Manual Context)

For a short, maintained summary of the migration entry points, see [docs/MIGRATION.md](MIGRATION.md).
