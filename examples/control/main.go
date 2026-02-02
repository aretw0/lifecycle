package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// 1. Create the Control Router
	router := lifecycle.NewRouter()

	// 2. Register generic event handlers
	router.HandleFunc("signal.*", func(_ context.Context, e lifecycle.Event) error {
		fmt.Printf("[Router] Received Signal: %s\n", e)
		return nil
	})

	router.HandleFunc("tick.*", func(ctx context.Context, e lifecycle.Event) error {
		fmt.Printf("[Router] %s\n", e)
		return nil
	})

	// 3. Define the main application job
	job := lifecycle.Job(func(ctx context.Context) error {
		fmt.Println("Application started. Press Ctrl+C to exit.")

		// Start generic ticker source
		// In a real app, this might represent progress of a long running task
		ticker := lifecycle.NewTickerSource(500 * time.Millisecond)
		router.AddSource(ticker)

		// Start OS signal source
		sigSource := lifecycle.NewOSSignalSource(os.Interrupt, syscall.SIGTERM)
		router.AddSource(sigSource)

		// Create a background task using Managed Concurrency
		lifecycle.Go(ctx, func(ctx context.Context) error {
			fmt.Println("[Background] Task started (will run for 2s)")
			lifecycle.Sleep(ctx, 2*time.Second) // Safe sleep
			fmt.Println("[Background] Task finished")
			return nil
		})

		// Start the router (it blocks until ctx triggers shutdown)
		return router.Start(ctx)
	})

	// 4. Run the application
	if err := lifecycle.Run(job); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
