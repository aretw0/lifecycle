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

## 💡 1.2 Concurrency Safety Recipe: Channels vs Mutexes

When writing concurrent code with `lifecycle`, you must ensure safe access to shared state and deterministic coordination between goroutines. The Go runtime does not protect you from data races or non-deterministic test failures—these are the developer’s responsibility.

This recipe consolidates best practices for using channels and mutexes in Go, with practical examples for robust worker and test design.

### Recommended Patterns

- **Mutexes**: Use `sync.Mutex` (or helpers like `withLock`, `withLockResult`) to protect mutable state inside workers or components. This ensures only one goroutine can access or modify the state at a time.
- **Channels**: Use channels to signal events, synchronize test assertions, or coordinate between goroutines. Prefer channels over `time.Sleep` for waiting on asynchronous actions in tests or production code.

#### Example: Synchronizing Test Hooks

Bad (race-prone, flaky):

```go
var ran bool
go func() {
    ran = true // data race!
}()
time.Sleep(50 * time.Millisecond)
if !ran { t.Error("Did not run") }
```

Good (deterministic, race-free):

```go
ran := make(chan struct{})
go func() {
    close(ran)
}()
<-ran // blocks until goroutine runs
```

#### Example: Protecting State with Mutex

```go
type Worker struct {
    mu sync.Mutex
    value int
}

func (w *Worker) Set(v int) {
    w.mu.Lock()
    defer w.mu.Unlock()
    w.value = v
}

func (w *Worker) Get() int {
    w.mu.Lock()
    defer w.mu.Unlock()
    return w.value
}
```

### When to Use Each

- Use **channels** for signaling, coordination, and test synchronization.
- Use **mutexes** for protecting shared, mutable state.

### Project Philosophy

`lifecycle` provides helpers, patterns, and recipes to encourage safe concurrency, but cannot abstract away all Go-level pitfalls. Users are expected to understand Go’s concurrency model, but can rely on the project’s examples and documentation to avoid common mistakes.

For more, see:  

- [RECIPES.md](RECIPES.md) (Quiescent Worker, channel patterns)  
- [TECHNICAL.md](TECHNICAL.md) (withLock, shutdown protocol)  
- [TESTING.md](TESTING.md) (Avoid Sleep, prefer channels)

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

- **1st Ctrl+C**: Custom Logic (e.g., Suspend).
- **2nd Ctrl+C**: Custom Logic (or Ignored).
- **3rd Ctrl+C**: **FORCE EXIT** (Runtime Kill Switch).

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

---

## 🚫 11. The Inhibition Pattern (Debounce + Filter)

**Problem**: You are watching a directory for changes (like a Git repository) and you want to trigger a build or sync. However, you don't want to react to noisy internal files (like `.git/index.lock` or temporary editor files), and you don't want to trigger 50 builds when a user saves 50 files at once.

> [!WARNING]
> OS Watcher Limits (Inotify)
> Operating systems limit how many file watchers you can have open simultaneously (e.g., `fs.inotify.max_user_watches` on Linux). Watching massive nested directories like `node_modules` or `vendor` can quickly exhaust this limit and crash your supervisor. **You must aggressively filter out huge dependencies.**

**Solution**: Combine `events.WithFilter` on your Source with an `events.DebounceHandler` on your Router. This pattern offloads "project awareness" and "throttling" from your business logic directly into the Control Plane.

```go
// 1. Create a recursive FileWatchSource that inhibits (filters) noisy files
filter := func(path string) bool {
    // Inhibit git metadata and lock files
    if strings.Contains(path, ".git") || strings.HasSuffix(path, ".lock") {
        return false 
    }
    return true
}
source := events.NewFileWatchSource("./my-project", 
    events.WithRecursive(true), 
    events.WithFilter(filter),
)

// 2. Define your business logic handler (e.g., triggering a build)
buildHandler := events.HandlerFunc(func(ctx context.Context, e events.Event) error {
    slog.Info("Changes settled, triggering build...")
    return runBuild()
})

// 3. Wrap it in a DebounceHandler
// We use a 500ms window. Nil mergeFunc means we only care that *something* 
// changed, dropping the intermediate spam.
debouncedBuild := events.DebounceHandler(buildHandler, 500*time.Millisecond, nil)

// 4. Register and Run
router.AddSource(source)
router.Handle("file/*", debouncedBuild)
```

---

## ⛓️ 12. The Chained Cancels Pattern (Orphan Prevention)

**Problem**: You are spawning child goroutines or external processes from a worker, but you are using `context.Background()` or `context.TODO()` because you don't want them to be cancelled immediately when the *main* context is cancelled (e.g., they need their own timeout). This creates **Pathological Detachments** where the children become orphans if the application crashes or is force-killed.

**Solution**: Always derive your child contexts from the provided worker `ctx`. If you need a specific timeout, use `context.WithTimeout(ctx, ...)`. If you need it to outlive a specific phase but still be tied to the app, use the Supervisor's base context or `signal.Context`.

```go
func (w *MyWorker) Start(ctx context.Context) error {
    lifecycle.Go(ctx, func(taskCtx context.Context) error {
        // ✅ GOOD: Derived from taskCtx (which is derived from ctx)
        // If the main application cancels, this timeout is also cancelled.
        childCtx, cancel := context.WithTimeout(taskCtx, 5*time.Second)
        defer cancel()

        cmd := exec.CommandContext(childCtx, "long-running-task")
        return proc.Start(cmd)
    })
    
    // ❌ BAD: Detached from lifecycle
    // if the app shuts down, this process might become a zombie or hang the server.
    // go func() {
    //    cmd := exec.CommandContext(context.Background(), "task")
    //    cmd.Run()
    // }()

    return nil
}
```

> [!IMPORTANT]
> **Zombie Prevention**: Combining **Chained Contexts** with `procio`'s hygiene guarantees (Job Objects/PDeathSig) ensures that orphans are reaped at both the software level (Go Context) and the OS level (Process Group/Job).
