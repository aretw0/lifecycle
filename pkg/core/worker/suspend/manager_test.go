package suspend

import (
	"context"
	"testing"
	"time"
)

func TestManager_InitialState(t *testing.T) {
	mgr := NewManager()

	if mgr.IsPaused() {
		t.Error("New manager should be in running state")
	}
}

func TestManager_PauseResume(t *testing.T) {
	mgr := NewManager()

	// Pause
	mgr.Pause()
	if !mgr.IsPaused() {
		t.Error("Manager should be paused")
	}

	// Resume
	mgr.Resume()
	if mgr.IsPaused() {
		t.Error("Manager should be running")
	}
}

func TestManager_IdempotentPause(t *testing.T) {
	mgr := NewManager()

	// Multiple pauses should be safe
	mgr.Pause()
	mgr.Pause()
	mgr.Pause()

	if !mgr.IsPaused() {
		t.Error("Manager should still be paused")
	}
}

func TestManager_IdempotentResume(t *testing.T) {
	mgr := NewManager()

	mgr.Pause()

	// Multiple resumes should be safe
	mgr.Resume()
	mgr.Resume()
	mgr.Resume()

	if mgr.IsPaused() {
		t.Error("Manager should be running")
	}
}

func TestManager_WaitWhenRunning(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	// Should return immediately when not paused
	err := mgr.Wait(ctx)
	if err != nil {
		t.Errorf("Wait should return nil when running, got: %v", err)
	}
}

func TestManager_WaitWhenPausedThenResumed(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	mgr.Pause()

	// Start resume in background
	go func() {
		time.Sleep(100 * time.Millisecond)
		mgr.Resume()
	}()

	// Wait should block until resumed
	start := time.Now()
	err := mgr.Wait(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Wait returned error: %v", err)
	}

	if duration < 100*time.Millisecond {
		t.Errorf("Wait should have blocked until resume, took %v", duration)
	}
}

func TestManager_WaitContextCancellation(t *testing.T) {
	mgr := NewManager()
	mgr.Pause()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// Wait should return context error
	err := mgr.Wait(ctx)
	if err != context.Canceled {
		t.Errorf("Wait should return context.Canceled, got: %v", err)
	}
}

func TestManager_WaitContextTimeout(t *testing.T) {
	mgr := NewManager()
	mgr.Pause()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Wait should timeout
	start := time.Now()
	err := mgr.Wait(ctx)
	duration := time.Since(start)

	if err != context.DeadlineExceeded {
		t.Errorf("Wait should return DeadlineExceeded, got: %v", err)
	}

	if duration < 100*time.Millisecond || duration > 200*time.Millisecond {
		t.Errorf("Wait should timeout around 100ms, took %v", duration)
	}
}

func TestManager_ConcurrentPauseResume(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	done := make(chan bool)

	// Worker goroutine
	go func() {
		for i := 0; i < 10; i++ {
			if err := mgr.Wait(ctx); err != nil {
				t.Errorf("Worker Wait error: %v", err)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Controller goroutine
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(20 * time.Millisecond)
			mgr.Pause()
			time.Sleep(20 * time.Millisecond)
			mgr.Resume()
		}
	}()

	// Should complete without hanging
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Test hung - possible deadlock")
	}
}

func TestManager_MultipleWaiters(t *testing.T) {
	mgr := NewManager()
	mgr.Pause()

	const numWaiters = 5
	resumed := make(chan bool, numWaiters)

	ctx := context.Background()

	// Start multiple waiters
	for i := 0; i < numWaiters; i++ {
		go func() {
			if err := mgr.Wait(ctx); err != nil {
				t.Errorf("Waiter error: %v", err)
				return
			}
			resumed <- true
		}()
	}

	// Resume should wake all waiters
	time.Sleep(50 * time.Millisecond)
	mgr.Resume()

	// All waiters should resume
	timeout := time.After(1 * time.Second)
	for i := 0; i < numWaiters; i++ {
		select {
		case <-resumed:
			// Good
		case <-timeout:
			t.Fatalf("Only %d/%d waiters resumed", i, numWaiters)
		}
	}
}



