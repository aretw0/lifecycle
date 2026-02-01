package main

import (
	"context"
	"fmt" // Kept fmt as it's used in the new code
	"os"
	"syscall"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// 1. Create Router
	router := lifecycle.NewRouter()

	// 2. Register Sources
	// Signal Source: "signal.interrupt", "signal.terminated"
	sigSrc := lifecycle.NewOSSignalSource(os.Interrupt, syscall.SIGTERM)
	router.AddSource(sigSrc)

	// Webhook Source: "webhook.reload"
	// Try: curl -X POST http://localhost:8080/reload
	webSrc := lifecycle.NewWebhookSource(":8080") // Added port back based on context
	router.AddSource(webSrc)

	// 3. Register Handlers (Reactions)
	// For v2, let's assume we control the main context cancel.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Shutdown on Signal
	shutdownHandler := lifecycle.NewShutdownHandler(cancel)
	router.Handle("signal.*", shutdownHandler)

	// Reload on Webhook
	router.Handle("webhook.reload", lifecycle.HandlerFunc(func(ctx context.Context, e lifecycle.Event) error {
		fmt.Println("🔄 Reload triggered via Webhook!")
		fmt.Println("APP: Reloading configuration...")
		time.Sleep(500 * time.Millisecond) // Simulate work
		fmt.Println("✅ Configuration Reloaded")
		return nil
	}))

	// 4. Start Router
	fmt.Println("🚀 Control Plane Started. PID:", os.Getpid())
	fmt.Println("   - Waiting for SIGINT/SIGTERM...")
	fmt.Println("   - Listening for Webhooks on :8080...")

	if err := router.Start(ctx); err != nil {
		fmt.Printf("Router error: %v\n", err)
	}

	fmt.Println("👋 Shutdown Complete")
}
