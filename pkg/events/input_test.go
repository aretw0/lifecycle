package events_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/events"
)

type mockReader struct {
	data string
	ptr  int
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	if m.ptr >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.ptr:])
	m.ptr += n
	return n, nil
}

func TestInputSource_WithBackoff(t *testing.T) {
	// Create a reader that fails instantly to trigger backoff
	// But since we can't easily mock error backoff without waiting,
	// we'll rely on inspecting the struct options if we could,
	// or observing behavior.
	// Actually, just testing that it constructs and runs without crashing using the option is a good start.
	// Testing time.Sleep is hard without injection.

	src := events.NewInputSource(
		events.WithInputReader(strings.NewReader("q\n")),
		events.WithInputBackoff(10*time.Millisecond),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// It should read 'q', map to ShutdownEvent, and exit.
	go func() {
		err := src.Start(ctx)
		if err != nil {
			t.Errorf("Start failed: %v", err)
		}
	}()

	select {
	case ev := <-src.Events():
		if _, ok := ev.(events.ShutdownEvent); !ok {
			t.Errorf("Expected ShutdownEvent, got %T", ev)
		}
	case <-ctx.Done():
		t.Error("Timed out waiting for event")
	}
}

func TestInputSource_UnknownHandler(t *testing.T) {
	r := strings.NewReader("unknown_cmd\n")

	handled := false
	src := events.NewInputSource(
		events.WithInputReader(r),
		events.WithUnknownHandler(func(cmd string, known []string) {
			if cmd != "unknown_cmd" {
				t.Errorf("Expected command 'unknown_cmd', got %q", cmd)
			}
			if len(known) == 0 {
				t.Error("Expected known commands list to be populated")
			}
			handled = true
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	src.Start(ctx)

	if !handled {
		t.Error("Unknown handler was not called")
	}
}
