package control_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/control"
	"github.com/aretw0/lifecycle/pkg/handlers"
	"github.com/aretw0/lifecycle/pkg/worker"
)

// strictWorker increments a counter continuously unless suspended.
type strictWorker struct {
	count   atomic.Int64
	suspend chan struct{}
	resume  chan struct{}
	paused  chan struct{}
	wait    chan error
}

func newStrictWorker() *strictWorker {
	return &strictWorker{
		suspend: make(chan struct{}),
		resume:  make(chan struct{}),
		paused:  make(chan struct{}),
		wait:    make(chan error),
	}
}

func (w *strictWorker) Start(ctx context.Context) error {
	go func() {
		defer close(w.wait)
		for {
			select {
			case <-w.suspend:
				// Signal that we are pausing
				w.paused <- struct{}{}
				// Wait for resume
				select {
				case <-w.resume:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			default:
				w.count.Add(1)
				// Small sleep to avoid eating CPU but keep loop tight
				time.Sleep(10 * time.Microsecond)
			}
		}
	}()
	return nil
}

func (w *strictWorker) Stop(ctx context.Context) error { return nil }
func (w *strictWorker) Wait() <-chan error             { return w.wait }
func (w *strictWorker) String() string                 { return "strictWorker" }
func (w *strictWorker) State() worker.State            { return worker.State{} }

func (w *strictWorker) Suspend(ctx context.Context) error {
	select {
	case w.suspend <- struct{}{}:
		// Block until the loop confirms it received the signal and is pausing
		select {
		case <-w.paused:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *strictWorker) Resume(ctx context.Context) error {
	select {
	case w.resume <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestSuspendQuiescence(t *testing.T) {
	handler := handlers.NewSuspendHandler()
	w := newStrictWorker()
	handler.Manage(w)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start worker
	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for worker to spin up
	time.Sleep(100 * time.Millisecond)
	initialCount := w.count.Load()
	if initialCount == 0 {
		t.Fatal("Worker should have started working")
	}

	// TRIGGER SUSPEND via Handler
	// HandleEvent should call w.Suspend, which blocks until w.paused is sent.
	// Therefore, when HandleEvent returns, the worker MUST be at the <-w.resume select.
	err := handler.HandleEvent(ctx, control.SuspendEvent{})
	if err != nil {
		t.Fatalf("Suspend failed: %v", err)
	}

	// Capture count IMMEDIATELY after suspension returns
	countAtSuspend := w.count.Load()

	// Wait long enough for many potential increments
	time.Sleep(200 * time.Millisecond)

	finalCount := w.count.Load()
	if finalCount != countAtSuspend {
		t.Errorf("Worker continued working after Suspend returned! Count at suspend: %d, Count after wait: %d (Delta: %d)",
			countAtSuspend, finalCount, finalCount-countAtSuspend)
	}

	// RESUME
	err = handler.HandleEvent(ctx, control.ResumeEvent{})
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	// Verify work resumes
	time.Sleep(50 * time.Millisecond)
	if w.count.Load() <= finalCount {
		t.Errorf("Worker did not resume working. Count at resume: %d, Current count: %d", finalCount, w.count.Load())
	}
}
