package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	ctx := lifecycle.Context()
	defer lifecycle.ShutdownAndWait(ctx)

	lifecycle.Go(ctx, func(ctx context.Context) error {
		<-ctx.Done()
		fmt.Println("background task stopped")
		return nil
	})

	fmt.Println("Press Ctrl+C to stop (or wait for timeout)...")
	select {
	case <-ctx.Done():
		fmt.Println("shutdown signal received")
	case <-time.After(5 * time.Second):
		fmt.Println("timeout reached, initiating shutdown")
		lifecycle.Shutdown(ctx)
	}
}
