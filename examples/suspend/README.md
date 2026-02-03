# Suspend/Resume Example (The Factory)

This example demonstrates a complex, stateful application (a "Factory") that can be **Suspended** and **Resumed** without losing data.

It showcases:

* **State Persistence**: Saving/Loading JSON state on suspend/resume.
* **Worker Coornidation**: Coordinating multiple workers (Generator, Worker, Watchdog).
* **Smart Signal Handling**: Using `Ctrl+C` to trigger Suspend instead of immediate Exit.

## 🎮 Controls

* **`s` / `Ctrl+C`**: Suspend the factory (Stop production, save state).
* **`r`**: Resume production.
* **`q` / `Ctrl+C` (while suspended)**: Quit gracefully.

## 🛡️ Safety Net Pattern

This example uses the **Safety Net Pattern** for signal handling:

* **1st `Ctrl+C`**: Tries to Suspend gracefully.
* **2nd `Ctrl+C`**: Ignored (if already suspending) or Tries to Quit.
* **3rd `Ctrl+C`**: **Force Exit** (Immediate Kill).

This ensures that even if the Suspend logic hangs, you can always force-quit the application by mashing `Ctrl+C`.

## implementation

See `main.go` for the full implementation, specifically:

* `SmartSignalHandler`: Arbitrates between Suspend and Quit.
* `lifecycle.WithForceExit(3)`: Configures the safety net threshold.
