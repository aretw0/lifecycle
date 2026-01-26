# Product Vision: Lifecycle

## Mission

To be the **standard foundation** for CLI application lifecycle management in the **Arbour ecosystem** (and beyond), abstracting away the pain of OS-specific signal handling, blocking I/O, and process orchestration.

## The Problem

Writing robust CLI tools in Go is deceptive. Handling `Ctrl+C` correctly implies:

1. **State Machines**: Is `Ctrl+C` a "Pause", a "Cancel Operation", or a "Force Quit"? The answer depends on what the app is doing.
2. **Windows Quirks**: On Windows, reading from `os.Stdin` can block signals from being delivered, or worse, closing the file handle can crash the runtime.
3. **Goroutine Leaks**: A `fmt.Scan()` inside a goroutine cannot be cancelled. If the main program quits, that goroutine leaks or hangs.

## The Solution

`lifecycle` provides:

* **Dual-Signal Context**: A standard pattern to differentiate "Soft Interrupt" vs "Hard Kill".
* **Safety Wrapper**: `pkg/termio` guarantees that I/O operations respect `context.Context`, implementing "Abandon Ship" logic for blocking reads.
* **Runtime Safety**: `pkg/runtime` prevents "Zombie Processes" and "Eternal Shutdowns" via strict timeouts and cleanup helpers.
* **Uniformity**: Ensures tools behave identically when the user tries to stop them.

## Target Audience

* **Framework Developers**: Building interactive CLIs (REPLs, TUI wizards).
* **Tool Builders**: Needing robust "Graceful Shutdown" for long-running processes.
