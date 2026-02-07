package metrics

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/log"
)

func TestNoOpProvider(t *testing.T) {
	p := &NoOpProvider{}
	// Safety check: Should not panic on representative calls
	p.IncSignalReceived("INT")
	p.IncCriticalSectionStarted()
	p.IncCriticalSectionFinished(true)
	p.IncProcessFailed()
	p.IncTerminalUpgrade(true)
}

func TestLogProvider(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}
	lp.IncSignalReceived("SIGINT")

	if !bytes.Contains(buf.Bytes(), []byte("metric incremented")) {
		t.Errorf("Expected log to contain 'metric incremented', got: %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_signals_total")) {
		t.Errorf("Expected log to contain 'lifecycle_signals_total', got: %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("SIGINT")) {
		t.Errorf("Expected log to contain 'SIGINT', got: %s", buf.String())
	}
}

func TestGlobalProvider(t *testing.T) {
	defaultP := GetProvider()
	if _, ok := defaultP.(*NoOpProvider); !ok {
		t.Errorf("Expected default provider to be NoOpProvider, got %T", defaultP)
	}

	mock := &NoOpProvider{}
	SetProvider(mock)
	if GetProvider() != mock {
		t.Error("SetProvider failed to update global provider")
	}

	SetProvider(nil)
	if _, ok := GetProvider().(*NoOpProvider); !ok {
		t.Error("SetProvider(nil) should restore NoOpProvider")
	}
}

func TestLogProvider_Behavior(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}

	tests := []struct {
		name     string
		action   func()
		contains []string
	}{
		{
			name:     "Signal",
			action:   func() { lp.IncSignalReceived("SIGINT") },
			contains: []string{"metric incremented", "lifecycle_signals_total", "SIGINT"},
		},
		{
			name:     "ProcessStarted",
			action:   func() { lp.IncProcessStarted() },
			contains: []string{"metric incremented", "lifecycle_processes_started_total"},
		},
		{
			name:     "HookDuration",
			action:   func() { lp.ObserveHookDuration(100 * time.Millisecond) },
			contains: []string{"metric observed", "lifecycle_hook_duration_seconds", "0.1"},
		},
		{
			name:     "CriticalSection",
			action:   func() { lp.IncCriticalSectionStarted() },
			contains: []string{"metric incremented", "lifecycle_critical_section_started_total"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.action()
			output := buf.String()
			for _, c := range tt.contains {
				if !bytes.Contains(buf.Bytes(), []byte(c)) {
					t.Errorf("Expected log to contain %q, got: %s", c, output)
				}
			}
		})
	}
}

// Removed TestLogProvider repetitive methods and TestNoOpProvider_All_Methods theater
