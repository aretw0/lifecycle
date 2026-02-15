# Migration Guide: v1.4.1 → v1.5.0

> [!IMPORTANT]
> **Breaking Changes Notice**: The v1.5.0 release introduces significant architectural changes in the **Control Plane** (Event-Driven Architecture). While we remain on the v1.x module path to avoid `go.mod` migration overhead, this release contains breaking API changes that may require code updates.

## Overview

Version 1.5.0 represents a major evolution of the `lifecycle` library, introducing the **Event-Driven Control Plane** while maintaining backward compatibility where possible. The previous "v2.0" nomenclature in documentation has been consolidated into the v1.5 release.

### What Changed?

* **Foundation (v1.0 - v1.4)**: Focused on "Death Management" (Signals, Graceful Shutdown, Leak Prevention).
* **Control Plane (v1.5+)**: Introduces "Life Management" (Event Router, Hot Reload, Suspend/Resume, Managed Concurrency).

## Breaking Changes

### 1. Event Router Architecture

**What Changed**: Signal handling is now generalized into an event-driven `Router` system.

**v1.4.x (Legacy)**:

```go
// Direct signal handling
ctx := lifecycle.SignalContext(context.Background())
// Manual goroutine management
```

**v1.5.0 (Current)**:

```go
// Event-driven router with multiple sources
router := lifecycle.NewRouter()
router.Bind("shutdown", lifecycle.ShutdownHandler())
router.Bind("reload", lifecycle.ReloadHandler(myReloadFn))

// Or use the simplified Run API
lifecycle.Run(func(ctx context.Context) error {
    // Your application logic
    return nil
})
```

**Migration Path**:
* The `lifecycle.Run` function provides a zero-config migration path for simple applications.
* For advanced use cases, migrate to `Router.Bind` + `Router.Dispatch`.
* Old `SignalContext` is still available but considered legacy.

### 2. Managed Concurrency

**What Changed**: Goroutines must now be registered with `lifecycle.Go` to ensure proper tracking and cleanup.

**v1.4.x**:

```go
go func() {
    // Untracked goroutine - potential leak
}()
```

**v1.5.0**:

```go
lifecycle.Go(ctx, func(ctx context.Context) error {
    // Tracked, panic-safe, context-aware
    return nil
})
```

**Migration Path**:
* Replace all `go func()` spawned during application runtime with `lifecycle.Go`.
* The system will automatically wait for all tracked goroutines on shutdown.

### 3. Worker State Machine

**What Changed**: Workers now have explicit states (`Health`, `Suspended`, `Stopped`) managed by the `Supervisor`.

**v1.4.x**:

```go
// Workers were simple Start/Stop
type MyWorker struct{}
func (w *MyWorker) Start(ctx context.Context) error { ... }
func (w *MyWorker) Stop(ctx context.Context) error { ... }
```

**v1.5.0**:

```go
// Workers now support Suspend/Resume
type MyWorker struct{}
func (w *MyWorker) Start(ctx context.Context) error { ... }
func (w *MyWorker) Stop(ctx context.Context) error { ... }
func (w *MyWorker) Suspend(ctx context.Context) error { ... }
func (w *MyWorker) Resume(ctx context.Context) error { ... }
```

**Migration Path**:
* Implement `Suspend`/`Resume` methods if your worker needs to support temporary pauses.
* For workers that don't support suspension, return `ErrNotSupported`.
* See `examples/suspend/` for reference implementations.

### 4. Shutdown Phases

**What Changed**: Shutdown is now a multi-phase process with explicit hooks.

**v1.4.x**:

```go
// Simple context cancellation
ctx, cancel := context.WithCancel(ctx)
defer cancel()
```

**v1.5.0**:

```go
// Structured shutdown with phases: Startup → Running → Shutdown → Stopped
lifecycle.OnShutdown(func(ctx context.Context) error {
    // PreShutdown hook
    return nil
})

lifecycle.Run(func(ctx context.Context) error {
    // Running phase
    <-ctx.Done()
    // PostShutdown happens automatically
    return nil
})
```

**Migration Path**:
* Use `lifecycle.OnShutdown` to register cleanup hooks instead of manual `defer` chains.
* Hooks execute in LIFO order, similar to `defer`.
* All hooks must respect `ctx.Done()` to avoid blocking shutdown.

### 5. Context Propagation

**What Changed**: Context cancellation is now strictly enforced throughout the library.

**v1.4.x**:

```go
// Some operations ignored context
doWork() // Blocks indefinitely
```

**v1.5.0**:

```go
// All operations must respect context
doWork(ctx) // Returns on ctx.Done()
```

**Migration Path**:
* Audit all long-running operations to ensure they check `ctx.Done()`.
* Use `lifecycle.Sleep(ctx, duration)` instead of `time.Sleep`.
* Replace blocking I/O with `lifecycle.InterruptibleReader`.

## New Features (Non-Breaking)

### Event Sources

v1.5.0 introduces multiple event sources beyond OS signals:

* **`OSSignalSource`**: Traditional SIGINT/SIGTERM handling (default).
* **`FileWatchSource`**: React to configuration file changes.
* **`WebhookSource`**: Admin HTTP endpoints for remote control.
* **`HealthCheckSource`**: Self-monitoring and automatic recovery.
* **`InputSource`**: User commands from stdin (for CLIs).

**Example**:

```go
router := lifecycle.NewRouter()
router.AddSource(lifecycle.NewFileWatchSource("config.json"))
router.Bind("reload", lifecycle.ReloadHandler(reloadConfig))
```

### Progress Events

Track long-running operations with progress updates:

```go
lifecycle.Go(ctx, func(ctx context.Context) error {
    for i := 0; i < 100; i++ {
        ctx.Progress(float64(i) / 100.0)
        // ... work ...
    }
    return nil
})
```

### Introspection

Generate Mermaid diagrams of your application's lifecycle state:

```go
diagram := lifecycle.SystemDiagram()
fmt.Println(diagram) // Outputs Mermaid stateDiagram-v2
```

## Deprecations

The following APIs are deprecated but still functional:

* **`SignalContext`**: Use `lifecycle.Run` or `Router` instead.
* **Manual goroutine spawning**: Use `lifecycle.Go` for tracking.
* **Direct signal handling**: Use the `Router` event system.

These APIs will be removed in v2.0.0 (future major version).

## Upgrade Checklist

* [ ] Replace `SignalContext` with `lifecycle.Run` or `Router`.
* [ ] Audit all `go func()` calls and migrate to `lifecycle.Go`.
* [ ] Implement `Suspend`/`Resume` for workers that support pausing.
* [ ] Register cleanup logic with `lifecycle.OnShutdown`.
* [ ] Ensure all blocking operations respect `context.Context`.
* [ ] Test shutdown behavior under signal conditions (`SIGINT`, `SIGTERM`).
* [ ] Review `examples/` directory for reference implementations.

## Getting Help

* **Documentation**: See [TECHNICAL.md](TECHNICAL.md) for architecture details.
* **Examples**: The `examples/` directory contains runnable migration patterns.
* **Issues**: Report problems at <https://github.com/aretw0/lifecycle/issues>

## Version History Reference

* **v1.0 - v1.2**: Signal handling and terminal I/O primitives.
* **v1.3**: Introduction of Worker and Supervisor.
* **v1.4**: Reliability primitives (Critical Sections, Introspection).
* **v1.5**: Event-Driven Control Plane (this release).
* **v1.6+**: Planned ecosystem integrations (see [PLANNING.md](PLANNING.md)).
