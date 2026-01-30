package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aretw0/lifecycle"
	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/supervisor"
	"github.com/aretw0/lifecycle/pkg/worker"
)

// MockWorker implements worker.Worker for testing.
type MockWorker struct {
	Name  string
	Logic func(ctx context.Context) error

	waitChan chan error
}

func (w *MockWorker) Start(ctx context.Context) error {
	w.waitChan = make(chan error, 1)
	go func() {
		defer close(w.waitChan)
		w.waitChan <- w.Logic(ctx)
	}()
	return nil
}

func (w *MockWorker) Stop(ctx context.Context) error {
	// Our Logic should handle context cancellation, so manual stop isn't strictly needed
	// unless we want to force something.
	return nil
}

func (w *MockWorker) Wait() <-chan error {
	return w.waitChan
}

func (w *MockWorker) String() string {
	return w.Name
}

func (w *MockWorker) State() worker.State {
	return worker.State{Name: w.Name, Status: worker.StatusRunning}
}

func main() {
	// 1. Setup Lifecycle
	ctx := lifecycle.NewSignalContext(context.Background(), lifecycle.WithInterrupt(true))

	metricProvider := lifecycle.NewLogMetricsProvider()
	lifecycle.SetMetricsProvider(metricProvider)

	// Default is Info, if we want Debug:
	// logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	// lifecycle.SetLogger(logger)

	// 2. Define Worker Factories
	tickerFactory := func() (worker.Worker, error) {
		return &MockWorker{
			Name: "Ticker",
			Logic: func(ctx context.Context) error {
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
			},
		}, nil
	}

	crasherFactory := func() (worker.Worker, error) {
		return &MockWorker{
			Name: "Crasher",
			Logic: func(ctx context.Context) error {
				fmt.Println("[Crasher] I will crash in 2 seconds...")
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(2 * time.Second):
					fmt.Println("[Crasher] BOOM!")
					return errors.New("unexpected crash")
				}
			},
		}, nil
	}

	// 3. Create Supervisor
	sup := supervisor.New("MainSup", supervisor.StrategyOneForOne,
		supervisor.Spec{Name: "Ticker", Factory: tickerFactory},
		supervisor.Spec{Name: "Crasher", Factory: crasherFactory},
	)

	// 4. Start Supervisor
	if err := sup.Start(ctx); err != nil {
		log.Error("Failed to start supervisor", "error", err)
		os.Exit(1)
	}

	// 5. Wait for Shutdown
	// We wait for the signal context to be done (graceful shutdown)
	<-ctx.Done()

	// 6. Stop Supervisor
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Info("Shutting down supervisor...")
	if err := sup.Stop(shutdownCtx); err != nil {
		log.Error("Failed to stop supervisor", "error", err)
	} else {
		log.Info("Supervisor stopped successfully")
	}
}
