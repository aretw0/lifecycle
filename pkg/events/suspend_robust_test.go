package events_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/events"
)

func TestSuspendHandler_ConcurrentEvents(t *testing.T) {
	handler := events.NewSuspendHandler()
	var suspendCalls atomic.Int64
	var resumeCalls atomic.Int64

	handler.OnSuspend(func(ctx context.Context) error {
		suspendCalls.Add(1)
		time.Sleep(50 * time.Millisecond) // Simulate work
		return nil
	})

	handler.OnResume(func(ctx context.Context) error {
		resumeCalls.Add(1)
		time.Sleep(50 * time.Millisecond) // Simulate work
		return nil
	})

	ctx := context.Background()
	var wg sync.WaitGroup
	concurrentCount := 10

	// Trigger concurrent suspends
	wg.Add(concurrentCount)
	for i := 0; i < concurrentCount; i++ {
		go func() {
			defer wg.Done()
			_ = handler.HandleEvent(ctx, events.SuspendEvent{})
		}()
	}
	wg.Wait()

	if suspendCalls.Load() != 1 {
		t.Errorf("expected exactly 1 suspend hook execution, got %d", suspendCalls.Load())
	}

	// Trigger concurrent resumes
	wg.Add(concurrentCount)
	for i := 0; i < concurrentCount; i++ {
		go func() {
			defer wg.Done()
			_ = handler.HandleEvent(ctx, events.ResumeEvent{})
		}()
	}
	wg.Wait()

	if resumeCalls.Load() != 1 {
		t.Errorf("expected exactly 1 resume hook execution, got %d", resumeCalls.Load())
	}
}

func TestSuspendHandler_HookOrdering(t *testing.T) {
	handler := events.NewSuspendHandler()
	var results []string
	var mu sync.Mutex

	// Heavy worker hook (functional)
	handler.OnSuspend(func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		results = append(results, "functional")
		mu.Unlock()
		return nil
	})

	// UI hook (reporting)
	handler.OnSuspend(func(ctx context.Context) error {
		mu.Lock()
		results = append(results, "ui")
		mu.Unlock()
		return nil
	})

	_ = handler.HandleEvent(context.Background(), events.SuspendEvent{})

	mu.Lock()
	if len(results) != 2 || results[0] != "functional" || results[1] != "ui" {
		t.Errorf("unexpected hook execution order: %v", results)
	}
	mu.Unlock()
}

func TestSuspendHandler_ContextCancellation(t *testing.T) {
	handler := events.NewSuspendHandler()

	handler.OnSuspend(func(ctx context.Context) error {
		select {
		case <-time.After(1 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := handler.HandleEvent(ctx, events.SuspendEvent{})
	if err == nil || err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded error, got %v", err)
	}

	// State should NOT be suspended if the hook failed
	state := handler.State().(map[string]any)
	if state["suspended"].(bool) {
		t.Error("handler should not be in suspended state after hook failure")
	}
}

func TestSuspendHandler_RaceDetector(t *testing.T) {
	handler := events.NewSuspendHandler()
	ctx := context.Background()

	// Register a simple worker
	worker := newStrictWorker()
	handler.Manage(worker)
	_ = worker.Start(ctx)

	var wg sync.WaitGroup
	wg.Add(2)

	// One goroutine spams Suspend/Resume
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = handler.HandleEvent(ctx, events.SuspendEvent{})
			_ = handler.HandleEvent(ctx, events.ResumeEvent{})
		}
	}()

	// Another goroutine spams State inspection
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = handler.State()
		}
	}()

	wg.Wait()
}
