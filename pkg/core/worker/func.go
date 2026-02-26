package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
)

// FromFunc creates a Worker from a simple function.
func FromFunc(name string, fn func(context.Context) error) Worker {
	return &funcWorker{
		BaseWorker: NewBaseWorker(name),
		fn:         fn,
	}
}

type funcWorker struct {
	*BaseWorker
	fn func(context.Context) error

	cancel context.CancelFunc
}

func (w *funcWorker) Start(ctx context.Context) error {
	var fnCtx context.Context
	var cancel context.CancelFunc
	err := withLockResult(w.BaseWorker, func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if w.status != StatusCreated && w.status != StatusPending {
			return fmt.Errorf("worker %s already started (status: %s)", w.String(), w.status)
		}
		fnCtx, cancel = context.WithCancel(context.Background())
		w.cancel = cancel
		return nil
	})
	if err != nil {
		return err
	}

	w.SetStatus(StatusRunning)

	// Monitor in background
	go func() {
		start := time.Now()
		metrics.GetProvider().IncWorkerStarted("func")
		log.Info("func worker started", "name", w.String())

		err := w.fn(fnCtx)
		duration := time.Since(start)

		w.Finish(err)
		metrics.GetProvider().ObserveWorkerDuration("func", duration)
	}()

	return nil
}

func (w *funcWorker) Stop(ctx context.Context) error {
	withLock(&w.BaseWorker.mu, func() {
		if w.cancel != nil {
			w.StopRequested = true
			w.cancel()
			log.Debug("signaled func worker to stop", "name", w.String())
		}
	})

	// Wait for quiescence usando BaseWorker
	return w.BaseWorker.Stop(ctx)
}

func (w *funcWorker) State() State {
	return w.ExportState(func(s *State) {
		s.Error = w.Err
		s.Metadata = map[string]string{"type": "func"}
	})
}
