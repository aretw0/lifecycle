package metrics

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/aretw0/lifecycle/pkg/core/log"
)

func TestNoOpProvider(t *testing.T) {
	p := &NoOpProvider{}
	// Should not panic or do anything
	p.IncSignalReceived("INT")
	p.IncProcessStarted()
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



