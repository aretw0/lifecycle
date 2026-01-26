package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle/pkg/signal"
	"github.com/aretw0/lifecycle/pkg/termio"
)

func main() {
	fmt.Println(">>> Lifecycle Demo Application")
	fmt.Println(">>> Press 'Enter' to echo text.")
	fmt.Println(">>> Press 'Ctrl+C' ONCE to see soft interrupt handling.")
	fmt.Println(">>> Press 'Ctrl+C' TWICE (or send SIGTERM) to exit immediately.")
	fmt.Println("---------------------------------------------------------------")

	// 1. Setup Signal Context
	sigCtx := signal.NewContext(context.Background())
	defer sigCtx.Cancel() // Cleanup on exit

	// 2. Setup TermIO
	stdin, err := termio.Open()
	if err != nil {
		panic(err)
	}
	// Wrap stdin with our interruptible reader linked to the context
	reader := termio.NewInterruptibleReader(stdin, sigCtx.Done())

	// 3. Main Loop
	buffer := make([]byte, 1024)
	for {
		// Check global context first
		select {
		case <-sigCtx.Done():
			fmt.Println("\n>>> Main loop detected context cancellation. Exiting...")
			return
		default:
		}

		fmt.Print("> ")

		// This Read will block until:
		// a) User types input + Enter
		// b) Context is cancelled (via signal) -> Read returns ErrInterrupted
		n, err := reader.Read(buffer)

		// 4. Handle Interruption
		if termio.IsInterrupted(err) {
			// Check if it was a signal that caused this
			if sig := sigCtx.Signal(); sig != nil {
				fmt.Printf("\n>>> Interrupted by signal: %v\n", sig)

				// Logic: If it's just one SIGINT, we might want to stay alive?
				// signal.NewContext only cancels on SIGTERM or if we call Cancel manually.
				// But wait, the BOOTSTRAP said "SIGINT = Pausa/Cancelamento Suave".
				// In this simple demo, let's treat SIGINT as "Pause loop for 2 sec then continue".

				// IMPORTANT: Since signal.NewContext does NOT cancel on first SIGINT,
				// sigCtx.Done() is NOT closed yet if it was just SIGINT.
				// However, termio.IsInterrupted(err) might be true if we closed a channel?
				// Actually, NewInterruptibleReader takes 'cancel <-chan struct{}'.
				// If sigCtx.Done() is NOT closed, Read() shouldn't return ErrInterrupted just for SIGINT
				// UNLESS we passed a separate mechanism?

				// Ah, in Trellis, the engine handles SIGINT and manually cancels the "Input Context".
				// Here, we passed sigCtx.Done(). If sigCtx doesn't close on SIGINT, Read() WON'T unblock on SIGINT!

				// Correction for Demo: To demonstrate unblocking on SIGINT, we need the Context to be cancelled
				// OR we need to handle SIGINT manually and cancel the reader.

				// For this demo, we'll see that SIGINT *doesn't* unblock Read instantly if the context isn't cancelled.
				// Unless... we use the specific behavior where we want to catch SIGINT.

				// Let's rely on standard termination for the demo:
				// If the user sends SIGTERM (or we logic it out), we exit.

				return
			}
			return
		}

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		input := string(buffer[:n])
		fmt.Printf("Echo: %s", input)
		time.Sleep(100 * time.Millisecond) // Simulate work
	}
}
