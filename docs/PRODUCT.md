# Product Vision: Lifecycle

## Mission

To be the **standard foundation** for **Living Applications** (Services, Agents, CLIs) in the **Arbour ecosystem** (and beyond).
We abstract away the pain of "Death Management" (Shutdowns) and "Life Management" (Reactions), ensuring tools are both robust and dynamic.

## The Problem

Writing robust CLI tools in Go is deceptive. Handling `Ctrl+C` correctly implies:

1. **State Machines**: Is `Ctrl+C` a "Pause", a "Cancel Operation", or a "Force Quit"? The answer depends on what the app is doing.
2. **Windows Quirks**: On Windows, reading from `os.Stdin` can block signals from being delivered, or worse, closing the file handle can crash the runtime.
3. **Goroutine Leaks**: A `fmt.Scan()` inside a goroutine cannot be cancelled. If the main program quits, that goroutine leaks or hangs.

## The Solution

`lifecycle` provides:

* **Dual-Signal Context**: A standard pattern to differentiate "Soft Interrupt" vs "Hard Kill".
* **Context-Aware I/O**: Guarantees that I/O operations respect `context.Context` cancellation, preventing blocked goroutines and leaks.
* **Process Hygiene**: Prevents "Zombie Processes" and "Eternal Shutdowns" via deterministic timeouts and cleanup strategies.
* **Durable Execution**: Reliability primitives that shield critical state transitions from interruption.
* **Control Plane** (v2): An event-driven router that allows applications to react to external stimuli (Config Changes, Health Checks) without restarting.
* **Uniformity**: Ensures tools behave identically when the user tries to stop them.

## Target Audience

* **Framework Developers**: Building interactive CLIs (REPLs, TUI wizards).
* **Tool Builders**: Needing robust "Graceful Shutdown" for long-running processes.
