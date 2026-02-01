package main

import (
	"context"
	"log/slog"
	"os"
	"syscall"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// 1. Setup interactions
	ctx := context.Background()

	// Configure Logger (Application Level)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	// Configure Library Logger & Metrics
	lifecycle.SetLogger(logger)
	lifecycle.SetMetricsProvider(lifecycle.NewLogMetricsProvider())

	// 2. Define the Control Router
	router := lifecycle.NewRouter()

	// 3. Add Sources
	// Source A: OS Signals (SIGINT, SIGTERM)
	sigSource := lifecycle.NewOSSignalSource(syscall.SIGINT, syscall.SIGTERM)
	router.AddSource(sigSource)

	// Source B: Simulated Webhook (Simulating an event after 2s)
	webhookSource := lifecycle.NewWebhookSource()
	router.AddSource(webhookSource)

	// 4. Register Reactions
	// When a signal arrives, we log it.
	// Note: We haven't implemented "pattern matching" yet, so we match the string representation.
	// This is just a demo of the wiring.
	router.On("Signal(interrupt)", func(ctx context.Context) error {
		slog.Info("Reaction: Received Interrupt! Initiating Shutdown...")
		return nil
	})

	// 5. Use Managed Concurrency for the Application
	g, _ := lifecycle.NewGroup(ctx)

	// Task 1: Run the Router
	g.Go(func(ctx context.Context) error {
		slog.Info("Router: Starting...")
		return router.Start(ctx)
	})

	// Task 2: Simulate a Panic (to test Recovery)
	g.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(1 * time.Second):
			// Uncomment to test panic recovery
			// panic("Oops! Something went wrong in a worker")
			slog.Info("Worker: Still alive...")
		}
		return nil
	})

	// Task 3: Simulate external webhook trigger
	g.Go(func(ctx context.Context) error {
		time.Sleep(2 * time.Second)
		slog.Info("Simulating Webhook Event...")
		// In a real scenario, this would come from the WebhookSource internals
		return nil
	})

	// Wait for everything
	slog.Info("Main: Waiting for group...")
	if err := g.Wait(); err != nil {
		slog.Error("Main: Group error", "error", err)
	}
	slog.Info("Main: Exited.")

}
