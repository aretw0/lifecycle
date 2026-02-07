package termio

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestInterruptibleReader_Read_Success(t *testing.T) {
	data := []byte("hello")
	base := bytes.NewReader(data)
	cancel := make(chan struct{})
	r := NewInterruptibleReader(base, cancel)

	buf := make([]byte, 5)
	n, err := r.Read(buf)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected 5 bytes, got %d", n)
	}
	if string(buf) != "hello" {
		t.Errorf("Expected 'hello', got '%s'", string(buf))
	}
}

func TestInterruptibleReader_Read_PreCancelled(t *testing.T) {
	base := bytes.NewReader([]byte("hello"))
	cancel := make(chan struct{})
	close(cancel) // Cancel immediately

	r := NewInterruptibleReader(base, cancel)
	buf := make([]byte, 5)
	_, err := r.Read(buf)

	if !errors.Is(err, ErrInterrupted) {
		t.Errorf("Expected ErrInterrupted, got %v", err)
	}
}

// blockingReader blocks forever until we somehow unblock it (not possible with empty select)
// or just waits. mocking a blocking reader is hard without a pump.
// Since InterruptibleReader.Read() simply checks before/after, we can't easily test the "During" case
// unless base.Read() returns.
// So we verify simpler behavior.

type slowReader struct {
	delay time.Duration
	data  string
}

func (s *slowReader) Read(p []byte) (int, error) {
	time.Sleep(s.delay)
	return copy(p, s.data), nil
}

func TestInterruptibleReader_Read_Slow(t *testing.T) {
	// This test ensures that if the reader finishes, but we cancelled in the meantime,
	// we get the interruption error.
	base := &slowReader{delay: 100 * time.Millisecond, data: "ok"}
	cancel := make(chan struct{})
	r := NewInterruptibleReader(base, cancel)

	go func() {
		time.Sleep(50 * time.Millisecond)
		close(cancel) // Cancel while slowReader is sleeping
	}()

	buf := make([]byte, 2)
	n, err := r.Read(buf)

	// In v1.4, we return data first if available.
	if err != nil {
		t.Errorf("Expected nil error for first read with data, got %v", err)
	}
	if n != 2 || string(buf) != "ok" {
		t.Errorf("Expected 2 bytes 'ok', got %d bytes '%s'", n, string(buf))
	}

	// SUBSEQUENT read should return ErrInterrupted
	n, err = r.Read(buf)
	if !errors.Is(err, ErrInterrupted) {
		t.Errorf("Expected ErrInterrupted on second read, got %v", err)
	}
}

func TestInterruptibleReader_ReadInteractive_Success(t *testing.T) {
	base := bytes.NewReader([]byte("yes"))
	cancel := make(chan struct{})
	r := NewInterruptibleReader(base, cancel)

	buf := make([]byte, 10)
	n, err := r.ReadInteractive(buf)
	if err != nil {
		t.Fatalf("ReadInteractive success failed: %v", err)
	}
	if n != 3 || string(buf[:n]) != "yes" {
		t.Errorf("ReadInteractive yielded %q, want 'yes'", string(buf[:n]))
	}
}

func TestInterruptibleReader_ReadInteractive_Discard(t *testing.T) {
	// This test simulates the "Strict Discard" policy.
	// If data is read BUT the context is cancelled, it should be discarded.

	// We use the slow reader but with a delay shorter than the test timeline,
	// but we cancel *during* the read availability.
	// Actually, strict discard checks cancel *after* base.Read returns.

	// base returns "data" after 50ms
	base := &slowReader{delay: 50 * time.Millisecond, data: "unsafe input"}
	cancel := make(chan struct{})
	r := NewInterruptibleReader(base, cancel)

	// We trigger cancel at 60ms (after read finished, but before we check?)
	// No, the race is tight.
	// Logic:
	// 1. n, err = r.Read(p) [sleeps 50ms, returns "unsafe input"]
	// 2. select check cancel

	// To ensure we hit the "Strict Check", we must have cancel closed
	// exactly when checking.
	// Let's just pre-close verify.

	close(cancel) // Cancelled BEFORE we even start

	buf := make([]byte, 20)
	n, err := r.ReadInteractive(buf)

	// Even though base has data instantly (or slow), Read() checks Pre-Cancel first.
	// Wait, Read() has a pre-check too.
	// If Read() pre-check hits, it returns ErrInterrupted.
	// ReadInteractive calls Read().

	if !errors.Is(err, ErrInterrupted) {
		t.Errorf("Expected ErrInterrupted (pre-check), got %v", err)
	}
	if n != 0 {
		t.Errorf("Expected 0 bytes, got %d", n)
	}
}

func TestInterruptibleReader_ReadInteractive_PostCheck(t *testing.T) {
	// We need to bypass Read()'s pre-check but hit ReadInteractive's post-check.
	// Accessing the struct directly avoids the race? No.
	// We can use a custom reader that CLOSES the cancel channel inside Read!
	// That simulates "Cancel happened DURING the syscall".

	cancel := make(chan struct{})

	trigger := &triggerReader{
		data:   "dangerous",
		cancel: cancel,
	}

	r := NewInterruptibleReader(trigger, cancel)

	buf := make([]byte, 20)

	// For standard Read(), "Data First" means if trigger returns data,
	// even if it cancels, we get data.
	n, err := r.Read(buf)
	if err != nil {
		t.Errorf("Standard Read should prioritize data: %v", err)
	}
	if string(buf[:n]) != "dangerous" {
		t.Errorf("Standard Read missed data")
	}

	// Reset
	cancel2 := make(chan struct{})
	trigger2 := &triggerReader{data: "dangerous", cancel: cancel2}
	r2 := NewInterruptibleReader(trigger2, cancel2)

	// For ReadInteractive(), "Error First" means data is discarded.
	n, err = r2.ReadInteractive(buf)
	if !errors.Is(err, ErrInterrupted) {
		t.Errorf("ReadInteractive should return ErrInterrupted, got %v", err)
	}
	if n != 0 {
		t.Errorf("ReadInteractive should discard data, got %d bytes", n)
	}
}

type triggerReader struct {
	data   string
	cancel chan struct{}
}

func (t *triggerReader) Read(p []byte) (int, error) {
	// Trigger the cancellation "during" the read
	select {
	case <-t.cancel:
	default:
		close(t.cancel)
	}
	return copy(p, t.data), nil
}

func TestIsInterrupted(t *testing.T) {
	// 1. ErrInterrupted
	if !IsInterrupted(ErrInterrupted) {
		t.Error("ErrInterrupted should be Interrupted")
	}
	// 2. Context Canceled
	if !IsInterrupted(context.Canceled) {
		t.Error("context.Canceled should be Interrupted")
	}
	// 3. String Match
	if !IsInterrupted(errors.New("interrupted")) {
		t.Error("'interrupted' string should be Interrupted")
	}
	// 4. EOF (New Test Case)
	if !IsInterrupted(io.EOF) {
		t.Error("io.EOF should be considered Interrupted (for shell exit)")
	}
	// 5. Normal Error
	if IsInterrupted(errors.New("other")) {
		t.Error("Random error should NOT be Interrupted")
	}
	// 6. Nil
	if IsInterrupted(nil) {
		t.Error("Nil error should NOT be Interrupted")
	}
}
