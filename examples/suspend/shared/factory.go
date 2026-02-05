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
		suspendedCh := make(chan struct{})
		resumedCh := make(chan struct{})
		quitCh := make(chan struct{})

		router := lifecycle.NewInteractiveRouter(suspendHandler,
			lifecycle.WithShutdown(func() {
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
			select {
			case suspendedCh <- struct{}{}:
			default:
			}
			return nil
		})

		suspendHandler.OnResume(func(ctx context.Context) error {
			select {
			case resumedCh <- struct{}{}:
			default:
			}
			return nil
		})

		lifecycle.Go(ctx, router.Start)
		if err := sup.Start(ctx); err != nil {
			return err
		}
		defer sup.Stop(context.WithoutCancel(ctx))

		slog.Info(">>> FACTORY RUNNING <<<")
		slog.Info("Commands: [s]uspend, [r]esume, [q]uit")
		slog.Info("Auto-Suspend in 10s (unless you interact)")

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

			case <-suspendedCh:
				autoSuspend.Stop()
				if autoSuspendActive {
					fmt.Println("\n🛑 SYSTEM SUSPENDED (Auto).")
				} else {
					fmt.Println("\n🛑 SYSTEM SUSPENDED (Manual).")
				}
				fmt.Println("👉 Commands: [r]esume | [q]uit")

			case <-resumedCh:
				autoSuspendActive = false
				fmt.Println("\n🟢 SYSTEM RESUMED.")
				fmt.Println("👉 Manual Mode Active. Commands: [s]uspend | [q]uit")

			case <-time.After(1 * time.Second):
				store.Mu.Lock()
				count := store.State.ItemsProcessed
				store.Mu.Unlock()
				if count >= TargetGoal {
					slog.Info("🏆 GOAL REACHED! Shutting down factory.")
					store.Cleanup()
					return nil
				}
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
