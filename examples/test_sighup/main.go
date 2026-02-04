package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	fmt.Println("=== SIGHUP Hot Reload Test ===")
	fmt.Printf("PID: %d\n", os.Getpid())
	fmt.Println("Send test signal with: kill -SIGHUP", os.Getpid())
	fmt.Println()

	router := lifecycle.NewRouter()

	// Add SIGHUP signal source
	router.AddSource(lifecycle.NewOSSignalSource(syscall.SIGHUP))

	// Handle reload with ReloadHandler
	reloadCount := 0
	router.Handle("signal.*", lifecycle.NewReloadHandler(func(ctx context.Context) error {
		reloadCount++
		fmt.Printf("✅ RELOAD #%d triggered at %s\n", reloadCount, time.Now().Format("15:04:05"))
		fmt.Println("   Simulating config reload...")
		time.Sleep(200 * time.Millisecond)
		fmt.Println("   Config reloaded successfully!")
		return nil
	}))

	// Also log all events for debugging
	router.HandleFunc("*", func(ctx context.Context, e lifecycle.Event) error {
		fmt.Printf("📨 Event received: %s\n", e.String())
		return nil
	})

	fmt.Println("Router started. Waiting for SIGHUP...")
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println()

	lifecycle.Run(router)

	fmt.Printf("\n👋 Shutdown complete. Total reloads: %d\n", reloadCount)
}
