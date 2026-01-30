package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// 1. Create a Signal Context (Stateful)
	ctx := lifecycle.NewSignalContext(context.Background())
	defer ctx.Stop()

	// 2. Create Workers (Stateful)
	p := lifecycle.NewProcessWorker("demo-process", os.Args[0], "worker")

	// Create a Mock Container
	mock := lifecycle.NewMockContainer("redis-mock")
	c := lifecycle.NewContainerWorker("redis-container", mock)

	fmt.Println("=== Introspection Demo (v1.3.1) ===")
	fmt.Println("1. Initial States:")
	fmt.Println("\n--- Unified System Dashboard ---")
	fmt.Println(lifecycle.SystemDiagram(ctx.State(), p.State())) // Single worker for now

	// Start Workers
	if err := p.Start(ctx); err != nil {
		panic(err)
	}
	if err := c.Start(ctx); err != nil {
		panic(err)
	}

	fmt.Println("\n2. Running States (Multiple Workers):")
	// SystemDiagram currently only takes one root worker state.
	// In a real app, you'd have a Supervisor.
	// Let's print the Mermaid tree for a manual "composite" or just the container
	fmt.Println("\n--- Container Diagnostic Snapshot ---")
	fmt.Println(lifecycle.WorkerTreeDiagram(c.State()))

	fmt.Println("\n--- Process Diagnostic Snapshot ---")
	fmt.Println(lifecycle.WorkerTreeDiagram(p.State()))

	// Stop Workers
	p.Stop(ctx)
	c.Stop(ctx)
	<-p.Wait()
	<-c.Wait()

	fmt.Println("\n3. Final States:")
	fmt.Println("\n--- Unified System Dashboard ---")
	fmt.Println(lifecycle.SystemDiagram(ctx.State(), p.State()))
}

// Helper for the worker process simulation
func init() {
	if os.Getenv("GO_HELPER_PROCESS") == "1" {
		time.Sleep(100 * time.Millisecond) // Simulate work
		os.Exit(0)
	}
}
