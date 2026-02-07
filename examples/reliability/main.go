package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// 1. Setup Metrics (Log provider for easy viewing)
	lifecycle.SetMetricsProvider(lifecycle.NewLogMetricsProvider())

	// 3. Define the "Showcase" Supervisor
	// This supervisor will manage three types of workers to demonstrate reliability.

	// A. The Stable Worker: Never fails.
	stableFactory := func() (lifecycle.Worker, error) {
		return lifecycle.NewWorkerFromFunc("stable-worker", func(ctx context.Context) error {
			fmt.Println(" [✓] Stable worker started.")
			<-ctx.Done()
			fmt.Println(" [✓] Stable worker stopped.")
			return nil
		}), nil
	}

	// B. The Flaky Worker: Retries periodically but will eventually trigger the circuit breaker.
	flakyFactory := func() (lifecycle.Worker, error) {
		return lifecycle.NewWorkerFromFunc("flaky-worker", func(ctx context.Context) error {
			fmt.Println(" [!] Flaky worker starting...")
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(3 * time.Second):
				fmt.Println(" [!] Flaky worker CRASHING!")
				return errors.New("temporary failure")
			}
		}), nil
	}

	// C. The Critical Worker: If this fails, the app has issues.
	criticalFactory := func() (lifecycle.Worker, error) {
		return lifecycle.NewWorkerFromFunc("critical-worker", func(ctx context.Context) error {
			fmt.Println(" [★] Critical worker active.")
			<-ctx.Done()
			return nil
		}), nil
	}

	sup := lifecycle.NewSupervisor("Cluster-A", lifecycle.SupervisorStrategyOneForOne,
		lifecycle.SupervisorSpec{
			Name:    "stable-1",
			Type:    "process",
			Factory: stableFactory,
		},
		lifecycle.SupervisorSpec{
			Name:    "flaky-api",
			Type:    "container",
			Factory: flakyFactory,
			Backoff: lifecycle.SupervisorBackoff{
				InitialInterval: 500 * time.Millisecond,
				MaxInterval:     2 * time.Second,
				Multiplier:      2.0,
				MaxRestarts:     3, // Trigger circuit breaker after 3 restarts
				MaxDuration:     30 * time.Second,
			},
		},
		lifecycle.SupervisorSpec{
			Name:    "core-logic",
			Type:    "func",
			Factory: criticalFactory,
		},
	)

	// 6. Run Everything
	fmt.Println("================================================================")
	fmt.Println(" LIFECYCLE V2.0 - RELIABILITY SHOWCASE")
	fmt.Println("================================================================")
	fmt.Println(" This demo shows:")
	fmt.Println(" 1. Auto-healing (Supervisor)")
	fmt.Println(" 2. Protection (Circuit Breaker)")
	fmt.Println(" 3. Introspection (Mermaid via 'status' command)")
	fmt.Println(" 4. Control (interactive 's' to suspend, 'r' to resume, 'q' to quit)")
	fmt.Println("================================================================")
	fmt.Println(" Type 'status' to see the initial tree.")

	err := lifecycle.Run(lifecycle.Job(func(ctx context.Context) error {
		// Setup Interactive Router
		suspendHandler := lifecycle.NewSuspendHandler()

		router := lifecycle.NewInteractiveRouter(suspendHandler,
			lifecycle.WithShutdown(func() {
				fmt.Println("\n [!] Shutdown requested via command.")
				lifecycle.Shutdown(ctx)
			}),
			lifecycle.WithCommand("status", lifecycle.HandlerFunc(func(ctx context.Context, e lifecycle.Event) error {
				fmt.Println("--- LIVE TOPOLOGY DIAGRAM ---")
				fmt.Println(lifecycle.WorkerTreeDiagram(sup.State()))
				fmt.Println("----------------------------")
				return nil
			})),
		)

		// Wire up the Supervisor to the Suspend/Resume system
		suspendHandler.Manage(sup)

		// Start supervisor in background via lifecycle.Go
		lifecycle.Go(ctx, func(ctx context.Context) error {
			return sup.Start(ctx)
		})

		// Start the interactive router (blocks until 'q' or signals)
		return router.Start(ctx)
	}),
		lifecycle.WithShutdownTimeout(3*time.Second), // Parameterized shutdown diagnostics
	)

	if err != nil {
		slog.Error("Showcase exited with error", "error", err)
	}
}
