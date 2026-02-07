package events_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/worker"
	"github.com/aretw0/lifecycle/pkg/events"
)

// strictWorker increments a counter continuously unless suspended.
type strictWorker struct {
	count atomic.Int64
	gate  *worker.SuspendGate
	wait  chan error
}

func newStrictWorker() *strictWorker {
	return &strictWorker{
		gate: worker.NewSuspendGate(),
		wait: make(chan error),
	}
}

func (w *strictWorker) Start(ctx context.Context) error {
	go func() {
		defer close(w.wait)
		for {
			// Check for suspension before each unit of work
			if err := w.gate.Check(ctx); err != nil {
				return
			}

			// Do work
			w.count.Add(1)
			time.Sleep(10 * time.Microsecond)

			// Simple exit if context done
			if ctx.Err() != nil {
				return
			}
		}
	}()
	return nil
}

func (w *strictWorker) Stop(ctx context.Context) error { return nil }
func (w *strictWorker) Wait() <-chan error             { return w.wait }
func (w *strictWorker) String() string                 { return "strictWorker" }
func (w *strictWorker) State() worker.State            { return worker.State{} }

// Implement Suspendable using the Gate helper
func (w *strictWorker) Suspend(ctx context.Context) error { return w.gate.Suspend(ctx) }
func (w *strictWorker) Resume(ctx context.Context) error  { w.gate.Resume(); return nil }

func TestSuspendQuiescence(t *testing.T) {
	handler := events.NewSuspendHandler()
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
	err := handler.HandleEvent(ctx, events.SuspendEvent{})
	if err != nil {
		t.Fatalf("Suspend failed: %v", err)
	}

	// Capture count IMMEDIATELY after suspension returns
	countAtSuspend := w.count.Load()

	// Wait long enough for many potential increments if it didn't really stop
	time.Sleep(200 * time.Millisecond)

	finalCount := w.count.Load()
	if finalCount != countAtSuspend {
		t.Errorf("Worker continued working after Suspend returned! Count at suspend: %d, Count after wait: %d (Delta: %d)",
			countAtSuspend, finalCount, finalCount-countAtSuspend)
	}

	// RESUME
	err = handler.HandleEvent(ctx, events.ResumeEvent{})
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	// Verify work resumes
	time.Sleep(50 * time.Millisecond)
	if w.count.Load() <= finalCount {
		t.Errorf("Worker did not resume working. Count at resume: %d, Current count: %d", finalCount, w.count.Load())
	}
}

func TestSuspend_EdgeCases(t *testing.T) {
	handler := events.NewSuspendHandler()
	ctx := context.Background()

	// 1. Double Resume (should be no-op)
	if err := handler.HandleEvent(ctx, events.ResumeEvent{}); err != nil {
		t.Errorf("Double resume should be no-op, got error: %v", err)
	}

	// 2. Double Suspend (should be no-op)
	if err := handler.HandleEvent(ctx, events.SuspendEvent{}); err != nil {
		t.Fatalf("First suspend failed: %v", err)
	}

	state := handler.State().(map[string]any)
	if !state["suspended"].(bool) {
		t.Error("Should be suspended")
	}

	if err := handler.HandleEvent(ctx, events.SuspendEvent{}); err != nil {
		t.Errorf("Double suspend should be no-op, got error: %v", err)
	}

	// 3. Unknown Event (should be no-op)
	if err := handler.HandleEvent(ctx, events.StatusEvent{}); err != nil {
		t.Errorf("Unknown event should be ignored, got error: %v", err)
	}

	// 4. Hook Failure
	failHandler := events.NewSuspendHandler()
	failHandler.OnResume(func(ctx context.Context) error {
		return context.DeadlineExceeded
	})

	// Force suspended state to test resume failure
	// We use public API to suspend first
	failHandler.HandleEvent(ctx, events.SuspendEvent{})

	if err := failHandler.HandleEvent(ctx, events.ResumeEvent{}); err == nil {
		t.Error("Expected error from failing hook, got nil")
	}
}
