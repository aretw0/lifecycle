package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aretw0/lifecycle"
)

func main() {
	// Simple detection for Windows
	var cmdName string
	var args []string
	if os.PathSeparator == '\\' {
		cmdName = "ping"
		args = []string{"127.0.0.1", "-n", "30"}
	} else {
		cmdName = "ping"
		args = []string{"-c", "30", "127.0.0.1"}
	}

	fmt.Printf("Initializing Process Worker: %s %v\n", cmdName, args)
	procWorker := lifecycle.NewProcessWorker("pinger", cmdName, args...)

	// Use lifecycle.Run for automatic signal handling and cleanup
	err := lifecycle.Run(lifecycle.Job(func(ctx context.Context) error {
		fmt.Printf("Worker starting: %s\n", procWorker)
		fmt.Println("Press Ctrl+C to exit.")

		// Start the worker in background
		lifecycle.Go(ctx, procWorker.Start)

		// Wait for shutdown or worker exit
		select {
		case <-ctx.Done():
			fmt.Println("\nShutdown signal received. Process worker will stop via internal lifecycle cleanup.")
			return nil
		case err := <-procWorker.Wait():
			return err
		}
	}))

	if err != nil && err != context.Canceled {
		fmt.Printf("Exited with error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Graceful exit.")
}
