package supervisor

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/worker"
)

// MockWorker is a controllable worker for testing.
type MockWorker struct {
	Name      string
	StartFunc func(ctx context.Context) error
	WaitChan  chan error
	mu        sync.Mutex
	status    worker.Status
}

func (w *MockWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.status = worker.StatusRunning
	if w.StartFunc != nil {
		return w.StartFunc(ctx)
	}
	return nil
}

func (w *MockWorker) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.status = worker.StatusStopped
	// Simulating stop by closing wait chan if not already closed is tricky without coordination.
	// Usually WaitChan is closed when work finishes.
	// For testing, we might want to trigger exit externally.
	return nil
}

func (w *MockWorker) Wait() <-chan error {
	return w.WaitChan
}

func (w *MockWorker) String() string { return w.Name }
func (w *MockWorker) State() worker.State {
	w.mu.Lock()
	defer w.mu.Unlock()
	return worker.State{Name: w.Name, Status: w.status}
}

// factoryHelper tracks instantiations
type factoryHelper struct {
	mu     sync.Mutex
	counts map[string]int
	active map[string]*MockWorker
}

func newFactoryHelper() *factoryHelper {
	return &factoryHelper{
		counts: make(map[string]int),
		active: make(map[string]*MockWorker),
	}
}

func (f *factoryHelper) makeFactory(name string) Factory {
	return func() (worker.Worker, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.counts[name]++

		w := &MockWorker{
			Name:     name,
			WaitChan: make(chan error, 1),
			status:   worker.StatusPending,
		}
		f.active[name] = w
		return w, nil
	}
}

func (f *factoryHelper) getWorker(name string) *MockWorker {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.active[name]
}

func (f *factoryHelper) getCount(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.counts[name]
}

func TestOneForOne(t *testing.T) {
	helper := newFactoryHelper()

	sup := New("test-sup", StrategyOneForOne,
		Spec{Name: "worker-1", Factory: helper.makeFactory("worker-1")},
		Spec{Name: "worker-2", Factory: helper.makeFactory("worker-2")},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sup.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for start
	time.Sleep(50 * time.Millisecond)

	if helper.getCount("worker-1") != 1 || helper.getCount("worker-2") != 1 {
		t.Fatalf("Expected 1 instance of each worker, got %v and %v", helper.getCount("worker-1"), helper.getCount("worker-2"))
	}

	// Fail worker-1
	w1 := helper.getWorker("worker-1")
	w1.WaitChan <- errors.New("oops")
	close(w1.WaitChan)

	// Wait for restart
	time.Sleep(100 * time.Millisecond)

	// Worker 1 should be restarted (count 2)
	if helper.getCount("worker-1") != 2 {
		t.Errorf("Expected worker-1 to revert (count 2), got %d", helper.getCount("worker-1"))
	}
	// Worker 2 should NOT be restarted (count 1)
	if helper.getCount("worker-2") != 1 {
		t.Errorf("Expected worker-2 to remain (count 1), got %d", helper.getCount("worker-2"))
	}
}

func TestOneForAll(t *testing.T) {
	helper := newFactoryHelper()

	sup := New("test-sup-all", StrategyOneForAll,
		Spec{Name: "worker-A", Factory: helper.makeFactory("worker-A")},
		Spec{Name: "worker-B", Factory: helper.makeFactory("worker-B")},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sup.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Fail worker-A
	wA := helper.getWorker("worker-A")
	wA.WaitChan <- errors.New("fail")
	close(wA.WaitChan)

	// Wait for restart cycle
	time.Sleep(200 * time.Millisecond)

	// Both should be restarted
	if count := helper.getCount("worker-A"); count != 2 {
		t.Errorf("Expected worker-A restarts: 2, got %d", count)
	}
	if count := helper.getCount("worker-B"); count != 2 {
		t.Errorf("Expected worker-B restarts: 2, got %d", count)
	}
}
