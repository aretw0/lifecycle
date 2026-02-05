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
    router := lifecycle.NewInteractiveRouter(suspendHandler,
        lifecycle.WithShutdown(func() {
            fmt.Println("Cleaning up before exit...")
        }),
    )

    // 3. Run
    lifecycle.Run(router)
}
```

---

## 🏗️ 1.1 Manual Router Setup (Advanced)

If you need full control over every source and middleware, you can still wire everything manually.

```go
package main

import (
    "context"
    "os"
    "github.com/aretw0/lifecycle"
    "github.com/aretw0/lifecycle/pkg/sources"
    "github.com/aretw0/lifecycle/pkg/control"
)

func main() {
    // 1. Setup Router & Sources
    router := lifecycle.NewRouter()
    
    // Listen for OS Signals
    router.AddSource(lifecycle.NewOSSignalSource(os.Interrupt))
    
    // Listen for Interactive Commands (s=Suspend, r=Resume, q=Quit)
    router.AddSource(sources.NewInputSource())

    // 2. Setup Handlers
    suspendHandler := lifecycle.NewSuspendHandler()
    router.Handle("lifecycle/suspend", suspendHandler)
    router.Handle("lifecycle/resume", suspendHandler)
    router.Handle("command/quit", control.HandlerFunc(func(ctx context.Context, _ control.Event) error {
        // Shutdown logic
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
router.AddSource(sources.NewTickerSource(100 * time.Millisecond))
router.Handle("source/tick", control.HandlerFunc(func(_ context.Context, e control.Event) error {
    // Update UI
    return nil
}))
```

---

## 🧠 4. Smart Signal Handling (State-Aware Ctrl+C)

**Problem**: You want `Ctrl+C` to have context-aware behavior: Suspend on the first press, Quit on the second (or if already suspended).

**Solution**: Use a custom Handler that checks state before deciding the action.

```go
// In main():
smartHandler := lifecycle.NewSmartSignalHandler(
    suspendHandler, 
    lifecycle.HandlerFunc(func(ctx context.Context, _ control.Event) error {
        // Quit Logic
        close(quitCh)
        return nil
    }),
)


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
    lifecycle.Run(job, 
        // If the user mashes Ctrl+C 3 times, we assume our custom logic is broken
        // and we force-kill the process.
        lifecycle.WithForceExit(3),
    )
}
```

> [!TIP]
> This pattern breaks the "Zombie Process" fear. Even if your `SmartHandler` deadlocks or fails to valid state, the user always has a panic button (Mash Ctrl+C).
