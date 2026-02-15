package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// Level 3: Control Plane
	// This example demonstrates the Event-Driven architecture of lifecycle v1.5+.
	// - Routing Events (Webhooks)
	// - Managed Concurrency (lifecycle.Run)

	// 1. Setup Sources
	// Generic signals are handled by Run(), but we add a WebhookSource.
	// Try: curl -X POST http://localhost:8080/reload
	webSrc := lifecycle.NewWebhookSource(":8080")
	lifecycle.DefaultRouter.AddSource(webSrc)

	// 2. Register Handlers
	// Pattern matching works like net/http
	lifecycle.DefaultRouter.HandleFunc("webhook.reload", func(ctx context.Context, e lifecycle.Event) error {
		fmt.Println("🔄 Reload triggered via Webhook!")
		fmt.Println("APP: Reloading configuration...")
		time.Sleep(500 * time.Millisecond) // Simulate work
		fmt.Println("✅ Configuration Reloaded")
		return nil
	})

	// 3. Start Application
	fmt.Println("🚀 Control Plane Started. PID:", os.Getpid())
	fmt.Println("   - Waiting for SIGINT/SIGTERM...")
	fmt.Println("   - Listening for Webhooks on :8080...")

	// Run manages the DefaultRouter's lifecycle
	if err := lifecycle.Run(lifecycle.DefaultRouter); err != nil {
		fmt.Printf("Exit error: %v\n", err)
	}

	fmt.Println("👋 Shutdown Complete")
}
