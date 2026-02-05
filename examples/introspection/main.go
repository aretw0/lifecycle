package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aretw0/lifecycle"
	"github.com/aretw0/lifecycle/pkg/adapters/mermaid"
	"github.com/aretw0/lifecycle/pkg/introspection"
	"github.com/aretw0/lifecycle/pkg/signal"
	"github.com/aretw0/lifecycle/pkg/worker"
)

func main() {
	// 1. Setup lifecycle context (The Control Plane)
	// We use a threshold of 2 to demonstrate "Double-Tap" logic:
	// 1st Ctrl+C -> Cancels context (Graceful Shutdown)
	// 2nd Ctrl+C -> os.Exit(1) (Force Exit)
	ctx := lifecycle.NewSignalContext(context.Background(),
		lifecycle.WithForceExit(2),
	)

	// 2. Setup supervisor (The Data Plane)
	sup := lifecycle.NewSupervisor("app", lifecycle.SupervisorStrategyOneForOne,
		lifecycle.SupervisorSpec{
			Name: "ticker",
			Type: "func",
			Factory: func() (lifecycle.Worker, error) {
				return lifecycle.NewWorkerFromFunc("ticker", func(ctx context.Context) error {
					ticker := time.NewTicker(1 * time.Second)
					defer ticker.Stop()
					count := 0
					for {
						select {
						case <-ticker.C:
							count++
							fmt.Printf("  ⏱️  Tick %d\n", count)
						case <-ctx.Done():
							// fmt.Printf("  🛑 Ticker worker received STOP signal (last tick: %d)\n", count)
							// Simulate some cleanup work
							time.Sleep(500 * time.Millisecond)
							return nil
						}
					}
				}), nil
			},
		},
		lifecycle.SupervisorSpec{
			Name: "short-job",
			Type: "func",
			Factory: func() (lifecycle.Worker, error) {
				return lifecycle.NewWorkerFromFunc("short-job", func(ctx context.Context) error {
					fmt.Println("  📝 Short job starting...")
					time.Sleep(2 * time.Second)
					fmt.Println("  ✅ Short job completed")
					return nil
				}), nil
			},
			RestartPolicy: lifecycle.RestartNever,
		},
	)

	// Print initial state diagram
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("📊 INITIAL STATE")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(mermaid.SystemDiagram(ctx.State(), sup.State()))

	// Aggregate ALL state changes
	watchCtx, cancelWatch := context.WithCancel(context.Background())
	defer cancelWatch()

	snapshots := introspection.AggregateWatchers(watchCtx, ctx, sup)

	// Watch for state changes in background
	go func() {
		for snapshot := range snapshots {
			var output strings.Builder
			output.WriteString(fmt.Sprintf("\n📡 EVENT [%s]: %s changed", snapshot.Timestamp.Format("15:04:05.000"), snapshot.ComponentID))

			switch payload := snapshot.Payload.(type) {
			case worker.State:
				output.WriteString(fmt.Sprintf(" -> Status: %s\n", payload.Status))
			case signal.State:
				output.WriteString(fmt.Sprintf(" -> Signals: %d (Reason: %s, Threshold: %d)\n",
					payload.SignalCount, payload.Reason, payload.ForceExitThreshold))
			}

			// Render updated diagram
			output.WriteString(mermaid.SystemDiagram(ctx.State(), sup.State()))
			output.WriteString("\n")
			output.WriteString(strings.Repeat("-", 40))
			output.WriteString("\n")

			fmt.Print(output.String())
		}
	}()

	// Start supervisor
	fmt.Println("\n🚀 Starting supervisor...")
	if err := sup.Start(ctx); err != nil {
		fmt.Printf("CRITICAL: Supervisor failed to start: %v\n", err)
		return
	}

	fmt.Print("\n⏳ Press Ctrl+C once for GRACEFUL shutdown, twice to FORCE exit\n\n")

	// Wait for context cancellation (SIGINT/SIGTERM)
	<-ctx.Done()
	fmt.Println("\n🛑 Graceful shutdown initiated in main (Ctrl+C received)")

	// The supervisor will stop automatically because its context is linked to 'ctx'.
	// We just need to wait for it to finish gracefully.

	fmt.Println("⏳ Waiting for supervisor and children to terminate...")
	select {
	case <-sup.Wait():
		fmt.Println("✅ Supervisor finished successfully")
	case <-time.After(5 * time.Second):
		fmt.Println("⚠️  Timeout waiting for supervisor finish")
	}

	// Give the event aggregator plenty of time to flush the last events
	// (Signal received, Stopping transitions, Worker exits, Final Stopped state)
	time.Sleep(1 * time.Second)

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("📊 FINAL STATE (Captured from components)")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(mermaid.SystemDiagram(ctx.State(), sup.State()))

	fmt.Println("\n👋 Demo ended.")
}
