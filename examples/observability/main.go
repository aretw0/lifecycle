package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// 1. Setup Structured Logging (slog)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	lifecycle.SetLogger(logger)

	// 2. Setup Metrics Provider
	lifecycle.SetMetricsProvider(lifecycle.NewLogMetricsProvider())

	fmt.Println("--- Observability & Signal Demo ---")
	fmt.Println("This demo shows opinionated defaults with high flexibility.")
	fmt.Println("1. First Ctrl+C starts graceful shutdown.")
	fmt.Println("2. Second Ctrl+C forces an immediate exit.")

	// 3. Signal Context with functional options
	// Here we use the defaults, but we could do:
	// lifecycle.NewSignalContext(context.Background(), lifecycle.WithForceExit(3))
	ctx := lifecycle.NewSignalContext(context.Background(),
		lifecycle.WithForceExit(2), // Default: 2nd signal force exits
	)
	defer ctx.Stop()

	// 4. Start a dummy process
	cmd := exec.Command("ping", "127.0.0.1", "-n", "10")
	if err := lifecycle.StartProcess(cmd); err != nil {
		fmt.Printf("Failed to start process: %v\n", err)
	}

	// 5. Wait for signal or timeout
	select {
	case <-ctx.Done():
		if sig := ctx.Signal(); sig != nil {
			fmt.Printf("\nGraceful shutdown initiated by: %v\n", sig)
		}
	case <-time.After(15 * time.Second):
		fmt.Println("\nDemo finished by timeout.")
	}

	// 6. Demonstrate cleanup phase
	fmt.Println("\nSimulating a slow cleanup phase (3 seconds)...")
	fmt.Println("Try hitting Ctrl+C NOW to trigger the 'Force Exit' (Second signal).")

	done := make(chan struct{})
	go func() {
		time.Sleep(3 * time.Second)
		close(done)
	}()

	// Wait with 2s timeout (shorter than sleep) to show timeout warning
	_ = lifecycle.BlockWithTimeout(done, 2*time.Second)

	fmt.Println("\nDemo complete. Application exiting naturally.")
	os.Exit(0)
}



