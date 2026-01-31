package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
)

// FromFunc creates a Worker from a simple function.
// The function is executed in a goroutine when Start is called.
// The context passed to the function is cancelled when Stop is called.
func FromFunc(name string, fn func(context.Context) error) Worker {
	return &funcWorker{
		name:   name,
		fn:     fn,
		wait:   make(chan error, 1),
		status: StatusPending,
	}
}

type funcWorker struct {
	name string
	fn   func(context.Context) error

	mu     sync.Mutex
	status Status

	cancel context.CancelFunc
	wait   chan error

	// Result
	err error
}

func (w *funcWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.status != StatusPending {
		return fmt.Errorf("worker %s already started", w.name)
	}

	// Create context for the function
	fnCtx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.status = StatusRunning

	// Monitor in background
	go func() {
		start := time.Now()
		metrics.GetProvider().IncWorkerStarted("func")
		log.Info("func worker started", "name", w.name)

		err := w.fn(fnCtx)
		duration := time.Since(start)

		w.mu.Lock()
		w.status = StatusStopped
		if err != nil {
			// Check if cancelled (normal stop)
			if errors.Is(err, context.Canceled) {
				// Treat context.Canceled as a clean stop
				w.status = StatusStopped
				w.err = nil
				metrics.GetProvider().IncWorkerStopped("func")
				log.Info("func worker stopped (canceled)", "name", w.name, "duration", duration)
			} else {
				w.status = StatusFailed
				w.err = err
				metrics.GetProvider().IncWorkerFailed("func")
				log.Error("func worker failed", "name", w.name, "error", err, "duration", duration)
			}
		} else {
			metrics.GetProvider().IncWorkerStopped("func")
			log.Info("func worker stopped", "name", w.name, "duration", duration)
		}
		w.mu.Unlock()

		metrics.GetProvider().ObserveWorkerDuration("func", duration)

		w.wait <- err
		close(w.wait)
	}()

	return nil
}

func (w *funcWorker) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cancel != nil {
		w.cancel()
		log.Debug("signaled func worker to stop", "name", w.name)
	}
	return nil
}

func (w *funcWorker) Wait() <-chan error {
	return w.wait
}

func (w *funcWorker) String() string { return w.name }

func (w *funcWorker) State() State {
	w.mu.Lock()
	defer w.mu.Unlock()
	return State{
		Name:     w.name,
		Status:   w.status,
		Error:    w.err,
		Metadata: map[string]string{"type": "func"},
	}
}
