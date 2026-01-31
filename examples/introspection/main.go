package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// 1. Create a Signal Context
	ctx := lifecycle.NewSignalContext(context.Background())
	defer ctx.Stop()

	// 2. Create a Supervisor (The Root)
	sup := lifecycle.NewSupervisor("root-system", lifecycle.StrategyOneForAll)

	// 3. Add diverse workers
	// Process Worker
	sup.Add(lifecycle.SupervisorSpec{
		Name: "database",
		Type: "process",
		Factory: func() (lifecycle.Worker, error) {
			return lifecycle.NewProcessWorker("database", os.Args[0], "worker"), nil // Self-exec for demo
		},
	})

	// Functional Worker
	sup.Add(lifecycle.SupervisorSpec{
		Name: "health-check",
		Type: "func",
		Factory: func() (lifecycle.Worker, error) {
			return lifecycle.NewWorkerFromFunc("health-check", func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			}), nil
		},
	})

	// Container Worker (Mock)
	mock := lifecycle.NewMockContainer("redis-mock-id")
	sup.Add(lifecycle.SupervisorSpec{
		Name: "cache",
		Type: "container",
		Factory: func() (lifecycle.Worker, error) {
			return lifecycle.NewContainerWorker("cache", mock), nil
		},
	})

	fmt.Println("=== Introspection Demo (v1.3.1) ===")
	fmt.Println("1. Initial State (Pending):")
	fmt.Println(lifecycle.SystemDiagram(ctx.State(), sup.State()))

	// 4. Start Supervisor
	if err := sup.Start(ctx); err != nil {
		panic(err)
	}
	time.Sleep(100 * time.Millisecond) // Let them start

	fmt.Println("\n2. Running State:")
	fmt.Println(lifecycle.SystemDiagram(ctx.State(), sup.State()))

	// 5. Shutdown
	sup.Stop(ctx)
	<-sup.Wait()

	fmt.Println("\n3. Final State (Stopped):")
	fmt.Println(lifecycle.SystemDiagram(ctx.State(), sup.State()))
}

// Helper for the worker process simulation
func init() {
	if os.Getenv("GO_HELPER_PROCESS") == "1" {
		time.Sleep(100 * time.Millisecond) // Simulate work
		os.Exit(0)
	}
}
