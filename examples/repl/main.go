package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aretw0/lifecycle"
	"github.com/aretw0/lifecycle/pkg/events"
)

const (
	// WithForceExit(0) disables the killswitch.
	THRESHOLD_UNSAFE = 0
	// WithForceExit(3) enables the killswitch.
	THRESHOLD_SAFE = 3
)

func printHelp(forceExitThreshold int) {
	fmt.Println("Commands: any text... | exit | quit")
	fmt.Println("Behavior: Ctrl+C clears the line (like a real shell)")
	if forceExitThreshold <= 0 {
		fmt.Println("Unsafe Mode: WithForceExit(0) - Killswitch disabled")
	} else {
		fmt.Printf("Safe Mode: WithForceExit(%d) - Killswitch enabled\n", forceExitThreshold)
	}
	fmt.Println("> ")
}

func main() {
	// 1. Setup Signal Context with interactive mode
	// WithCancelOnInterrupt(false) means Ctrl+C won't cancel the context automatically.
	// We also set WithForceExit(3) so the user can still kill it with 3x Ctrl+C.
	ctx := lifecycle.NewSignalContext(context.Background(),
		lifecycle.WithForceExit(THRESHOLD_SAFE),
		lifecycle.WithCancelOnInterrupt(false), // Interactive mode: Ctrl+C handled manually
	)
	defer ctx.Stop()

	// 2. Setup Handlers & Router
	clearLineHandler := lifecycle.HandlerFunc(func(ctx context.Context, e events.Event) error {
		fmt.Print("\n^C (Line Cleared)\n> ")
		return nil
	})

	// Use NewInteractiveRouter with a stateless clearLineHandler.
	// This will not trigger "Suspending..." logs or persistent suspended state.
	mux := lifecycle.NewInteractiveRouter(clearLineHandler,
		lifecycle.WithInput(false), // We will add our own source below
	)

	// Create custom InputSource with our UnknownHandler
	input := lifecycle.NewInputSource(
		lifecycle.WithUnknownHandler(func(cmd string, known []string) {
			slog.Warn("Unknown command received",
				"command", cmd,
				"available", known,
			)
			fmt.Printf(" [!] '%s' is not valid. Try one of: %v\n> ", cmd, known)
		}),
	)
	mux.AddSource(input)

	// 3. Define Shell State
	fmt.Println("🚀 REPL STARTED")
	printHelp(lifecycle.GetForceExitThreshold(ctx))

	// 4. Handle Events
	// We use HandleFunc to register functions
	mux.HandleFunc(lifecycle.InterceptEvent{}.String(), clearLineHandler)

	mux.HandleFunc(lifecycle.ShutdownEvent{}.String(), func(ctx context.Context, e events.Event) error {
		fmt.Println("\n👋 Shell exiting gracefully...")
		ctx.(*lifecycle.Context).Cancel()
		return nil
	})

	// Unmapped inputs (Generic InputEvent)
	// Pattern matching supports glob-like patterns.
	mux.HandleFunc("input/*", func(ctx context.Context, ev events.Event) error {
		inputEv := ev.(lifecycle.InputEvent)
		fmt.Printf("Executing command: %q\n> ", inputEv.Command)
		return nil
	})

	// Start Router
	lifecycle.Go(ctx, mux.Start)

	// Wait for context cancellation (from ShutdownEvent)
	<-ctx.Done()
	fmt.Println("Shell terminated.")
}
