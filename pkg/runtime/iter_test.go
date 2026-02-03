package runtime_test

import (
	"context"
	"testing"
	"time"

	"github.com/aretw0/lifecycle"
)

func TestReceive_ConsumesUntilClose(t *testing.T) {
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)

	ctx := context.Background()
	var got []int
	for v := range lifecycle.Receive(ctx, ch) {
		got = append(got, v)
	}

	if len(got) != 3 {
		t.Errorf("expected 3 items, got %d", len(got))
	}
}

func TestReceive_StopsOnContextCancel(t *testing.T) {
	ch := make(chan int)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		ch <- 1
		// Ensure the first item is processed so we are waiting for the second
		time.Sleep(10 * time.Millisecond)
		cancel()
		// Keep channel open to prove we stop due to context
		time.Sleep(100 * time.Millisecond)
		// Try to send (should not be received if we stopped iterating)
		select {
		case ch <- 2:
		default:
		}
	}()

	count := 0
	for range lifecycle.Receive(ctx, ch) {
		count++
		if count == 1 {
			// Wait a bit to ensure the goroutine invokes cancel()
			time.Sleep(50 * time.Millisecond)
		}
	}

	if count != 1 {
		t.Errorf("expected 1 item before cancel, got %d", count)
	}
}
