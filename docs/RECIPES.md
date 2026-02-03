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

    // 3. Define Worker (See 'Quiescent Worker' pattern below)
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
