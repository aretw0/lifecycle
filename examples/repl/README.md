# Shell/REPL Example (The Infinite Loop)

This example demonstrates a interactive command-line application (a "Shell") that uses **Escalation Mode** to capture `Ctrl+C` without quitting.

It showcases:

* **Interrupt Capturing**: `Ctrl+C` emits a `ClearLineEvent` instead of cancelling the context.
* **Input Resilience**: The input reader remains active on Windows even after multiple interruputs.
* **Custom Shutdown Commands**: Explicitly handling `exit` or `quit` to trigger a graceful shutdown.
* **Force Exit Safety Net**: Mashing `Ctrl+C` N times will still kill the process if it hangs.

## 🎮 Controls

* **`Ctrl+C` (1st to N-1th)**: Clears the current line (emits `lifecycle/clear-line`).
* **`Ctrl+C` (N-th)**: **Force Exit** (Immediate Kill).
* **`exit` / `quit`**: Graceful shutdown.

## Implementation

See `main.go` for the full implementation, specifically:

* `lifecycle.WithForceExit(3)`: Configures the shell to allow 2 "clears" and kill on the 3rd interrupt.
* `router.HandleFunc("lifecycle/clear-line", ...)`: UI reaction to the interrupt.
