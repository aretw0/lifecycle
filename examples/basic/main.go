package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// 1. Wrap your application logic in a Job
	// lifecycle.Run handles:
	// - Signal listening (Ctrl+C, SIGTERM)
	// - Context cancellation
	// - Waiting for background tasks
	err := lifecycle.Run(lifecycle.Job(run))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func run(ctx context.Context) error {
	fmt.Println("Application started. Press Ctrl+C to exit.")

	// 2. Use lifecycle.Go for background tasks
	// This ensures the goroutine is tracked and waited for on shutdown.
	lifecycle.Go(ctx, func(ctx context.Context) error {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("Background task stopping...")
				return nil
			case t := <-ticker.C:
				fmt.Printf("Tick: %v\n", t.Format(time.TimeOnly))
			}
		}
	})

	// 3. Block until context is cancelled (by signal)
	<-ctx.Done()
	fmt.Println("Main context cancelled. Cleaning up...")

	// Simulate brief cleanup
	time.Sleep(500 * time.Millisecond)

	return nil
}



