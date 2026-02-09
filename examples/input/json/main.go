package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	// Demonstrate Raw Input Handling (e.g. for JSON streams)
	// Usage: echo '{"type":"ping"}' | go run examples/input/json/main.go

	// Configure source to handle raw lines
	source := lifecycle.NewInputSource(
		lifecycle.WithInputReader(os.Stdin),
		lifecycle.WithRawInput(func(line string) {
			fmt.Printf("RECEIVED RAW: %s\n", line)
			// In a real app, you would json.Unmarshal here
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("Listening for JSON input on Stdin... (Ctrl+C to exit)")
	if err := source.Start(ctx); err != nil {
		fmt.Printf("Source error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Done.")
}
