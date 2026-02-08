# Ecosystem Integration: Trellis (Durable Execution)

This case study explores how **Trellis**, a durable execution engine for Go, uses `lifecycle` as its foundational control plane to ensure zero-loss operations and state-machine integrity.

## The Challenge

Durable execution requires that a process state can be suspended, moved, or restarted without losing the current progress of a workflow. Traditional signal handling (`SIGTERM`) is too binary for this: it usually leads to a hard shutdown, which might corrupt in-flight state-machine transitions if not handled with extreme care.

## The Lifecycle Solution

> [!WARNING]
> **Version Discrepancy**: The patterns described below represent the **Target Architecture** for Trellis v1.0.
> Currently, `trellis` (v0.7.1) depends on `lifecycle` v0.1.1 (legacy) and uses `lifecycle.NewSignalContext` mostly for basic interruption.
> The full "Suspend/Resume" integration described here is the roadmap for the next Trellis iteration.

Trellis integrates with `lifecycle` using three primary pillars: **Managed Suspension**, **Detached Execution**, and **Hierarchical Signaling**.

### 1. Robust Suspension (The Pause Button)

Trellis uses the `SuspendHandler` to coordinate state checkpointing. When a `SuspendEvent` is received (triggered via a Control API or a specific OS signal), Trellis pauses its task dispatchers.

```go
// Trellis Worker Integration
handler := lifecycle.NewRouter()
suspend := lifecycle.NewSuspendHandler()

// Register Trellis dispatchers
suspend.OnSuspend(func(ctx context.Context) error {
    log.Info("Trellis: Suspending dispatchers for checkpointing...")
    return engine.Checkpoint(ctx)
})

suspend.OnResume(func(ctx context.Context) error {
    log.Info("Trellis: Resuming dispatchers...")
    return engine.Resume()
})

handler.AddSource(lifecycle.NewOSSignalSource()) // Or custom control source
```

### 2. Durable Shutdowns with `DoDetached`

Some Trellis operations, like committing a transaction to the event store, **must not be interrupted** even if the main application logic has received a shutdown signal.

Trellis uses `lifecycle.DoDetached` for these "indiscriminate" operations:

```go
func (e *Engine) CommitState(ctx context.Context) error {
    // This work will continue even if the parent ctx is cancelled,
    // as long as the process is still running (within the ForceExit timeout).
    return lifecycle.DoDetached(ctx, func(detachedCtx context.Context) error {
        return e.store.Save(detachedCtx, e.currentState)
    })
}
```

### 3. Hierarchical Signals

Trellis workers often run as children of a supervisor. By using `lifecycle.Run`, Trellis ensures that:

- `SIGTERM` initiates a graceful drain.
- `SIGINT` (User Interrupt) can be differentiated to allow for local debugging/restarts.
- Child processes (if any) are cleaned up via Job Objects (Windows) or Subreapers (Linux).

## Key Patterns for Durable Workers

1. **Fail-Closed by default**: Assume the process can die at any time. Use `lifecycle` to maximize the "Grace Period".
2. **Idempotent Suspension**: Ensure `OnSuspend` can be called multiple times without side effects (guaranteed by `lifecycle.SuspendHandler` v2.0+).
3. **Observability**: Use `router.State()` to expose whether the durable worker is currently `suspended` or `running` to external monitoring tools.

## Conclusion

By offloading the "dirty work" of signal management and I/O coordination to `lifecycle`, Trellis can focus on its core value: **reliable state machines**.

---
> [!TIP]
> See `examples/suspend` for a runnable demonstration of the suspension pattern used in Trellis.
