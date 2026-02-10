package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/aretw0/lifecycle"
	"github.com/aretw0/lifecycle/pkg/events"
)

// EchoHandler is a simple handler that echoes the input line.
type EchoHandler struct{}

func (h *EchoHandler) HandleEvent(ctx context.Context, e events.Event) error {
	lineEvent, ok := e.(events.LineEvent)
	if !ok {
		return nil
	}
	fmt.Printf("Echo: %s\n", lineEvent.Line)
	return nil
}

func main() {
	// Setup Logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	// Intercept Handler (No-op for this example)
	intercept := events.HandlerFunc(func(ctx context.Context, e events.Event) error {
		fmt.Println("Interrupted! Press Ctrl+C again to force quit.")
		return nil
	})

	// Create Router with Passthrough
	router := lifecycle.NewInteractiveRouter(
		lifecycle.WithInterruptHandler(intercept),
		lifecycle.WithDefaultHandler(&EchoHandler{}),
		lifecycle.WithDefaultMappings(), // Enable 'quit' command
	)

	fmt.Println("Type something (or 'quit' to exit):")

	// Run
	// Note: NewInteractiveRouter returns a generic Router which has Start(ctx)
	// But usually we use lifecycle.Run or similar. Here we just Start it directly for the demo.
	ctx := context.Background()
	if err := router.Start(ctx); err != nil {
		slog.Error("Router failed", "error", err)
		os.Exit(1)
	}
}
