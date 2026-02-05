package shared

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aretw0/lifecycle"
)

// RunFactory executes the standard interactive factory logic for suspend examples.
func RunFactory(sup lifecycle.Supervisor, store *Store, suspendHandler *lifecycle.SuspendHandler) {
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	err := lifecycle.Run(lifecycle.Job(func(ctx context.Context) error {
		resumedCh := make(chan struct{})
		quitCh := make(chan struct{})

		router := lifecycle.NewInteractiveRouter(suspendHandler,
			lifecycle.WithShutdown(func() {
				lifecycle.Shutdown(ctx)
				close(quitCh)
			}),
		)

		simSource := make(chan lifecycle.Event)
		router.AddSource(lifecycle.NewChannelSource(simSource))

		if suspendable, ok := sup.(lifecycle.Suspendable); ok {
			suspendHandler.Manage(suspendable)
		}

		suspendHandler.OnSuspend(func(ctx context.Context) error {
			if err := store.Save(ctx); err != nil {
				return err
			}
			fmt.Println("\n🛑 SYSTEM SUSPENDED.")
			fmt.Println("👉 Commands: [r]esume | [q]uit | [x] terminate")
			return nil
		})

		suspendHandler.OnResume(func(ctx context.Context) error {
			fmt.Println("\n🟢 SYSTEM RESUMED.")
			fmt.Println("👉 Manual Mode Active. Commands: [s]uspend | [q]uit | [x] terminate")
			select {
			case resumedCh <- struct{}{}:
			default:
			}
			return nil
		})

		lifecycle.Go(ctx, func(ctx context.Context) error {
			return router.Start(ctx)
		})
		if err := sup.Start(ctx); err != nil {
			return err
		}
		defer func() {
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer stopCancel()
			sup.Stop(stopCtx)
		}()

		slog.Info(">>> FACTORY RUNNING <<<")
		fmt.Println("\n🚀 FACTORY IS LIVE!")
		fmt.Println("👉 Commands: [s]uspend | [r]esume | [q]uit | [x] terminate (save & quit)")
		fmt.Println("👉 Auto-Suspend in 10s (unless you interact)")

		autoSuspend := time.NewTimer(10 * time.Second)
		defer autoSuspend.Stop()
		autoSuspendActive := true

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case <-quitCh:
				fmt.Println("👋 Quitting via command...")
				return nil

			case <-autoSuspend.C:
				if !autoSuspendActive {
					continue
				}
				store.Mu.Lock()
				done := store.State.ItemsProcessed >= TargetGoal
				store.Mu.Unlock()
				if !done {
					slog.Warn(">>> AUTO-SUSPEND TRIGGERED <<<")
					simSource <- lifecycle.SuspendEvent{}
				}

			case <-resumedCh:
				autoSuspendActive = false
			}

			// Immediate health check for the goal
			store.Mu.Lock()
			count := store.State.ItemsProcessed
			store.Mu.Unlock()
			if count >= TargetGoal {
				slog.Info("🏆 GOAL REACHED! Shutting down factory.")
				store.Cleanup()
				lifecycle.Shutdown(ctx)
				return nil
			}
		}
	}),
		lifecycle.WithLogger(l),
		lifecycle.WithForceExit(2),
		lifecycle.WithCancelOnInterrupt(false), // Interactive mode: Ctrl+C suspends, doesn't cancel
	)

	if err != nil && err != context.Canceled {
		fmt.Printf("Factory exited with error: %v\n", err)
	}
}
