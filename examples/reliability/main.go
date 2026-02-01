package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// 1. Setup Signal Context
	ctx := lifecycle.NewSignalContext(context.Background())
	defer ctx.Cancel()

	fmt.Println("Example: Reliability Primitives (Critical Sections)")
	fmt.Println("Press Ctrl+C to test shielding...")
	fmt.Println("---------------------------------------------------")

	// 2. Simulate normal work
	work(ctx, "Phase 1: Normal Work")

	// 3. Enter Critical Section
	// Even if you hit Ctrl+C here, the inner function will complete.
	fmt.Println("\n>>> Entering Critical Section (Shielded) <<<")
	err := lifecycle.Do(ctx, func(innerCtx context.Context) {
		// Attempt to cancel during the critical section?
		// The innerCtx is NOT cancelled by the parent ctx.
		work(innerCtx, "Phase 2: Critical Transaction (Commit)")
	})

	if err != nil {
		fmt.Printf("\n>>> Lifecycle Error: %v <<<\n", err)
	} else {
		fmt.Println("\n>>> Critical Section Completed Successfully <<<")
	}

	// 4. Back to normal (or already cancelled if interrupted during shield)
	if ctx.Err() != nil {
		fmt.Println("Context was cancelled during the shield! Cleaning up now...")
	} else {
		work(ctx, "Phase 3: Cleanup")
	}

	fmt.Println("Done.")
}

func work(ctx context.Context, name string) {
	fmt.Printf("[%s] Starting...\n", name)
	select {
	case <-time.After(3 * time.Second):
		fmt.Printf("[%s] Finished.\n", name)
	case <-ctx.Done():
		fmt.Printf("[%s] Cancelled!\n", name)
	}
}
