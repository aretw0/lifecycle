package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// 1. Create a configured SignalContext
	// We customize it to show how it affects the diagram
	ctx := lifecycle.NewSignalContext(
		context.Background(),
		lifecycle.WithInterrupt(true),            // Default
		lifecycle.WithForceExit(3),               // Require 3 signals to force exit
		lifecycle.WithHookTimeout(2*time.Second), // Short timeout
	)
	defer ctx.Stop()

	// 2. Introspect the state
	state := ctx.State()

	// 3. Generate diagram
	diagram := lifecycle.Mermaid(state)

	fmt.Println("--- Lifecycle Mermaid Diagram ---")
	fmt.Println(diagram)
	fmt.Println("---------------------------------")
	fmt.Println("Copy the above into https://mermaid.live to visualize!")
}
