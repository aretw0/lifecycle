package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// 1. Create a SignalContext
	// This will handle SIGINT/SIGTERM and provide a context that cancels on shutdown.
	ctx := lifecycle.NewSignalContext(context.Background())
	defer ctx.Stop()

	// 2. Define our worker
	// We use 'ping' as a universally available long-running process for this example.
	// Arguments are adjusted for Windows (using -n) vs Linux/macOS (using -c).

	var cmdName string
	var args []string

	// Simple detection for Windows
	if os.PathSeparator == '\\' {
		cmdName = "ping"
		args = []string{"127.0.0.1", "-n", "30"}
	} else {
		cmdName = "ping"
		args = []string{"-c", "30", "127.0.0.1"}
	}

	fmt.Printf("Initializing Process Worker: %s %v\n", cmdName, args)
	procWorker := lifecycle.NewProcessWorker("pinger", cmdName, args...)

	// 3. Start the worker
	if err := procWorker.Start(ctx); err != nil {
		fmt.Printf("Failed to start worker: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Worker started: %s\n", procWorker)

	// 4. Wait for shutdown signal OR worker exit
	select {
	case <-ctx.Done():
		fmt.Println("\nShutdown signal received. Stopping worker...")
		// Give it a timeout to stop gracefully
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := procWorker.Stop(stopCtx); err != nil {
			fmt.Printf("Error stopping worker: %v\n", err)
		} else {
			fmt.Println("Worker stopped gracefully.")
		}

	case err := <-procWorker.Wait():
		if err != nil {
			fmt.Printf("Worker exited with error: %v\n", err)
		} else {
			fmt.Println("Worker exited successfully.")
		}
	}
}
