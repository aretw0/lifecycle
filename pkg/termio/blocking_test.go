package termio

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestSpike_BlockingPipe_AbandonShip verifies the "Peek & Abandon" strategy behavior.
// We expect that if the context is cancelled while we are blocked in a read,
// and then the read *completes* (because data arrived), we still return the Interruption error,
// effectively abandoning the data we just read from the OS buffer.
func TestSpike_BlockingPipe_AbandonShip(t *testing.T) {
	// 1. Create a real OS pipe
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	// 2. Setup Interruptible Reader
	ctx, cancel := context.WithCancel(context.Background())
	reader := NewInterruptibleReader(r, ctx.Done())

	// Channel to capture result from the blocking read
	type result struct {
		n   int
		err error
	}
	resChan := make(chan result, 1)

	// 3. Launch Reader in Goroutine
	go func() {
		buf := make([]byte, 1024)
		n, err := reader.Read(buf)
		resChan <- result{n: n, err: err}
	}()

	// Ensure the goroutine is blocked
	time.Sleep(50 * time.Millisecond)

	// 4. Cancel Context
	// The reader is currently blocked in `r.base.Read(p)`.
	// Cancelling the context won't unblock the syscall on its own (that's the problem being solved),
	// but it sets up the condition for the "Abandon" check.
	cancel()

	// 5. Unblock the Reader
	// We write data to the pipe. The OS `read()` will now return with this data.
	_, err = w.Write([]byte("abandon me"))
	if err != nil {
		t.Fatalf("Failed to write to pipe: %v", err)
	}

	// 6. Assert Result
	select {
	case res := <-resChan:
		// We expect an error because the context was cancelled.
		if !IsInterrupted(res.err) {
			t.Errorf("Expected interruption error, got %v", res.err)
		}
		if res.n != 0 {
			// This is the critical "Data Loss" behavior.
			// Ideally we return 0 bytes to the caller so they handle the error cleanly.
			t.Errorf("Expected 0 bytes, got %d", res.n)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Test timed out waiting for read to return")
	}
}

// TestSpike_Race_ImmediateCancel verifies that if context is already cancelled,
// we don't even attempt to read, even if data is available.
func TestSpike_Race_ImmediateCancel(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	// Pre-fill data
	w.Write([]byte("available data"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	reader := NewInterruptibleReader(r, ctx.Done())

	buf := make([]byte, 10)
	n, err := reader.Read(buf)

	if !IsInterrupted(err) {
		t.Errorf("Expected interruption error, got %v", err)
	}
	if n != 0 {
		t.Errorf("Expected 0 bytes, got %d", n)
	}
}
