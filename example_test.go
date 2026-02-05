package lifecycle_test

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle"
)

// ExampleNewSignalContext demonstrates how to use the Dual Signal context.
// Note: This example is illustrative; in a real run, it waits for SIGINT/SIGTERM.
func ExampleNewSignalContext() {
	// Create a context that listens for signals.
	ctx := lifecycle.NewSignalContext(context.Background())

	// For checking output deterministically in this example, we cancel manually
	// after a short delay, allowing "work" to happen first.
	go func() {
		time.Sleep(50 * time.Millisecond)
		ctx.Cancel()
	}()

	// Simulate work
	select {
	case <-ctx.Done():
		fmt.Println("Context cancelled too early")
	case <-time.After(10 * time.Millisecond):
		fmt.Println("Doing work...")
	}

	// Output:
	// Doing work...
}

// ExampleOpenTerminal demonstrates how to open the terminal safely.
func ExampleOpenTerminal() {
	// OpenTerminal handles OS-specific logic (like CONIN$ on Windows)
	reader, err := lifecycle.OpenTerminal()
	if err != nil {
		fmt.Printf("Error opening terminal: %v\n", err)
		return
	}
	defer reader.Close()

	fmt.Println("Terminal opened successfully")

	// Wrap with InterruptibleReader to respect context cancellation
	// r := lifecycle.NewInterruptibleReader(reader, ctx.Done())

	// Output:
	// Terminal opened successfully
}

// ExampleBlockWithTimeout demonstrates how to enforce a deadline on shutdown cleanup.
func ExampleBlockWithTimeout() {
	done := make(chan struct{})

	// Simulate a cleanup task
	go func() {
		defer close(done)
		// Simulate fast cleanup
		time.Sleep(10 * time.Millisecond)
	}()

	// Wait for cleanup, but give up after 1 second
	err := lifecycle.BlockWithTimeout(done, 1*time.Second)
	if err != nil {
		fmt.Println("Cleanup timed out!")
	} else {
		fmt.Println("Cleanup finished successfully")
	}

	// Output:
	// Cleanup finished successfully
}



