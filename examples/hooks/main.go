package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// Create a SignalContext
	// This captures SIGINT (Ctrl+C) and SIGTERM.
	ctx := lifecycle.NewSignalContext(context.Background())
	defer ctx.Stop() // Ensure we stop the monitoring goroutine

	fmt.Println("Application started. Press Ctrl+C to shutdown.")

	// Register a simple hook (LIFO execution)
	ctx.OnShutdown(func() {
		fmt.Println("[Hook 3] Fastest cleanup (runs last)")
	})

	// Register a slow hook to demonstrate Wait()
	ctx.OnShutdown(func() {
		fmt.Println("[Hook 2] Slow cleanup starting... (runs 2nd)")
		time.Sleep(1 * time.Second)
		fmt.Println("[Hook 2] Slow cleanup finished.")
	})

	// Dynamic Hook Registration
	ctx.OnShutdown(func() {
		fmt.Println("[Hook 1] Main cleanup (runs 1st)")

		// You can register more hooks from INSIDE a hook!
		// They will be executed immediately after this hook returns (LIFO-ish).
		ctx.OnShutdown(func() {
			fmt.Println("[Hook 1.1] Dynamic sub-task")
		})
	})

	// Simulate application work
	select {
	case <-ctx.Done():
		fmt.Printf("\nShutdown signal received! (Reason: %s)\n", ctx.Reason())
	case <-time.After(10 * time.Second):
		fmt.Println("\nTimeout! Exiting normally.")
	}

	// CRITICAL: Block until all hooks have finished.
	// If you forget this, the program might exit while "Hook 2" is still sleeping!
	fmt.Println("Waiting for hooks...")
	ctx.Wait()
	fmt.Println("All cleanups done. Goodbye!")
}
