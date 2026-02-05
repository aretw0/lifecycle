package supervisor

import (
	"context"
	"testing"

	"github.com/aretw0/lifecycle/pkg/worker"
)

// MockSuspendableWorker implements worker.Suspendable
type MockSuspendableWorker struct {
	MockWorker
	suspended bool
}

func (m *MockSuspendableWorker) Suspend(ctx context.Context) error {
	m.suspended = true
	return nil
}

func (m *MockSuspendableWorker) Resume(ctx context.Context) error {
	m.suspended = false
	return nil
}

func TestSupervisor_SuspendResume(t *testing.T) {
	// Create mock workers
	w1 := &MockSuspendableWorker{MockWorker: MockWorker{Name: "worker1", WaitChan: make(chan error)}}
	w2 := &MockWorker{Name: "worker2", WaitChan: make(chan error)} // Not suspendable

	s := New("root", StrategyOneForOne,
		Spec{
			Name: "worker1",
			Factory: func() (worker.Worker, error) {
				return w1, nil
			},
		},
		Spec{
			Name: "worker2",
			Factory: func() (worker.Worker, error) {
				return w2, nil
			},
		},
	).(*supervisor)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer s.Stop(context.Background())

	// Test Suspend
	if err := s.Suspend(ctx); err != nil {
		t.Fatalf("Suspend failed: %v", err)
	}

	if !w1.suspended {
		t.Error("worker1 should be suspended")
	}

	// Test Resume
	if err := s.Resume(ctx); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	if w1.suspended {
		t.Error("worker1 should be active")
	}

	// Clean up: finalize workers so guards can exit
	cancel() // Cancel context first
	close(w1.WaitChan)
	close(w2.WaitChan)
}
