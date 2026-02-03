# Lifecycle Recipes 📖

This document contains common architectural patterns and "recipes" for building robust Go applications with `lifecycle`.

---

## 🏗️ 1. The Interactive Service (CLI + Worker)

**Problem**: You want a long-running worker (e.g., consumer, processor) that can be controlled interactively via a CLI (Ctrl+C, Commands) without losing data.

**Solution**: Combine `Supervisor` for reliability, `InputSource` for control, and `Quiescence` for safe pausing.

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "os"
    "github.com/aretw0/lifecycle"
    "github.com/aretw0/lifecycle/pkg/sources"
    "github.com/aretw0/lifecycle/pkg/control"
    // "github.com/aretw0/lifecycle/pkg/worker" (Theoretical helper)
)

func main() {
    // 1. Setup Router & Sources
    router := lifecycle.NewRouter()
    
    // Listen for OS Signals (Ctrl+C as fallback)
    router.AddSource(lifecycle.NewOSSignalSource(os.Interrupt))
    
    // Listen for Interactive Commands (s=Suspend, r=Resume, q=Quit)
    router.AddSource(sources.NewInputSource())

    // 2. Setup Handlers
    suspendHandler := lifecycle.NewSuspendHandler()
    router.Handle("lifecycle/suspend", suspendHandler)
    router.Handle("lifecycle/resume", suspendHandler)
    router.Handle("input/quit", control.HandlerFunc(func(ctx context.Context, _ control.Event) error {
        // Trigger Shutdown logic here
        return nil 
    }))

    // 3. Define Worker (See implementation in 'The Quiescent Worker Pattern' section below)
    // ...

    // 4. Run Loop
    err := lifecycle.Run(lifecycle.Job(func(ctx context.Context) error {
        lifecycle.Go(ctx, router.Start)
        
        // Block until done
        <-ctx.Done()
        return nil
    }))
}
```

### 💡 The "Quiescent Worker" Pattern

To safely suspend a worker without losing "in-flight" data, the worker must support **Quiescence** (Paused State).

1. **Check Pause BEFORE work**: Before taking an item from a queue/channel, check if a pause was requested.
2. **Wait**: If paused, block on a `sync.Cond` until resumed.
3. **Finish In-Flight**: If a pause request comes *during* work, finish the current item, *then* pause.

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

## 🛡️ 5. The Safety Net Pattern (Interactive Robustness)

**Problem**: You are disabling default signal handling (`WithInterrupt(false)`) to use `Ctrl+C` for custom logic (like Suspend), but you are afraid of creating an "unkillable" zombie process if your custom logic fails.

**Solution**: Use `WithForceExit(N)` as a "Deadman Switch".

* **1st Ctrl+C**: Custom Logic (e.g., Suspend).
* **2nd Ctrl+C**: Custom Logic (or Ignored).
* **3rd Ctrl+C**: **FORCE EXIT** (Runtime Kill Switch).

```go
func main() {
    // ... setup router and custom handlers ...
    
    // We DISABLE default context cancellation on Interrupt
    // because we want to handle it ourselves (e.g. to Suspend).
    lifecycle.Run(job, 
        lifecycle.WithInterrupt(false), 
        
        // BUT we keep the "Safety Net".
        // If the user mashes Ctrl+C 3 times, we assume our custom logic is broken
        // and we force-kill the process.
        lifecycle.WithForceExit(3),
    )
}
```

> [!TIP]
> This pattern breaks the "Zombie Process" fear. Even if your `SmartHandler` deadlocks or fails to valid state, the user always has a panic button (Mash Ctrl+C).
