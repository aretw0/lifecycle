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
	// Output to Stdout in JSON format for production-like feel
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	lifecycle.SetLogger(logger)

	// 2. Setup Metrics Provider
	// We use the LogProvider for development to see metrics in logs
	lifecycle.SetMetricsProvider(lifecycle.NewLogMetricsProvider())

	fmt.Println("--- Observability Demo ---")
	fmt.Println("Check the JSON logs below to see observability in action.")
	fmt.Println("Press Ctrl+C to trigger signal metrics.")
	fmt.Println("The app will start a dummy process to trigger process metrics.")

	// 3. Signal Context with metrics
	ctx := lifecycle.NewSignalContext(context.Background())

	// 4. Start a process with metrics
	// We'll use 'ping' or 'timeout' as a dummy process
	cmd := exec.Command("ping", "127.0.0.1", "-n", "5")
	if err := lifecycle.StartProcess(cmd); err != nil {
		fmt.Printf("Failed to start process: %v\n", err)
	}

	// 5. Wait for signal or timeout
	select {
	case <-ctx.Done():
		if sig := ctx.Signal(); sig != nil {
			fmt.Printf("\nReceived signal: %v (Recorded in metrics)\n", sig)
		}
	case <-time.After(10 * time.Second):
		fmt.Println("\nDemo finished by timeout.")
	}

	// 6. Demonstrate cleanup timeout logging
	fmt.Println("Simulating a slow cleanup to trigger timeout log...")
	done := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Second)
		close(done)
	}()

	// This should log a warning because timeout is 1s and sleep is 2s
	_ = lifecycle.BlockWithTimeout(done, 1*time.Second)

	fmt.Println("Demo complete.")
}
