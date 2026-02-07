package mock

import (
	"testing"
)

func TestMockProvider_Behavior(t *testing.T) {
	m := New()

	t.Run("Signals", func(t *testing.T) {
		m.IncSignalReceived("SIGINT")
		m.IncSignalReceived("SIGTERM")
		m.IncSignalReceived("SIGINT")

		m.Mu.Lock()
		defer m.Mu.Unlock()
		if len(m.Signals) != 3 {
			t.Errorf("expected 3 signals, got %d", len(m.Signals))
		}
		counts := make(map[string]int)
		for _, s := range m.Signals {
			counts[s]++
		}
		if counts["SIGINT"] != 2 {
			t.Errorf("expected 2 SIGINT, got %d", counts["SIGINT"])
		}
		if counts["SIGTERM"] != 1 {
			t.Errorf("expected 1 SIGTERM, got %d", counts["SIGTERM"])
		}
	})

	t.Run("WorkerEvents", func(t *testing.T) {
		m.IncWorkerStarted("worker-1")
		m.IncWorkerStopped("worker-1")
		m.IncWorkerStarted("worker-2")

		m.Mu.Lock()
		defer m.Mu.Unlock()
		if m.WorkerStarts["worker-1"] != 1 {
			t.Errorf("worker-1 started count wrong")
		}
		if m.WorkerStops["worker-1"] != 1 {
			t.Errorf("worker-1 stopped count wrong")
		}
		if m.WorkerStarts["worker-2"] != 1 {
			t.Errorf("worker-2 started count wrong")
		}
	})

	t.Run("ControlPlane", func(t *testing.T) {
		m.IncHandlerExecuted("test.event")
		m.IncHandlerError("test.event", nil)

		m.Mu.Lock()
		defer m.Mu.Unlock()
		if m.HandlersExecuted["test.event"] != 1 {
			t.Errorf("handler execute count wrong")
		}
		if m.HandlerErrors["test.event"] != 1 {
			t.Errorf("handler error count wrong")
		}
	})
}
