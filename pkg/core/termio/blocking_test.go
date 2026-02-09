package termio

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestSpike_BlockingPipe_SaveShip verifies the "Data First, Error Second" strategy.
// We expect that if the context is cancelled while we are blocked in a read,
// and then the read *completes* (because data arrived), we return the data
// and a nil error. The SUBSEQUENT read must then return the interruption error.
func TestSpike_BlockingPipe_SaveShip(t *testing.T) {
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
		// In v1.4, we return data first.
		if res.err != nil {
			t.Errorf("Expected nil error for first read with data, got %v", res.err)
		}
		if res.n != 10 {
			t.Errorf("Expected 10 bytes, got %d", res.n)
		}

		// 7. Verify subsequent read returns error
		n, err := reader.Read(make([]byte, 10))
		if !IsInterrupted(err) {
			t.Errorf("Expected interruption error on second read, got %v", err)
		}
		if n != 0 {
			t.Errorf("Expected 0 bytes on second read, got %d", n)
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

// TestSpike_Interactive_StrictCancel verifies that ReadInteractive DISCARDS data
// if the context is cancelled, prioritizing the error.
func TestSpike_Interactive_StrictCancel(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())
	reader := NewInterruptibleReader(r, ctx.Done())

	type result struct {
		n   int
		err error
	}
	resChan := make(chan result, 1)

	// Launch Reader
	go func() {
		buf := make([]byte, 1024)
		// USE STRICT READ
		n, err := reader.ReadInteractive(buf)
		resChan <- result{n: n, err: err}
	}()

	time.Sleep(50 * time.Millisecond)

	// Cancel Context ("Stop!")
	cancel()

	// But also simulate data arriving ("y\n") at the same time
	w.Write([]byte("yes"))

	select {
	case res := <-resChan:
		// STRICT MODE: Should return Error, NOT data.
		if !IsInterrupted(res.err) {
			t.Errorf("Expected interruption error, got %v", res.err)
		}
		if res.n != 0 {
			// This is the key difference: Data is discarded!
			t.Errorf("Expected 0 bytes (discarded), got %d: %q", res.n, string(make([]byte, res.n)))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Test timed out")
	}
}
