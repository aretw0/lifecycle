package events

import (
	"context"
	"io"
	"os"
	"testing"
	"time"
)

func TestInputSource(t *testing.T) {
	r, w, _ := os.Pipe()

	source := NewInputSource(WithInputReader(r))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := source.Events()

	go func() {
		_ = source.Start(ctx)
	}()

	// Test mapping: "suspend"
	_, _ = w.Write([]byte("suspend\n"))

	select {
	case ev := <-ch:
		if _, ok := ev.(SuspendEvent); !ok {
			t.Errorf("Expected SuspendEvent, got %T (%s)", ev, ev.String())
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for suspend event")
	}

	// Test unknown command
	unknownCalled := make(chan string, 1)
	source.unknownHandler = func(cmd string, known []string) {
		unknownCalled <- cmd
	}

	_, _ = w.Write([]byte("unknown\n"))
	select {
	case cmd := <-unknownCalled:
		if cmd != "unknown" {
			t.Errorf("Expected unknown command 'unknown', got '%s'", cmd)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for unknown handler call")
	}
}

// MockReader implements io.Reader to simulate errors and partial reads
type MockReader struct {
	chunks [][]byte
	err    error
	delay  time.Duration
	index  int
}

func (m *MockReader) Read(p []byte) (n int, err error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.index >= len(m.chunks) {
		if m.err != nil {
			return 0, m.err
		}
		// Default to blocking if no more chunks and no error, or EOF
		return 0, io.EOF
	}
	chunk := m.chunks[m.index]
	m.index++
	copy(p, chunk)
	return len(chunk), nil
}

func TestInputSource_Options(t *testing.T) {
	handled := false
	handler := func(cmd string, known []string) { handled = true }

	src := NewInputSource(
		WithInputBackoff(10*time.Millisecond),
		WithUnknownHandler(handler),
		WithInputMapping("custom", SuspendEvent{}),
	)

	if src.backoff != 10*time.Millisecond {
		t.Error("WithInputBackoff failed")
	}
	if _, ok := src.mappings["custom"]; !ok {
		t.Error("WithInputMapping failed")
	}

	// Trigger unknown handler
	src.processCommand(context.Background(), "unknown")
	if !handled {
		t.Error("WithUnknownHandler failed")
	}
}

func TestInputSource_PartialReads(t *testing.T) {
	reader := &MockReader{
		chunks: [][]byte{
			[]byte("su"),
			[]byte("spe"),
			[]byte("nd\n"),
		},
	}

	src := NewInputSource(WithInputReader(reader))
	ch := src.Events()

	go src.readLoop(context.Background())

	select {
	case ev := <-ch:
		if _, ok := ev.(SuspendEvent); !ok {
			t.Errorf("Expected SuspendEvent from partial reads, got %T", ev)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for event from partial reads")
	}
}

func TestInputSource_EOF(t *testing.T) {
	// Simulate EOF threshold
	reader := &MockReader{
		chunks: [][]byte{}, // Immediate EOF
		err:    io.EOF,
	}

	// Use very short backoff for speed
	src := NewInputSource(
		WithInputReader(reader),
		WithInputBackoff(1*time.Millisecond),
	)

	// Start in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		src.readLoop(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Logic says it retries 3 times then stops.
		// If it returns, success.
	case <-time.After(1 * time.Second):
		t.Error("readLoop should exit after persistent EOF")
	}
}
