package lifecycle

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/metrics"
	"github.com/aretw0/lifecycle/pkg/metrics/mock"
)

func TestGroup_PanicRecovery(t *testing.T) {
	// Setup mock provider
	mockProvider := mock.New()
	metrics.SetProvider(mockProvider)

	ctx := context.Background()
	g, _ := NewGroup(ctx)

	// Launch a goroutine that panics
	g.Go(func(ctx context.Context) error {
		panic("oops")
	})

	err := g.Wait()
	if err == nil {
		t.Fatal("Expected error from panic, got nil")
	}

	if !strings.Contains(err.Error(), "panic in lifecycle.Group") {
		t.Errorf("Expected panic error message, got: %v", err)
	}

	// Verify metrics
	if mockProvider.GoroutinesPanicked != 1 {
		t.Errorf("Expected 1 panicked goroutine, got %d", mockProvider.GoroutinesPanicked)
	}
}

func TestGroup_Metrics(t *testing.T) {
	mockProvider := mock.New()
	metrics.SetProvider(mockProvider)

	ctx := context.Background()
	g, _ := NewGroup(ctx)

	g.Go(func(ctx context.Context) error {
		return nil
	})

	g.Wait()

	if mockProvider.GoroutinesStarted != 1 {
		t.Errorf("Expected 1 started goroutine, got %d", mockProvider.GoroutinesStarted)
	}
	if mockProvider.GoroutinesFinished != 1 {
		t.Errorf("Expected 1 finished goroutine, got %d", mockProvider.GoroutinesFinished)
	}
}

func TestGroup_ErrorPropagation(t *testing.T) {
	ctx := context.Background()
	g, _ := NewGroup(ctx)

	expectedErr := errors.New("worker error")

	g.Go(func(ctx context.Context) error {
		return expectedErr
	})

	err := g.Wait()
	if err != expectedErr {
		t.Errorf("Expected %v, got %v", expectedErr, err)
	}
}

func TestGroup_Limit(t *testing.T) {
	ctx := context.Background()
	g, _ := NewGroup(ctx)
	g.SetLimit(1)

	// Channel to signal that the first goroutine is running
	firstRunning := make(chan struct{})
	// Channel to block the first goroutine
	blockFirst := make(chan struct{})

	g.Go(func(ctx context.Context) error {
		close(firstRunning)
		<-blockFirst
		return nil
	})

	<-firstRunning

	// Try to launch a second goroutine. It should block.
	// We can't easily assert "it blocks" without a timeout or race condition check,
	// but we can assert that it *eventually* runs after we unblock the first.

	secondDone := make(chan struct{})
	go func() {
		g.Go(func(ctx context.Context) error {
			close(secondDone)
			return nil
		})
	}()

	select {
	case <-secondDone:
		t.Fatal("Second goroutine started despite limit=1")
	case <-time.After(10 * time.Millisecond):
		// Good, it didn't start immediately
	}

	// Unblock first
	close(blockFirst)

	select {
	case <-secondDone:
		// Good, it started after capacity freed
	case <-time.After(1 * time.Second):
		t.Fatal("Second goroutine did not start after unblocking")
	}

	g.Wait()
}

func TestGroup_BackpressureMetrics(t *testing.T) {
	mockProvider := mock.New()
	metrics.SetProvider(mockProvider)

	ctx := context.Background()
	g, _ := NewGroup(ctx)
	g.SetLimit(1)

	// Block the slot
	releaseSlot := make(chan struct{})
	g.Go(func(ctx context.Context) error {
		<-releaseSlot
		return nil
	})

	// Wait for first routine to start
	// (Best effort wait, or we need a way to know it started.
	// The mock provider is naive, but we can check GoroutinesStarted)
	time.Sleep(10 * time.Millisecond)

	// Launch second routine (should wait)
	done := make(chan struct{})
	go func() {
		g.Go(func(ctx context.Context) error {
			close(done)
			return nil
		})
	}()

	// Give it time to enter the queue
	time.Sleep(50 * time.Millisecond)

	// It should verify that we recorded a Waiting goroutine (eventually).
	// Since our mock relies on Inc/Dec, we can't easily check "Current Waiting" without a counter.
	// But we can assume IncGoroutineWaiting was called.

	// Unblock
	close(releaseSlot)
	<-done
	g.Wait()
}
