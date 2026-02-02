package sources

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTickerSource(t *testing.T) {
	interval := 10 * time.Millisecond
	source := NewTickerSource(interval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start source in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		source.Start(ctx)
	}()

	// Listen for events
	events := source.Events()

	select {
	case e := <-events:
		if e.String() == "" {
			t.Error("received empty event string")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for tick event")
	}

	// Verify cancellation
	cancel()

	// Wait for Start to exit
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success: Start() returned
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for source to stop")
	}

	// Double check channel is closed
	select {
	case _, ok := <-events:
		if ok {
			// Drain potentially one queued event
			select {
			case _, ok := <-events:
				if ok {
					t.Error("expected events channel to be closed")
				}
			case <-time.After(50 * time.Millisecond):
				t.Error("timeout waiting for channel close")
			}
		}
	default:
		// Closed or empty? Read should block if open and empty, or return false if closed
		// Safest is to rely on wg.Wait() above which ensures Start() deferred close() ran.
	}
}
