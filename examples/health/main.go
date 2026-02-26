package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/supervisor"
	"github.com/aretw0/lifecycle/pkg/core/worker"
)

// HealthWorker is a custom worker that implements the Prober interface.
type HealthWorker struct {
	worker.BaseWorker
	healthy bool
	msg     string
}

func NewHealthWorker(name string) *HealthWorker {
	w := &HealthWorker{
		BaseWorker: *worker.NewBaseWorker(name),
		healthy:    true,
		msg:        "System Normal",
	}
	// Demonstrate setting worker type
	w.SetType(worker.TypeGoroutine)
	return w
}

func (w *HealthWorker) Start(ctx context.Context) error {
	return w.StartFunc(ctx, func(innerCtx context.Context) error {
		w.SetStatus(worker.StatusRunning)

		// Simulate health changes
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		count := 0
		for {
			select {
			case <-innerCtx.Done():
				return nil
			case <-ticker.C:
				count++
				w.Lock()
				if count%2 == 0 {
					w.healthy = true
					w.msg = fmt.Sprintf("System Normal (cycle %d)", count)
				} else {
					w.healthy = false
					w.msg = fmt.Sprintf("Load High (cycle %d)", count)
				}
				w.Unlock()
			}
		}
	})
}

// Probe implements the worker.Prober interface.
func (w *HealthWorker) Probe(ctx context.Context) worker.ProbeResult {
	w.Lock()
	defer w.Unlock()
	return worker.ProbeResult{
		Healthy:   w.healthy,
		Message:   w.msg,
		Timestamp: time.Now(),
	}
}

func main() {
	fmt.Println("=== Lifecycle Health Probing Example ===")

	// 1. Create a supervisor with our custom healthy worker
	sup := supervisor.New("main-supervisor", supervisor.StrategyOneForOne)

	hw := NewHealthWorker("database-engine")
	sup.Add(supervisor.Spec{
		Name:    "database-engine",
		Type:    string(worker.TypeGoroutine),
		Factory: func() (worker.Worker, error) { return hw, nil },
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := sup.Start(ctx); err != nil {
		fmt.Printf("Error starting supervisor: %v\n", err)
		os.Exit(1)
	}

	// 2. Monitor state and print health
	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nExample finished.")
			sup.Stop(context.Background())
			return
		case <-ticker.C:
			state := sup.State()
			fmt.Printf("\n--- State Snapshot at %s ---\n", time.Now().Format("15:04:05"))

			for _, child := range state.Children {
				healthStr := "Healthy ❤️"
				if child.Health != nil && !child.Health.Healthy {
					healthStr = "Unhealthy 💔"
				}

				fmt.Printf("Worker: %-15s | Status: %-10s | Type: %-10s | Health: %s (%s)\n",
					child.Name, child.Status, child.Type, healthStr, child.Health.Message)
				fmt.Printf("  Restarts: %d | StartedAt: %s\n",
					child.Restarts, child.StartedAt.Format(time.Kitchen))
			}

			// 3. Demonstrate Mermaid output
			fmt.Println("\nMermaid Snippet (Health Integrated):")
			fmt.Println(worker.MermaidTree(state))
		}
	}
}
