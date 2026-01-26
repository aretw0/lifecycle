// Package lifecycle provides a centralized library for managing application lifecycles
// and interactive I/O within the aretw0 ecosystem (Trellis, Tobot, Fiscus).
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
// preventing clean cancellation. `lifecycle` provides `OpenTerminal` (using `CONIN$`) and
// `NewInterruptibleReader` to ensure I/O operations respect `context.Context` cancellation.
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
