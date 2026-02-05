package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// Use lifecycle.Run to manage the SignalContext automatically.
	// It handles:
	// 1. Context Creation (SIGINT/SIGTERM)
	// 2. Monitoring Goroutine Cleanup (Stop)
	// 3. Waiting for Hooks (Wait) on signal
	err := lifecycle.Run(lifecycle.Job(runApp))
	if err != nil && !lifecycle.IsInterrupted(err) {
		fmt.Printf("Exit error: %v\n", err)
	}
	fmt.Println("All cleanups done. Goodbye!")
}

func runApp(ctx context.Context) error {
	fmt.Println("Application started. Press Ctrl+C to shutdown.")

	// Register a simple hook (LIFO execution)
	// Use the facade helper to avoid manual casting.
	lifecycle.OnShutdown(ctx, func() {
		fmt.Println("[Hook 3] Fastest cleanup (runs last)")
	})

	// Register a slow hook to demonstrate Wait()
	lifecycle.OnShutdown(ctx, func() {
		fmt.Println("[Hook 2] Slow cleanup starting... (runs 2nd)")
		time.Sleep(1 * time.Second)
		fmt.Println("[Hook 2] Slow cleanup finished.")
	})

	// Dynamic Hook Registration
	lifecycle.OnShutdown(ctx, func() {
		fmt.Println("[Hook 1] Main cleanup (runs 1st)")

		// You can register more hooks from INSIDE a hook!
		// They will be executed immediately after this hook returns (LIFO-ish).
		lifecycle.OnShutdown(ctx, func() {
			fmt.Println("[Hook 1.1] Dynamic sub-task")
		})
	})

	// Simulate application work
	// Using the helper Sleep instead of select!
	if err := lifecycle.Sleep(ctx, 10*time.Second); err != nil {
		// To check the specific reasons (like IsInterrupted), we might generally just return err.
		// If we want the Reason string, we'd need to cast, but usually err is enough.
		// fmt.Printf("\nShutdown signal received! (Reason: %s)\n", sCtx.Reason())
		fmt.Printf("\nShutdown signal received! (%v)\n", err)
		return err
	}

	fmt.Println("\nTimeout! Exiting normally.")
	return nil
}



