// Package lifecycle provides a centralized library for managing application lifecycles
// and interactive I/O.
//
// # Dual Signal Context
//
// Standard Go `signal.NotifyContext` cancels on the first signal. `lifecycle` distinguishes between:
//   - SIGINT (Ctrl+C): "Soft" interrupt. Captures the signal but keeps the Context active.
//     Allows the application to decide whether to pause, confirm exit, or ignore.
//   - SIGTERM: "Hard" stop. Cancels the Context immediately, triggering graceful shutdown.
//
// # Interruptible I/O
//
// On many systems (especially Windows), reading from `os.Stdin` blocks the goroutine indefinitely,
// preventing clean cancellation. Furthermore, on Windows, receiving a signal can close the standard
// input handle, causing an unexpected EOF. `lifecycle` provides `OpenTerminal` (using `CONIN$`)
// and `NewInterruptibleReader` to ensure I/O operations respect `context.Context` cancellation
// and signals are handled gracefully without premature termination.
//
// # Shutdown Timeouts
//
// Graceful shutdown often involves waiting for background goroutines to finish (e.g., closing database
// connections, flushing logs). To prevent the application from hanging indefinitely if a cleanup
// operation stalls, `lifecycle` provides `BlockWithTimeout`. This ensures the process exits
// deterministically even if some components are stuck.
//
// # Worker Protocol & Supervision (v1.3)
//
// For complex applications, `lifecycle` provides a `Worker` interface and a `Supervisor`.
// This allows managing hierarchies of processes, goroutines, and containers with
// automatic restarts, session persistence (Handover Protocol), and unified introspection.
//
// # Usage
//
//	ctx := lifecycle.NewSignalContext(context.Background())
//	defer ctx.Cancel()
//
//	// Safe terminal reading
//	term, _ := lifecycle.OpenTerminal()
//	reader := lifecycle.NewInterruptibleReader(term, ctx.Done())
//
// See examples/demo for a full interactive application.
package lifecycle
