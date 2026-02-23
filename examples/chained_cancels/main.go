package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/aretw0/lifecycle"
	"github.com/aretw0/procio/proc"
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

			// 4. Spawn child via standard exec with chainedCtx
			// We use 'ping' or 'timeout' as a dummy long-running process
			var cmd *exec.Cmd
			if os.Getenv("OS") == "Windows_NT" {
				cmd = exec.CommandContext(chainedCtx, "timeout", "10")
			} else {
				cmd = exec.CommandContext(chainedCtx, "sleep", "10")
			}

			// Use proc.Start to ensure process hygiene (PDeathSig/Job Objects)
			if err := proc.Start(cmd); err != nil {
				return fmt.Errorf("failed to start child: %w", err)
			}

			slog.Info("Child started", "pid", cmd.Process.Pid)

			// 5. Wait for child to finish
			// Because we used exec.CommandContext(chainedCtx), the OS process
			// will be signalled/killed by the Go runtime when chainedCtx expires or is cancelled.
			err := cmd.Wait()

			if err != nil {
				slog.Warn("Child process finished with error (expected due to timeout)", "error", err)
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
