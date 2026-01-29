package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle/pkg/signal"
)

func main() {
	// 1. Create a configured SignalContext
	// We customize it to show how it affects the diagram
	ctx := signal.NewContext(
		context.Background(),
		signal.WithInterrupt(true),            // Default
		signal.WithForceExit(3),               // Require 3 signals to force exit
		signal.WithHookTimeout(2*time.Second), // Short timeout
	)
	defer ctx.Stop()

	// 2. Introspect the state
	state := ctx.State()

	// 3. Generate diagram
	diagram := signal.Mermaid(state)

	fmt.Println("--- Lifecycle Mermaid Diagram ---")
	fmt.Println(diagram)
	fmt.Println("---------------------------------")
	fmt.Println("Copy the above into https://mermaid.live to visualize!")
}
