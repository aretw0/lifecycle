package main

import (
	"context"
	"fmt"

	"github.com/aretw0/lifecycle"
	"github.com/aretw0/lifecycle/pkg/control"
)

func main() {
	// 1. Setup Signal Context with WithInterrupt(false)
	// This means Ctrl+C won't cancel the context automatically.
	// We also set WithForceExit(3) so the user can still kill it with 3x Ctrl+C.
	ctx := lifecycle.NewSignalContext(context.Background(),
		lifecycle.WithForceExit(0),
	)
	defer ctx.Stop()

	// 2. Setup Router and InputSource
	router := lifecycle.NewRouter()
	input := lifecycle.NewInputSource()
	router.AddSource(input)

	// 3. Define Shell State
	fmt.Println("🚀 REPL STARTED")
	fmt.Println("Commands: any text... | exit | quit")
	fmt.Println("Behavior: Ctrl+C clears the line (like a real shell)")
	fmt.Println("Unsafe Mode: WithForceExit(0) - Killswitch disabled")
	fmt.Print("> ")

	// 4. Handle Events
	// We use HandleFunc to register functions
	router.HandleFunc(lifecycle.ClearLineEvent{}.String(), func(ctx context.Context, e control.Event) error {
		fmt.Print("\n^C (Line Cleared)\n> ")
		return nil
	})

	router.HandleFunc(lifecycle.ShutdownEvent{}.String(), func(ctx context.Context, e control.Event) error {
		fmt.Println("\n👋 Shell exiting gracefully...")
		ctx.(*lifecycle.Context).Cancel()
		return nil
	})

	// Unmapped inputs (Generic InputEvent)
	// Pattern matching supports glob-like patterns.
	router.HandleFunc("input/*", func(ctx context.Context, ev control.Event) error {
		inputEv := ev.(lifecycle.InputEvent)
		fmt.Printf("Executing command: %q\n> ", inputEv.Command)
		return nil
	})

	// Start Router
	lifecycle.Go(ctx, router.Start)

	// Wait for context cancellation (from ShutdownEvent)
	<-ctx.Done()
	fmt.Println("Shell terminated.")
}
