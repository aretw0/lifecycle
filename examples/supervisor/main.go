package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"log/slog"

	"github.com/aretw0/lifecycle"
)

func main() {
	// 1. Setup Lifecycle
	ctx := lifecycle.NewSignalContext(context.Background())

	metricProvider := lifecycle.NewLogMetricsProvider()
	lifecycle.SetMetricsProvider(metricProvider)

	// Optional: Bridge library logs (internal events like restarts) to your application logger.
	// If you don't call this, the library uses its default logger (or discards if not configured).
	// logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	// lifecycle.SetLogger(logger)

	// 2. Define Worker Factories
	tickerFactory := func() (lifecycle.Worker, error) {
		return lifecycle.NewWorkerFromFunc("Ticker", func(ctx context.Context) error {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return nil
				case t := <-ticker.C:
					fmt.Printf("[%s] Tick: %v\n", "Ticker", t.Format(time.Kitchen))
				}
			}
		}), nil
	}

	crasherFactory := func() (lifecycle.Worker, error) {
		return lifecycle.NewWorkerFromFunc("Crasher", func(ctx context.Context) error {
			fmt.Println("[Crasher] I will crash in 2 seconds...")
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Second):
				fmt.Println("[Crasher] BOOM!")
				return errors.New("unexpected crash")
			}
		}), nil
	}

	// 3. Create Supervisor
	sup := lifecycle.NewSupervisor("MainSup", lifecycle.SupervisorStrategyOneForOne,
		lifecycle.SupervisorSpec{Name: "Ticker", Factory: tickerFactory},
		lifecycle.SupervisorSpec{Name: "Crasher", Factory: crasherFactory},
	)

	// 4. Start Supervisor
	if err := sup.Start(ctx); err != nil {
		slog.Error("Failed to start supervisor", "error", err)
		os.Exit(1)
	}

	// 5. Wait for Shutdown
	// We wait for the signal context to be done (graceful shutdown)
	<-ctx.Done()

	fmt.Println("\n--- Introspection (Recursive Tree) ---")
	fmt.Println(lifecycle.WorkerTreeDiagram(sup.State()))

	fmt.Println("\n--- Introspection (Worker FSM) ---")
	// Using a dummy state for illustration as we don't have easy access to children instances here
	// without creating them.
	dummyState := lifecycle.WorkerState{Name: "ExampleWorker", Status: lifecycle.WorkerStatusRunning, PID: 1234}
	fmt.Println(lifecycle.WorkerStateDiagram(dummyState))

	// 6. Stop Supervisor
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	slog.Info("Shutting down supervisor...")
	if err := sup.Stop(shutdownCtx); err != nil {
		slog.Error("Failed to stop supervisor", "error", err)
	} else {
		slog.Info("Supervisor stopped successfully")
	}
}
