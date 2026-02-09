package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	fmt.Println(">>> Lifecycle Demo Application")
	fmt.Println(">>> Press 'Enter' to echo text.")
	fmt.Println(">>> Press 'Ctrl+C' ONCE to see soft interrupt handling.")
	fmt.Println(">>> Press 'Ctrl+C' TWICE (or send SIGTERM) to exit immediately.")
	fmt.Println("---------------------------------------------------------------")

	// 1. Setup Signal Context (Using Root API)
	sigCtx := lifecycle.NewSignalContext(context.Background())
	defer sigCtx.Cancel() // Cleanup on exit

	// 2. Setup TermIO (Using Root API)
	stdin, err := lifecycle.OpenTerminal()
	if err != nil {
		panic(err)
	}
	// Wrap stdin with our interruptible reader linked to the context
	reader := lifecycle.NewInterruptibleReader(stdin, sigCtx.Done())

	// 3. Monitor Signals (The "Dual Signal" demonstration)
	// Since SignalContext doesn't cancel on SIGINT (it treats it as "Soft"),
	// the Reader linked to sigCtx.Done() will NOT unblock on Ctrl+C.
	// We monitor signals asynchronously to prove the app is still alive.
	go func() {
		ticker := time.NewTicker(2 * time.Second) // Slower tick
		defer ticker.Stop()
		for {
			select {
			case <-sigCtx.Done():
				return // Context cancelled (SIGTERM), exit monitor
			case <-ticker.C:
				// Pulse (silent or minimal)
			}

			// Check if we received a signal but are still running
			// removed spammy log
		}
	}()

	// 4. Main Input Loop
	buffer := make([]byte, 1024)
	sigIntCount := 0

	for {
		// Check global context first
		select {
		case <-sigCtx.Done():
			fmt.Println("\n>>> Main loop detected context cancellation (SIGTERM). Exiting...")
			return
		default:
		}

		fmt.Print("> ")

		// This Read will block until:
		// a) User types input + Enter
		// b) Context is cancelled (SIGTERM) -> Read returns ErrInterrupted
		n, err := reader.Read(buffer)

		// Check for interruption
		if lifecycle.IsInterrupted(err) {
			// Mitigation: Signal handler (goroutine) might be slightly slower than IO interruption (kernel).
			// If we don't see the signal in context yet, wait a tiny bit.
			if sigCtx.Signal() == nil {
				time.Sleep(50 * time.Millisecond)
			}

			if sig := sigCtx.Signal(); sig != nil {
				// Check if this signal actually cancelled the context (Hard Stop)
				select {
				case <-sigCtx.Done():
					fmt.Printf("\n>>> Interrupted by signal: %v. Exiting safely.\n", sig)
					return
				default:
					// Context still active -> Soft Interrupt (SIGINT)
					sigIntCount++
					if sigIntCount >= 2 {
						fmt.Printf("\n>>> Soft Interrupt (%v) x%d. Exiting manually.\n", sig, sigIntCount)
						sigCtx.Cancel() // Ensure cleanup
						return
					}
					fmt.Printf("\n>>> Soft Interrupt (%v) detected. Press Ctrl+C again to exit.\n", sig)
					time.Sleep(200 * time.Millisecond) // Prevent busy loop if Reader keeps erroring
					continue
				}
			}
			fmt.Println("\n>>> Interrupted by Context Cancel (or unknown).")
			return
		}

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		input := string(buffer[:n])
		fmt.Printf("Echo: %s", input)
		// sigIntCount = 0 // Reset counter on successful input? Maybe not, keep it sticky to allow quick double press?
		// Actually, standard behavior is usually sticky or time-based. Let's keep it sticky for simplicity.
	}
}
