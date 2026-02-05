package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aretw0/lifecycle"
	"github.com/aretw0/lifecycle/pkg/control"
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
	suspendHandler := lifecycle.NewSuspendHandler()

	// Use NewInteractiveRouter but disable default input so we can inject our custom one
	router := lifecycle.NewInteractiveRouter(suspendHandler,
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
	router.AddSource(input)

	// 3. Define Shell State
	fmt.Println("🚀 REPL STARTED")
	printHelp(lifecycle.GetForceExitThreshold(ctx))

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
