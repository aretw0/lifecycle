# Lifecycle Worker State Machine

This document defines the authoritative state machine for `lifecycle` Workers. It establishes the semantic meaning of each state, the allowed transitions, and the philosophy behind "Intent vs Outcome".

## Core Philosophy: Intent vs Outcome

To resolve ambiguity between "finished" (success) and "stopped" (interruption), we distinguish based on **termination intent**:

1. **Natural Termination**: The worker finished its task because it was done. -> **Finished**
2. **Requested Termination**: The worker finished because it was asked to stop. -> **Stopped**
3. **Error Termination**: The worker finished because it crashed or errored. -> **Failed**

## The States

### 1. Created `(StatusCreated)`

* **Definition**: The worker instance exists and is initialized, but `Start()` has not been called.
* **Context**: Resources (structs) are allocated, but operational resources (goroutines, file handles) are not.
* **Next Valid**: `Starting`, `Pending`.

### 2. Pending `(StatusPending)`

* **Definition**: The worker is blocked from execution by an active governor (e.g., Supervisor backoff).
* **Context**: Waiting for a restart delay timer, rate limit slot, or circuit breaker reset.
* **Next Valid**: `Starting`, `Stopped` (if cancelled while pending).

### 3. Starting `(StatusStarting)`

* **Definition**: `Start()` has been called, and initialization logic is executing.
* **Context**: Opening connections, spawning background goroutines. This state is transient.
* **Next Valid**: `Running`, `Stopped`, `Failed` (if init fails).

### 4. Running `(StatusRunning)`

* **Definition**: The worker is actively executing its primary workload.
* **Context**: The main loop is active.
* **Next Valid**: `Suspended`, `Stopping`, `Stopped`, `Finished`, `Failed`.

### 5. Suspended `(StatusSuspended)`

* **Definition**: The worker has paused its execution (quiescence) but retains its resources.
* **Context**: Traffic is drained, no new work is accepted, but memory state is preserved.
* **Next Valid**: `Running` (via Resume), `Stopping` (via Stop).

### 6. Stopping `(StatusStopping)`

* **Definition**: `Stop()` has been called (or Context cancelled), and the worker is performing graceful shutdown.
* **Context**: Draining final requests, closing connections, flushing buffers.
* **Next Valid**: `Stopped`, `Failed`, `Killed`.

### 7. Stopped `(StatusStopped)`

* **Semantic**: **Compliance**.
* **Definition**: The worker terminated successfully (exit code 0 / no error) **IN RESPONSE** to a stop request or context cancellation.
* **Logic**: `StopRequested == true` OR `termio.IsInterrupted(err)`.

### 8. Finished `(StatusFinished)`

* **Semantic**: **Completion**.
* **Definition**: The worker terminated successfully (exit code 0 / no error) **WITHOUT** being requested to stop.
* **Logic**: `StopRequested == false` AND `err == nil`.

### 9. Failed `(StatusFailed)`

* **Semantic**: **Error**.
* **Definition**: The worker terminated with a non-nil error or non-zero exit code.
* **Logic**: `err != nil` (and not an Interruption).

### 10. Killed `(StatusKilled)`

* **Semantic**: **Non-Compliance / Obstinacy**.
* **Definition**: The worker failed to stop within the grace period and was forcefully terminated.
* **Logic**: `Killed == true`.

---

## Transition Matrix

| From | To | Trigger |
| :--- | :--- | :--- |
| **Created** | `Starting` | `Start()` called directly. |
| **Created** | `Pending` | Supervisor delays start (Backoff). |
| **Pending** | `Starting` | Backoff timer expires. |
| **Pending** | `Stopped` | Context cancelled while waiting. |
| **Starting** | `Running` | Initialization complete. |
| **Starting** | `Failed` | Initialization error. |
| **Running** | `Suspended` | `Suspend()` called. |
| **Running** | `Stopping` | `Stop()` or Context cancelled. |
| **Running** | `Finished` | Main loop returns `nil`. |
| **Running** | `Failed` | Main loop returns `error`. |
| **Suspended** | `Running` | `Resume()` called. |
| **Suspended** | `Stopping` | `Stop()` called while suspended. |
| **Stopping** | `Stopped` | Graceful shutdown complete. |
| **Stopping** | `Failed` | Shutdown error (non-timeout). |
| **Stopping** | `Killed` | Shutdown timeout (forced kill). |

---

## Implementation

The `BaseWorker` struct centralizes the state logic to ensure consistency across all worker types (`Process`, `Func`, `Container`).

### 1. State Fields

Workers track their **Termination Intent** using `BaseWorker` fields:

```go
type BaseWorker struct {
    // ...
    status        Status
    StopRequested bool  // Was Stop() called?
    Killed        bool  // Was it force-killed?
    Err           error // Did it exit with an error?
}
```

### 2. Runtime Transitions (`SetStatus`)

During execution, workers use `SetStatus(new)` to update their state. This method handles locking and emits `StateChange` events for introspection.

```go
// Example in ProcessWorker.Start
p.SetStatus(StatusRunning)
```

### 3. Terminal Logic (`Finish`)

When a worker exits (run loop returns), it calls `Finish(err)`. This method calculates the final status using `DeriveFinalStatus()`:

```go
func (b *BaseWorker) DeriveFinalStatus() Status {
    if b.Killed {
        return StatusKilled
    }
    // Interrupted errors (ContextCanceled, Terminated) are considered "Stopped"
    if b.Err != nil && termio.IsInterrupted(b.Err) {
        return StatusStopped
    }
    if b.Err != nil {
        return StatusFailed
    }
    if b.StopRequested {
        return StatusStopped
    }
    return StatusFinished
}
```

This ensures that a worker which was asked to stop (`StopRequested`) and exits cleanly (`err == nil`) is correctly classified as `Stopped`, distinguishing it from one that finished naturally (`Finished`).

---

## Design Rationale

### Why not `StatusCancelled`?

`StatusCancelled` typically mirrors `context.Canceled`. In `lifecycle`, context cancellation is the *mechanism* for stopping, not a distinct state. We map cancellation to `StatusStopped` (Compliance) to simplify Supervisor logic.

### Why `StatusKilled`?

Distinguishing `Killed` from `Failed` is crucial for reliability engineering. A `Failed` worker might be buggy, but a `Killed` worker is **obstinate**—it hangs during shutdown. This distinction allows Supervisors to apply different policies (e.g., alert on Killed, retry on Failed).
