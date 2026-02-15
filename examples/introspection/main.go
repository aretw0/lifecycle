package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aretw0/lifecycle"

	"github.com/aretw0/introspection"
	"github.com/aretw0/lifecycle/pkg/core/signal"
	"github.com/aretw0/lifecycle/pkg/core/worker"
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
							fmt.Printf("  [TICK] %d\n", count)
						case <-ctx.Done():
							fmt.Printf("  🛑 Ticker worker received STOP signal (last tick: %d)\n", count)
							// Simulate some cleanup work
							time.Sleep(3 * time.Second)
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
					fmt.Println("  [JOB] Short job starting...")
					time.Sleep(3 * time.Second)
					fmt.Println("  [JOB] Short job completed")
					return nil
				}), nil
			},
			RestartPolicy: lifecycle.RestartNever,
		},
	)

	// SYNC FIX: Register a dummy hook in the controller that lasts as long as worker cleanup.
	// This ensures the Lifecycle Controller stays in "Stopping" while workers are also stopping.
	ctx.OnShutdown(func() {
		// Matching worker cleanup time for visual synchronization
		time.Sleep(3 * time.Second)
	})

	// Print initial state diagram
	var initialOutput strings.Builder
	initialOutput.WriteString(strings.Repeat("=", 60))
	initialOutput.WriteString("\nSNAPSHOT: INITIAL STATE\n")
	initialOutput.WriteString(strings.Repeat("=", 60))
	initialOutput.WriteString("\n")
	initialOutput.WriteString(lifecycle.SystemDiagram(ctx.State(), sup.State()))
	initialOutput.WriteString("\n")
	fmt.Print(initialOutput.String())

	// Aggregate ALL state changes
	watchCtx, cancelWatch := context.WithCancel(context.Background())
	defer cancelWatch()

	snapshots := introspection.AggregateWatchers(watchCtx, ctx, sup)

	// DEBOUNCER: Throttles bursts of updates to avoid "snapshot dumps"
	// (Especially useful when multiple components transition at the same time)
	go func() {
		var lastUpdate time.Time
		timer := time.NewTimer(0)
		if !timer.Stop() {
			<-timer.C
		}

		var pending bool

		for {
			select {
			case snapshot, ok := <-snapshots:
				if !ok {
					return
				}

				// Just log the atomic event briefly
				eventDesc := ""
				switch payload := snapshot.Payload.(type) {
				case worker.State:
					eventDesc = fmt.Sprintf("Status: %s", payload.Status)
				case signal.State:
					eventDesc = fmt.Sprintf("Signals: %d", payload.SignalCount)
				}
				fmt.Printf("EVENT [%s]: %s -> %s\n",
					snapshot.Timestamp.Format("15:04:05.000"), snapshot.ComponentID, eventDesc)

				pending = true

				// Debounce logic: reset timer on every event
				timer.Reset(100 * time.Millisecond)

			case <-timer.C:
				if !pending {
					continue
				}

				// Limit output frequency even with constant updates
				if time.Since(lastUpdate) < 200*time.Millisecond {
					timer.Reset(100 * time.Millisecond)
					continue
				}

				// Render settled state diagram
				var output strings.Builder
				output.WriteString(strings.Repeat("-", 40))
				output.WriteString("\nSNAPSHOT: SYSTEM STATE\n")
				output.WriteString(lifecycle.SystemDiagram(ctx.State(), sup.State()))
				output.WriteString("\n")
				output.WriteString(strings.Repeat("-", 40))
				output.WriteString("\n")
				fmt.Print(output.String())

				lastUpdate = time.Now()
				pending = false
			}
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
	time.Sleep(1 * time.Second)

	var finalOutput strings.Builder
	finalOutput.WriteString(strings.Repeat("=", 60))
	finalOutput.WriteString("\nSNAPSHOT: FINAL STATE\n")
	finalOutput.WriteString(strings.Repeat("=", 60))
	finalOutput.WriteString("\n")
	finalOutput.WriteString(lifecycle.SystemDiagram(ctx.State(), sup.State()))
	finalOutput.WriteString("\n")
	fmt.Print(finalOutput.String())

	fmt.Println("\n👋 Demo ended.")
}
