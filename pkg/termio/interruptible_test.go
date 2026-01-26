package termio

import (
	"bytes"
	"context"
	"errors"
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
	_, err := r.Read(buf)

	if !errors.Is(err, ErrInterrupted) {
		// Note: The implementation checks AFTER read. So this should pass.
		t.Errorf("Expected ErrInterrupted because we cancelled during read, got %v", err)
	}
}

func TestIsInterrupted(t *testing.T) {
	if !IsInterrupted(ErrInterrupted) {
		t.Error("ErrInterrupted should be IsInterrupted")
	}
	if !IsInterrupted(context.Canceled) {
		t.Error("context.Canceled should be IsInterrupted")
	}
	if !IsInterrupted(errors.New("interrupted")) {
		t.Error("'interrupted' string error should be IsInterrupted")
	}
}
