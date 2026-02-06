package metrics_test

import (
	"testing"

	"github.com/aretw0/lifecycle/pkg/core/metrics"
	mockm "github.com/aretw0/lifecycle/pkg/core/metrics/mock"
)

func TestMockProviderIntegration(t *testing.T) {
	m := mockm.New()
	metrics.SetProvider(m)
	defer metrics.SetProvider(nil)

	// Events
	metrics.GetProvider().IncEventEmitted("OSSignal")
	if m.EventsEmitted["OSSignal"] != 1 {
		t.Fatalf("expected 1 event emitted, got %d", m.EventsEmitted["OSSignal"])
	}

	// Worker starts
	metrics.GetProvider().IncWorkerStarted("bg")
	if m.WorkerStarts["bg"] != 1 {
		t.Fatalf("expected 1 worker start, got %d", m.WorkerStarts["bg"])
	}

	// Supervisor restart
	metrics.GetProvider().IncSupervisorRestart("sup", "OnFailure")
	if m.Restarts["sup"] != 1 {
		t.Fatalf("expected 1 supervisor restart, got %d", m.Restarts["sup"])
	}

	// Signal list
	metrics.GetProvider().IncSignalReceived("SIGINT")
	if len(m.Signals) == 0 || m.Signals[len(m.Signals)-1] != "SIGINT" {
		t.Fatalf("expected last signal to be SIGINT, got %#v", m.Signals)
	}
}
