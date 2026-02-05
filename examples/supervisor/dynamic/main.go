package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// 1. Setup Logger and Metrics
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	lifecycle.SetLogger(l)
	lifecycle.SetMetricsProvider(lifecycle.NewLogMetricsProvider())

	ctx := lifecycle.NewSignalContext(context.Background())
	defer ctx.Stop()

	// 2. Create Supervisor
	sup := lifecycle.NewSupervisor("root", lifecycle.SupervisorStrategyOneForOne)

	// 3. Start Supervisor
	lifecycle.SetLogger(l) // Ensure logger is set? It's global.
	l.Info("Starting supervisor...")
	if err := sup.Start(ctx); err != nil {
		l.Error("Failed to start supervisor", "error", err)
		return
	}

	// 4. Dynamic Add: A stable worker
	l.Info("Adding worker-1 (stable)")
	err := sup.Add(lifecycle.SupervisorSpec{
		Name: "worker-1",
		Factory: func() (lifecycle.Worker, error) {
			return lifecycle.NewWorkerFromFunc("worker-1", func(ctx context.Context) error {
				l.Info("Worker 1 running")
				<-ctx.Done()
				l.Info("Worker 1 stopping")
				return nil
			}), nil
		},
	})
	if err != nil {
		l.Error("Failed to add worker-1", "error", err)
	}

	time.Sleep(1 * time.Second)

	// 5. Dynamic Add: A failing worker (Demonstrate Backoff)
	l.Info("Adding worker-2 (unstable)")
	err = sup.Add(lifecycle.SupervisorSpec{
		Name: "worker-2",
		Backoff: lifecycle.SupervisorBackoff{
			InitialInterval: 500 * time.Millisecond,
			MaxInterval:     2 * time.Second,
			Multiplier:      2.0,
		},
		Factory: func() (lifecycle.Worker, error) {
			return lifecycle.NewWorkerFromFunc("worker-2", func(ctx context.Context) error {
				l.Info("Worker 2 running (will fail)")
				time.Sleep(200 * time.Millisecond)
				return fmt.Errorf("simulated failure")
			}), nil
		},
	})

	// Let it crash a few times
	time.Sleep(3 * time.Second)

	// 6. Dynamic Remove
	l.Info("Removing worker-2...")
	if err := sup.Remove("worker-2"); err != nil {
		l.Error("Failed to remove worker-2", "error", err)
	} else {
		l.Info("Worker-2 removed successfully")
	}

	// 7. Wait
	l.Info("Running for 2 more seconds...")
	time.Sleep(2 * time.Second)
	l.Info("Exiting...")
}



