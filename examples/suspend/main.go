package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"log/slog"

	"github.com/aretw0/lifecycle"
	"github.com/aretw0/lifecycle/pkg/control"
	"github.com/aretw0/lifecycle/pkg/handlers"
	"github.com/aretw0/lifecycle/pkg/log"
)

func main() {
	// 1. Setup Logger
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, opts))
	log.SetLogger(logger)

	// 2. Create the Suspend Handler using the Facade API where possible,
	//    but handlers must be instantiated from their package.
	suspendHandler := handlers.NewSuspendHandler()

	// 3. Register Hooks
	suspendHandler.OnSuspend(func(ctx context.Context) error {
		log.Info("MAIN: Suspending processing... (Simulating checkpoint)")
		time.Sleep(500 * time.Millisecond) // Simulate work
		log.Info("MAIN: Checkpoint saved.")
		return nil
	})

	suspendHandler.OnResume(func(ctx context.Context) error {
		log.Info("MAIN: Resuming processing... (Simulating restore)")
		return nil
	})

	// 4. Register Routes on the Default Router
	lifecycle.Handle("lifecycle/suspend", suspendHandler)
	lifecycle.Handle("lifecycle/resume", suspendHandler)

	// 5. Run the Application
	lifecycle.Run(lifecycle.Job(func(ctx context.Context) error {
		log.Info("App started. Press Ctrl+C to exit.")

		// Simulate external events triggering Suspend/Resume
		lifecycle.Go(ctx, func(ctx context.Context) error {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			suspended := false

			for {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
					if suspended {
						// Trigger Resume
						fmt.Println("\n--- Triggering RESUME Event ---")
						// Note: In real apps, this comes from a Source (Webhook, etc).
						// Here we inject it manually into the DefaultRouter for the demo.
						control.DefaultRouter.Dispatch(ctx, control.ResumeEvent{})
						suspended = false
					} else {
						// Trigger Suspend
						fmt.Println("\n--- Triggering SUSPEND Event ---")
						control.DefaultRouter.Dispatch(ctx, control.SuspendEvent{})
						suspended = true
					}

					// Demonstrate Introspection
					state := suspendHandler.State()
					log.Debug("Handler State", "state", state)
				}
			}
		})

		return nil
	}))
}
