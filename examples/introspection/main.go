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

	// 2. Create a Worker (Stateful)
	w := lifecycle.NewProcessWorker("demo-worker", os.Args[0], "sleep")
	// Inject helper env to make strict sleep mock
	os.Setenv("GO_HELPER_PROCESS", "1")
	defer os.Unsetenv("GO_HELPER_PROCESS")

	fmt.Println("=== Introspection Demo ===")
	fmt.Println("1. Initial States:")
	printDiagrams(ctx, w)

	// Start Worker
	if err := w.Start(ctx); err != nil {
		panic(err)
	}

	fmt.Println("\n2. Running States:")
	printDiagrams(ctx, w)

	// Stop Worker
	w.Stop(ctx)
	<-w.Wait()

	fmt.Println("\n3. Final States:")
	printDiagrams(ctx, w)
}

func printDiagrams(ctx *lifecycle.Context, w lifecycle.Worker) {
	fmt.Println("\n--- Signal Context Diagram ---")
	fmt.Println(lifecycle.SignalStateDiagram(ctx.State()))

	fmt.Println("\n--- Worker Tree Diagram ---")
	fmt.Println(lifecycle.WorkerTreeDiagram(w.State()))

	fmt.Println("\n--- Worker State Diagram ---")
	fmt.Println(lifecycle.WorkerStateDiagram(w.State()))
}

// Helper for the worker process simulation
func init() {
	if os.Getenv("GO_HELPER_PROCESS") == "1" {
		time.Sleep(100 * time.Millisecond) // Simulate work
		os.Exit(0)
	}
}
