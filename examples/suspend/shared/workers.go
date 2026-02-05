package shared

import (
	"context"
	"log/slog"
	"time"

	"github.com/aretw0/lifecycle"
)

// Watchdog is a system service that runs continuously, ignoring suspension.
type Watchdog struct {
	lifecycle.BaseWorker
}

func NewWatchdog() *Watchdog {
	return &Watchdog{
		BaseWorker: lifecycle.NewBaseWorker("Watchdog"),
	}
}

func (w *Watchdog) Start(ctx context.Context) error {
	return w.StartFunc(ctx, w.Run)
}

func (w *Watchdog) Run(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			slog.Info("[WATCHDOG] System Healthy. (I never sleep!)")
		}
	}
}

// Blocker is a worker that refuses to suspend quickly, testing the USER's patience.
type Blocker struct {
	lifecycle.BaseWorker
}

func NewBlocker() *Blocker {
	return &Blocker{
		BaseWorker: lifecycle.NewBaseWorker("Blocker"),
	}
}

func (b *Blocker) Start(ctx context.Context) error {
	return b.StartFunc(ctx, func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})
}

func (b *Blocker) Resume(ctx context.Context) error { return nil }
func (b *Blocker) Suspend(ctx context.Context) error {
	slog.Warn("[BLOCKER] Suspending... (I am slow! Press Ctrl+C again 2x to Force Quit, or wait)")
	select {
	case <-time.After(5 * time.Second):
		slog.Info("[BLOCKER] Finally suspended.")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}



