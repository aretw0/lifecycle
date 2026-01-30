package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	fmt.Println("=== Lifecycle v1.3: Ecosystem Interfaces & Handover Example ===")

	// 1. Create a Mock Container
	mock := lifecycle.NewMockContainer("redis-mock")

	// 2. Wrap it in a ContainerWorker
	redisWorker := lifecycle.NewContainerWorker("redis", mock)

	// 3. Define a Supervisor with a restart strategy
	// We use StrategyOneForOne to demonstrate the Handover Protocol on restart.
	sup := lifecycle.NewSupervisor("main-supervisor", lifecycle.StrategyOneForOne,
		lifecycle.SupervisorSpec{
			Name: "redis",
			Factory: func() (lifecycle.Worker, error) {
				return redisWorker, nil
			},
		},
	)

	// 4. Start the Supervisor
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("Starting supervisor...")
	if err := sup.Start(ctx); err != nil {
		panic(err)
	}

	// Wait a bit to see it running
	time.Sleep(1 * time.Second)
	state := sup.State()
	fmt.Printf("Supervisor State: %s\n", state.Status)
	for _, child := range state.Children {
		fmt.Printf("  Child %s: %s\n", child.Name, child.Status)
		if len(child.Metadata) > 0 {
			fmt.Println("    Metadata:")
			for k, v := range child.Metadata {
				fmt.Printf("      %s: %s\n", k, v)
			}
		}
	}

	fmt.Println("Stopping supervisor gracefully...")
	if err := sup.Stop(ctx); err != nil {
		fmt.Printf("Stop error: %v\n", err)
	}

	fmt.Println("Example finished.")
}
