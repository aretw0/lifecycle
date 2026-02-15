# Suspend/Resume Example (The Factory)

This example demonstrates a complex, stateful application (a "Factory") that can be **Suspended** and **Resumed** without losing data.

It showcases:

* **State Persistence**: Saving/Loading JSON state on suspend/resume.
* **Worker Coornidation**: Coordinating multiple workers (Generator, Worker, Watchdog).
* **Smart Signal Handling**: Using `Ctrl+C` to trigger Suspend instead of immediate Exit.

## 🎮 Controls

* **`s`**: Suspend the factory via input.
* **`r`**: Resume production via input.
* **`Ctrl+C` (1st)**: Tries to Suspend gracefully (Escalation Mode).
* **`Ctrl+C` (2nd)**: **Force Exit** (Immediate Kill).
* **`q`**: Quit gracefully via input.

## 🛡️ Safety Net Pattern

This example uses the **Safety Net Pattern** for signal handling:

* **1st `Ctrl+C`**: Tries to Suspend gracefully.
* **2nd `Ctrl+C`**: **Force Exit** (Immediate Kill).

This ensures that even if the Suspend logic hangs, you can always force-quit the application by mashing `Ctrl+C`.

## 📁 Project Structure

* **`shared/`**: Common logic (persistence, shared workers, factory runner).
* **`cond/`**: Implementation using `sync.Cond`. Best for legacy code or simple wait/signal requirements.
* **`channels/`**: Implementation using channels and `select`. Idiomatic v1.5+ style, highly composable and better integration with Go's concurrency model.

## 🚀 Running

To run the idiomatic channel-based version:

```bash
go run ./examples/suspend/channels
```

To run the `sync.Cond` version:

```bash
go run ./examples/suspend/cond
```

## 🛠️ Implementation Details

Each version implements different suspension strategies:

### 1. `sync.Cond` (Legacy Style)

Uses a shared mutex and condition variable. Note that `cond.Wait()` cannot be easily cancelled by context, so it requires a wrapper or manual loop check.

### 2. Channels (v1.5+ Style)

Uses `select` to listen for `suspend`/`resume` signals. Fully context-aware and integrates seamlessly with `lifecycle.BaseWorker`.
