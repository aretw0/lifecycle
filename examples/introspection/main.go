package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aretw0/lifecycle/pkg/signal"
	"github.com/aretw0/lifecycle/pkg/worker"
)

func main() {
	// 1. Create a Signal Context (Stateful)
	ctx := signal.NewContext(context.Background())
	defer ctx.Stop()

	// 2. Create a Worker (Stateful)
	w := worker.NewProcess("demo-worker", os.Args[0], "sleep")
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

func printDiagrams(ctx *signal.Context, w worker.Worker) {
	fmt.Println("\n--- Signal Context Diagram ---")
	fmt.Println(signal.Mermaid(ctx.State()))

	fmt.Println("\n--- Worker Diagram ---")
	fmt.Println(worker.Mermaid(w.State()))
}

// Helper for the worker process simulation
func init() {
	if os.Getenv("GO_HELPER_PROCESS") == "1" {
		time.Sleep(100 * time.Millisecond) // Simulate work
		os.Exit(0)
	}
}
