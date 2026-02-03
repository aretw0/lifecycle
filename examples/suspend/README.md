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

## implementation

See `main.go` for the full implementation, specifically:

* `SmartSignalHandler`: Arbitrates between Suspend and Quit.
* `lifecycle.WithForceExit(2)`: Configures the safety net threshold.
