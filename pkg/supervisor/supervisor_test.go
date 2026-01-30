package supervisor

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/metrics"
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
	// Note: We do not close WaitChan here to simulate "work finished".
	// In strict testing, Stop() should likely trigger the teardown of StartFunc,
	// but for Supervisor restart logic, we primarily test spontaneous failures via externally closing WaitChan.
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

// MockProvider for verifying metric calls
type mockMetrics struct {
	mu            sync.Mutex
	restarts      map[string]int // Supervisor -> count
	childRestarts map[string]int // Child -> count
}

func (m *mockMetrics) IncSupervisorRestart(s, strategy string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restarts[s]++
}

func (m *mockMetrics) IncChildRestart(s, c string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.childRestarts[c]++
}

// Stubs for other interface methods
func (m *mockMetrics) IncSignalReceived(sig string)                     {}
func (m *mockMetrics) IncProcessStarted()                               {}
func (m *mockMetrics) IncProcessFailed()                                {}
func (m *mockMetrics) IncTerminalUpgrade(success bool)                  {}
func (m *mockMetrics) IncHookExecuted()                                 {}
func (m *mockMetrics) IncHookPanicked()                                 {}
func (m *mockMetrics) ObserveHookDuration(d time.Duration)              {}
func (m *mockMetrics) IncWorkerStarted(wt string)                       {}
func (m *mockMetrics) IncWorkerStopped(wt string)                       {}
func (m *mockMetrics) IncWorkerFailed(wt string)                        {}
func (m *mockMetrics) ObserveWorkerDuration(wt string, d time.Duration) {}

func TestMetrics(t *testing.T) {
	helper := newFactoryHelper()
	mm := &mockMetrics{
		restarts:      make(map[string]int),
		childRestarts: make(map[string]int),
	}

	// Inject mock
	original := metrics.GetProvider()
	metrics.SetProvider(mm)
	defer metrics.SetProvider(original)

	sup := New("metrics-sup", StrategyOneForOne,
		Spec{Name: "worker-1", Factory: helper.makeFactory("worker-1")},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sup.Start(ctx)
	time.Sleep(20 * time.Millisecond)

	// Fail worker
	w1 := helper.getWorker("worker-1")
	w1.WaitChan <- errors.New("fail")
	close(w1.WaitChan)

	time.Sleep(100 * time.Millisecond)

	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.restarts["metrics-sup"] != 1 {
		t.Errorf("Expected 1 supervisor restart metric, got %d", mm.restarts["metrics-sup"])
	}
	if mm.childRestarts["worker-1"] != 1 {
		t.Errorf("Expected 1 child restart metric, got %d", mm.childRestarts["worker-1"])
	}
}

func TestStateRecursion(t *testing.T) {
	helper := newFactoryHelper()

	// Create leaf workers
	w1Fac := helper.makeFactory("leaf-1")

	// Create child supervisor
	subSup := New("sub-sup", StrategyOneForOne,
		Spec{Name: "leaf-1", Factory: w1Fac},
	)

	// Create root supervisor
	rootSup := New("root-sup", StrategyOneForOne,
		Spec{Name: "sub-sup", Factory: func() (worker.Worker, error) {
			return subSup, nil // Reuse instance for simplicity or factory wrapper
		}},
	)

	// In real use, Factory would create NEW supervisor.
	// For State() testing, we can just inspect the static structure if we start it.

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start
	if err := rootSup.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Allow start
	time.Sleep(50 * time.Millisecond)

	state := rootSup.State()

	if state.Name != "root-sup" {
		t.Errorf("Expected root name, got %s", state.Name)
	}

	if len(state.Children) != 1 {
		t.Fatalf("Expected 1 child for root, got %d", len(state.Children))
	}

	childState := state.Children[0]
	if childState.Name != "sub-sup" {
		t.Errorf("Expected child state name sub-sup, got %s", childState.Name)
	}

	if len(childState.Children) != 1 {
		t.Fatalf("Expected 1 grandchild, got %d", len(childState.Children))
	}

	grandChild := childState.Children[0]
	if grandChild.Name != "leaf-1" {
		t.Errorf("Expected grandchild leaf-1, got %s", grandChild.Name)
	}
}
