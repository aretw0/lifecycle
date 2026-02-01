package main

import (
	"context"
	"fmt" // Kept fmt as it's used in the new code
	"os"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// 1. Use Default Router (Stdlib Pattern)
	// router := lifecycle.NewRouter() -> No longer needed for default usage

	// 2. Register Sources
	// Note: OS Signals are handled automatically by lifecycle.Run!
	// We only need to register extra sources like Webhooks.

	// Webhook Source: "webhook.reload"
	// Try: curl -X POST http://localhost:8080/reload
	webSrc := lifecycle.NewWebhookSource(":8080")
	lifecycle.DefaultRouter.AddSource(webSrc)

	// 3. Register Handlers (Reactions)
	// lifecycle.Run handles the context and cancellation for us.

	// Reload on Webhook
	lifecycle.HandleFunc("webhook.reload", func(ctx context.Context, e lifecycle.Event) error {
		fmt.Println("🔄 Reload triggered via Webhook!")
		fmt.Println("APP: Reloading configuration...")
		time.Sleep(500 * time.Millisecond) // Simulate work
		fmt.Println("✅ Configuration Reloaded")
		return nil
	})

	// 4. Start Router via Run (Managed Concurrency)
	fmt.Println("🚀 Control Plane Started. PID:", os.Getpid())
	fmt.Println("   - Waiting for SIGINT/SIGTERM...")
	fmt.Println("   - Listening for Webhooks on :8080...")

	// Use lifecycle.Run to manage the lifecycle of the DefaultRouter
	if err := lifecycle.Run(lifecycle.DefaultRouter); err != nil {
		fmt.Printf("Exit error: %v\n", err)
	}

	fmt.Println("👋 Shutdown Complete")
}
