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

	source := NewInputSource(
		WithInputReader(r),
		WithDefaultMappings(),
	)

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
	_, _ = w.Write([]byte("unknown\n"))

	select {
	case ev := <-ch:
		if unknownEv, ok := ev.(UnknownCommandEvent); ok {
			if unknownEv.Command != "unknown" {
				t.Errorf("Expected UnknownCommandEvent with command 'unknown', got '%s'", unknownEv.Command)
			}
		} else {
			t.Errorf("Expected UnknownCommandEvent, got %T (%s)", ev, ev.String())
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for unknown command event")
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
	src := NewInputSource(
		WithInputBackoff(10*time.Millisecond),
		WithInputMapping("custom", SuspendEvent{}),
	)

	if src.backoff != 10*time.Millisecond {
		t.Error("WithInputBackoff failed")
	}
	if _, ok := src.mappings["custom"]; !ok {
		t.Error("WithInputMapping failed")
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

	src := NewInputSource(
		WithInputReader(reader),
		WithDefaultMappings(),
	)
	ch := src.Events()

	go src.Start(context.Background())

	select {
	case ev := <-ch:
		if _, ok := ev.(SuspendEvent); !ok {
			t.Errorf("Expected SuspendEvent from partial reads, got %T", ev)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for event from partial reads")
	}
}

func TestInputSource_EmptyLine(t *testing.T) {
	reader := &MockReader{
		chunks: [][]byte{
			[]byte("\n"),
		},
	}

	src := NewInputSource(
		WithInputReader(reader),
	)
	ch := src.Events()

	go src.Start(context.Background())

	select {
	case ev := <-ch:
		if le, ok := ev.(LineEvent); ok {
			if le.Line != "" {
				t.Errorf("Expected empty LineEvent, got %q", le.Line)
			}
		} else {
			t.Errorf("Expected LineEvent for empty line, got %T", ev)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for empty line event")
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
		src.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("readLoop should exit after persistent EOF")
	}
}

func TestInputSource_BufferOptions(t *testing.T) {
	// Test Event Buffer
	src := NewInputSource(
		WithInputEventBuffer(50),
	)
	if cap(src.Events()) != 50 {
		t.Errorf("WithInputEventBuffer failed: expected capacity 50, got %d", cap(src.Events()))
	}

	// Test Read Buffer Size
	src = NewInputSource(
		WithInputBufferSize(2048),
	)
	if src.bufSize != 2048 {
		t.Errorf("WithInputBufferSize failed: expected 2048, got %d", src.bufSize)
	}

	// Test invalid size fallback
	src = NewInputSource(
		WithInputBufferSize(-1),
	)
	if src.bufSize != 1024 {
		t.Errorf("WithInputBufferSize fallback failed: expected 1024, got %d", src.bufSize)
	}
}
