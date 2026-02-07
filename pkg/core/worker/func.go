package worker

import (
	"context"
	"errors"
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

	// Result
	err error
}

func (w *funcWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.status != StatusCreated && w.status != StatusPending {
		w.mu.Unlock()
		return fmt.Errorf("worker %s already started (status: %s)", w.String(), w.status)
	}

	// Create context for the function
	fnCtx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.status = StatusRunning
	w.mu.Unlock()
	w.emitStateChange(State{Name: w.String(), Status: StatusCreated}, State{Name: w.String(), Status: StatusRunning})

	// Monitor in background
	go func() {
		start := time.Now()
		metrics.GetProvider().IncWorkerStarted("func")
		log.Info("func worker started", "name", w.String())

		err := w.fn(fnCtx)
		duration := time.Since(start)

		w.mu.Lock()
		oldStatus := w.status
		w.status = StatusStopped
		if err != nil {
			// Check if cancelled (normal stop)
			if errors.Is(err, context.Canceled) {
				// Treat context.Canceled as a clean stop
				w.status = StatusStopped
				w.err = nil
				metrics.GetProvider().IncWorkerStopped("func")
				log.Info("func worker stopped (canceled)", "name", w.String(), "duration", duration)
			} else {
				w.status = StatusFailed
				w.err = err
				metrics.GetProvider().IncWorkerFailed("func")
				log.Error("func worker failed", "name", w.String(), "error", err, "duration", duration)
			}
		} else {
			metrics.GetProvider().IncWorkerStopped("func")
			log.Info("func worker stopped", "name", w.String(), "duration", duration)
		}
		newStatus := w.status
		w.mu.Unlock()

		w.emitStateChange(State{Name: w.String(), Status: oldStatus}, State{Name: w.String(), Status: newStatus})
		metrics.GetProvider().ObserveWorkerDuration("func", duration)

		w.done <- err
		close(w.done)
	}()

	return nil
}

func (w *funcWorker) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cancel != nil {
		w.cancel()
		log.Debug("signaled func worker to stop", "name", w.String())
	}
	return nil
}

func (w *funcWorker) State() State {
	return w.ExportState(func(s *State) {
		s.Error = w.err
		s.Metadata = map[string]string{"type": "func"}
	})
}
