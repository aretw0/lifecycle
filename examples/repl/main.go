package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/aretw0/lifecycle"
	"github.com/aretw0/lifecycle/pkg/events"
)

const (
	// WithForceExit(0) disables the killswitch.
	THRESHOLD_UNSAFE = 0
	// WithForceExit(3) enables the killswitch.
	THRESHOLD_SAFE = 3
)

func showPrompt() {
	fmt.Print("> ")
}

func printHelp(forceExitThreshold int) {
	fmt.Println("Commands: any text... | exit | quit")
	fmt.Println("Behavior: Ctrl+C clears the line (like a real shell)")
	if forceExitThreshold <= 0 {
		fmt.Println("Unsafe Mode: WithForceExit(0) - Killswitch disabled")
	} else {
		fmt.Printf("Safe Mode: WithForceExit(%d) - Killswitch enabled\n", forceExitThreshold)
	}
	showPrompt()
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

	// 2. Setup Handlers (No Magic)
	clearLineHandler := lifecycle.HandlerFunc(func(ctx context.Context, e events.Event) error {
		fmt.Print("\n^C (Line Cleared via Escalator)\n")
		showPrompt()
		return nil // Successfully handled!
	})

	// Escalator: Primary = ClearLine, Fallback = Exit (from Context)
	// We use a stub fallback here because the SignalContext handles the "Force Exit" counting internally.
	// But to demonstrate the pattern:
	escalator := events.NewEscalator(clearLineHandler, lifecycle.HandlerFunc(func(ctx context.Context, e events.Event) error {
		fmt.Println("\nEscalator: Quitting...")
		return events.ErrNotHandled // Let the SignalContext take over or return error to stop
	}))

	// 3. Setup Router (Explicit Wiring)
	mux := events.NewRouter()

	// 3a. Wire Signal -> Escalator
	// Note: We use the string representation of the event we expect from the source.
	mux.Handle("Signal(interrupt)", escalator)

	// Define Commands and Handlers in one place
	commands := map[string]events.Handler{
		"help": lifecycle.HandlerFunc(func(ctx context.Context, e events.Event) error {
			printHelp(lifecycle.GetForceExitThreshold(ctx))
			return nil
		}),
		"quit": lifecycle.HandlerFunc(func(ctx context.Context, e events.Event) error {
			fmt.Println("👋 Shell exiting gracefully...")
			ctx.(*lifecycle.Context).Cancel()
			return nil
		}),
		"exit": lifecycle.HandlerFunc(func(ctx context.Context, e events.Event) error {
			fmt.Println("👋 Shell exiting gracefully...")
			ctx.(*lifecycle.Context).Cancel()
			return nil
		}),
	}

	// 3b. Wire Input
	// Use the shared map to configure valid inputs
	input := lifecycle.NewInputSource(
		lifecycle.WithInputHandlers(commands),
	)
	mux.AddSource(input)
	mux.AddSource(events.NewOSSignalSource(os.Interrupt))

	// 3. Define Shell State
	fmt.Println("🚀 REPL STARTED")
	printHelp(lifecycle.GetForceExitThreshold(ctx))

	// 4. Handle Events
	// Register the same handlers to the router
	// This maps "cmd" -> "command/cmd" route
	events.WithRouterHandlers(commands)(mux)

	// Remove explicit shutdown handler since it's now handled by the commands map
	// Remove Help handler since it's now handled by the commands map

	// Handle Unknown Commands from InputSource
	mux.HandleFunc(events.UnknownCommandEvent{}.String(), func(ctx context.Context, e events.Event) error {
		unknown := e.(events.UnknownCommandEvent)
		slog.Debug("Unknown command received", "cmd", unknown.Command)
		fmt.Printf(" [!] '%s' is not valid.\n", unknown.Command)
		showPrompt()
		return nil
	})

	// Unmapped inputs (Generic InputEvent)
	// Pattern matching supports glob-like patterns.
	mux.HandleFunc("input/*", func(ctx context.Context, ev events.Event) error {
		inputEv := ev.(lifecycle.InputEvent)
		fmt.Printf("Executing command: %q\n", inputEv.Command)
		showPrompt()
		return nil
	})

	// Start Router
	lifecycle.Go(ctx, mux.Start)

	// Wait for context cancellation (from ShutdownEvent)
	<-ctx.Done()
	fmt.Println("Shell terminated.")
}
