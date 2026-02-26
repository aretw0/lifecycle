package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aretw0/lifecycle"
)

// This example demonstrates "Chained Cancels" (ADR-0016).
// It shows how a worker can spawn a child process with a specific timeout
// that is derived from the worker's own context.
// If the timeout expires or the main app shuts down, the child is reaped correctly.

func main() {
	// 1. Standard lifecycle setup
	// We use Job() to wrap our logic into a Runnable
	app := lifecycle.Job(func(ctx context.Context) error {
		slog.Info("Main application started. Press Ctrl+C to abort early.")

		// 2. Start a background task that manages a child process
		lifecycle.Go(ctx, func(taskCtx context.Context) error {
			// 3. Create a Chained Context with a 2-second timeout
			// This context is derived from taskCtx, which is derived from main ctx.
			chainedCtx, cancel := context.WithTimeout(taskCtx, 2*time.Second)
			defer cancel()

			slog.Info("Starting child process with a 2-second 'Self-Destruct' timeout...")

			// 4. Spawn child via lifecycle.NewProcessCmd with chainedCtx
			// This automatically handles hygiene and context-linked cancellation.
			var name string
			var args []string
			if os.Getenv("OS") == "Windows_NT" {
				name = "timeout"
				args = []string{"/t", "10"}
			} else {
				name = "sleep"
				args = []string{"10"}
			}

			cmd := lifecycle.NewProcessCmd(chainedCtx, name, args...)

			if err := cmd.Start(); err != nil {
				return fmt.Errorf("failed to start child: %w", err)
			}

			slog.Info("Child started", "pid", cmd.Process.Pid)

			// 5. Wait for child to finish
			// Because we used NewProcessCmd(chainedCtx), the OS process
			// will be reaped automatically when chainedCtx expires or is cancelled.
			err := cmd.Wait()

			if err != nil {
				slog.Warn("Child process finished (expected error due to timeout)", "error", err)
			} else {
				slog.Info("Child process finished cleanly")
			}

			return nil
		})

		// Keep main alive for a bit to see the example run
		slog.Info("Main task keeping app alive for 5s...")
		return lifecycle.Sleep(ctx, 5*time.Second)
	})

	if err := lifecycle.Run(app); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}
