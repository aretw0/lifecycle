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
// # Reliability Primitives (v1.4)
//
// For "Durable Execution" patterns, `lifecycle` provides `Do(ctx, fn)`. This creates a
// "Critical Section" where value-critical operations (like state commits) are shielded
// from cancellation. If a user hits Ctrl+C during a critical section, the signal is
// captured but deferred until the section completes.
//
// # Worker Protocol & Supervision (v1.3)
//
// For complex applications, `lifecycle` provides a `Worker` interface and a `Supervisor`.
// This allows managing hierarchies of processes, goroutines, and containers with
// automatic restarts, session persistence (Handover Protocol), and unified introspection.
//
// # DX Helpers & Boilerplate (v1.4)
//
// To reduce friction, `lifecycle` provides helpers that standardizes common patterns:
//   - Run: Standardizes the `main` function (Context creation -> Run -> Stop -> Wait).
//   - Sleep: Replaces `select { case <-time.After... }` with a single context-aware call.
//   - OnShutdown: Registers hooks without manual type assertions.
//
// # Usage
//
//	func main() {
//		err := lifecycle.Run(runApp)
//		if err != nil {
//			// handle err
//		}
//	}
//
//	func runApp(ctx context.Context) error {
//		// Register cleanup
//		lifecycle.OnShutdown(ctx, func() {
//			fmt.Println("Closing DB...")
//		})
//
//		// Safe Sleep (Regret Window)
//		if err := lifecycle.Sleep(ctx, 5*time.Second); err != nil {
//			return err
//		}
//		return nil
//	}
//
// See examples/hooks for a full application structure.
package lifecycle
