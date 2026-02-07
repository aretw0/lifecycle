package supervisor

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"github.com/aretw0/lifecycle/pkg/core/metrics/mock"
	"github.com/aretw0/lifecycle/pkg/core/worker"
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
	if w.WaitChan != nil {
		select {
		case <-w.WaitChan:
			// Already closed or has error
		default:
			close(w.WaitChan)
		}
	}
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
			status:   worker.StatusCreated,
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
	defer func() {
		cancel()
		sup.Stop(context.Background())
	}()

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
	defer func() {
		cancel()
		<-sup.Wait()
	}()

	if err := sup.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Fail worker-A
	wA := helper.getWorker("worker-A")
	wA.WaitChan <- errors.New("fail")
	close(wA.WaitChan)

	// Wait deterministically for both workers to restart (count should be 2)
	// Poll the actual factory helper counts instead of guessing timing
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatalf("Timeout waiting for restarts. worker-A: %d, worker-B: %d",
				helper.getCount("worker-A"), helper.getCount("worker-B"))
		case <-ticker.C:
			if helper.getCount("worker-A") >= 2 && helper.getCount("worker-B") >= 2 {
				goto verified
			}
		}
	}

verified:
	// Both should be restarted
	if count := helper.getCount("worker-A"); count != 2 {
		t.Errorf("Expected worker-A restarts: 2, got %d", count)
	}
	if count := helper.getCount("worker-B"); count != 2 {
		t.Errorf("Expected worker-B restarts: 2, got %d", count)
	}
}

func TestMetrics(t *testing.T) {
	helper := newFactoryHelper()
	mm := mock.New() // Use centralized mock

	// Inject mock
	original := metrics.GetProvider()
	metrics.SetProvider(mm)
	defer metrics.SetProvider(original)

	sup := New("metrics-sup", StrategyOneForOne,
		Spec{Name: "worker-1", Factory: helper.makeFactory("worker-1")},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		cancel()
		<-sup.Wait()
	}()

	sup.Start(ctx)
	time.Sleep(20 * time.Millisecond)

	// Fail worker
	w1 := helper.getWorker("worker-1")
	w1.WaitChan <- errors.New("fail")
	close(w1.WaitChan)

	time.Sleep(100 * time.Millisecond)

	mm.Mu.Lock()
	defer mm.Mu.Unlock() // Access fields directly or via helper methods if added?
	// The mock struct fields are public in the file I created.
	// But Wait, I need to make sure I am importing the mock package.
	// I will update imports in a separate step or via multi-replace if possible.

	if mm.Restarts["metrics-sup"] != 1 {
		t.Errorf("Expected 1 supervisor restart metric, got %d", mm.Restarts["metrics-sup"])
	}
	if mm.ChildRestarts["worker-1"] != 1 {
		t.Errorf("Expected 1 child restart metric, got %d", mm.ChildRestarts["worker-1"])
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
	defer func() {
		cancel()
		rootSup.Stop(context.Background())
	}()

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

type InjectableMockWorker struct {
	MockWorker
	mu   sync.Mutex
	envs map[string]string
}

func (w *InjectableMockWorker) SetEnv(k, v string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.envs == nil {
		w.envs = make(map[string]string)
	}
	w.envs[k] = v
}

func (w *InjectableMockWorker) GetEnv(k string) string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.envs == nil {
		return ""
	}
	return w.envs[k]
}

func TestHandoverProtocol(t *testing.T) {
	var (
		mu           sync.Mutex
		resumeID     string
		prevExit     string
		restartCount int
	)

	sup := New("handover-sup", StrategyOneForOne,
		Spec{
			Name: "worker-1",
			Factory: func() (worker.Worker, error) {
				mu.Lock()
				defer mu.Unlock()
				restartCount++
				return &InjectableMockWorker{
					MockWorker: MockWorker{
						Name:     "worker-1",
						WaitChan: make(chan error, 1),
					},
				}, nil
			},
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		cancel()
		<-sup.Wait()
	}()

	sup.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	// Capture first run info
	supImpl := sup.(*supervisor)
	supImpl.mu.Lock()
	w1 := supImpl.children["worker-1"].(*InjectableMockWorker)
	supImpl.mu.Unlock()

	resumeID = w1.GetEnv(worker.EnvResumeID)
	if resumeID == "" {
		t.Fatal("LIFECYCLE_RESUME_ID should be set")
	}

	// Fail the worker
	w1.WaitChan <- errors.New("crash")
	close(w1.WaitChan)

	// Wait for restart
	time.Sleep(100 * time.Millisecond)

	// Verify second run
	supImpl.mu.Lock()
	w2 := supImpl.children["worker-1"].(*InjectableMockWorker)
	supImpl.mu.Unlock()

	if got := w2.GetEnv(worker.EnvResumeID); got != resumeID {
		t.Errorf("ResumeID should persist. Expected %s, got %s", resumeID, got)
	}
	prevExit = w2.GetEnv(worker.EnvPrevExit)
	if prevExit != "-1" {
		t.Errorf("PREV_EXIT should be -1 after crash, got %s", prevExit)
	}

	if restartCount != 2 {
		t.Errorf("Expected 2 factory calls, got %d", restartCount)
	}
}

func TestSupervisor_Backoff(t *testing.T) {
	helper := newFactoryHelper()

	backoff := Backoff{
		InitialInterval: 50 * time.Millisecond,
		MaxInterval:     200 * time.Millisecond,
		Multiplier:      2.0,
	}

	sup := New("backoff-sup", StrategyOneForOne,
		Spec{
			Name:    "worker-1",
			Factory: helper.makeFactory("worker-1"),
			Backoff: backoff,
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		<-sup.Wait()
	}()

	if err := sup.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	time.Sleep(20 * time.Millisecond) // Wait start

	// Fail 1st time (Interval = 50ms)
	w1 := helper.getWorker("worker-1")
	w1.WaitChan <- errors.New("fail-1")
	close(w1.WaitChan)

	// Wait check (should be > 50ms)
	time.Sleep(20 * time.Millisecond)
	if count := helper.getCount("worker-1"); count != 1 {
		t.Errorf("Worker restarted too fast! Count=%d", count)
	}

	time.Sleep(60 * time.Millisecond) // Total wait ~80ms
	if count := helper.getCount("worker-1"); count != 2 {
		t.Errorf("Worker should represent after 50ms! Count=%d", count)
	}

	// Fail 2nd time (Interval = 100ms)
	w2 := helper.getWorker("worker-1")
	w2.WaitChan <- errors.New("fail-2")
	close(w2.WaitChan) // Should trigger backoff 100ms

	time.Sleep(60 * time.Millisecond)
	if count := helper.getCount("worker-1"); count != 2 {
		t.Errorf("Worker restarted too fast (expected 100ms delay)! Count=%d", count)
	}

	time.Sleep(150 * time.Millisecond) // Total ~210ms
	if count := helper.getCount("worker-1"); count != 3 {
		t.Errorf("Worker should restart after 100ms! Count=%d", count)
	}
}

func TestSupervisor_DynamicTopology(t *testing.T) {
	helper := newFactoryHelper()
	sup := New("dynamic-sup", StrategyOneForOne)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		cancel()
		sup.Stop(context.Background())
	}()

	if err := sup.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Add worker
	spec := Spec{Name: "dynamic-1", Factory: helper.makeFactory("dynamic-1")}
	if err := sup.Add(spec); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	time.Sleep(20 * time.Millisecond)
	if helper.getCount("dynamic-1") != 1 {
		t.Error("Worker not started after Add")
	}

	// Add duplicate
	if err := sup.Add(spec); err == nil {
		t.Error("Add duplicate should error")
	}

	// Remove worker
	if err := sup.Remove("dynamic-1"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify stopped
	w1 := helper.getWorker("dynamic-1")
	w1.mu.Lock()
	if w1.status != worker.StatusStopped {
		t.Error("Worker should be stopped after Remove")
	}
	w1.mu.Unlock()

	// Remove unknown
	if err := sup.Remove("dynamic-1"); err == nil {
		t.Error("Remove unknown should error")
	}
}

func TestSupervisor_CircuitBreaker(t *testing.T) {
	helper := newFactoryHelper()
	mm := mock.New()
	original := metrics.GetProvider()
	metrics.SetProvider(mm)
	defer metrics.SetProvider(original)

	backoff := Backoff{
		MaxRestarts: 3,
		MaxDuration: 1 * time.Second,
	}

	sup := New("breaker-sup", StrategyOneForOne,
		Spec{
			Name:    "flaky-worker",
			Factory: helper.makeFactory("flaky-worker"),
			Backoff: backoff,
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		cancel()
		sup.Stop(context.Background())
	}()

	if err := sup.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	// Trigger 4 failures quickly
	for i := 0; i < 4; i++ {
		w := helper.getWorker("flaky-worker")
		if w == nil {
			t.Fatalf("Worker not found at iteration %d", i)
		}
		w.WaitChan <- errors.New("boom")
		close(w.WaitChan)
		time.Sleep(50 * time.Millisecond) // Wait for supervisor to process and restart
	}

	// The 4th failure should trigger the breaker and stop restarts
	time.Sleep(100 * time.Millisecond)

	mm.Mu.Lock()
	count := helper.getCount("flaky-worker")
	triggers := mm.CircuitBreakerTriggers["flaky-worker"]
	mm.Mu.Unlock()

	if count > 4 {
		t.Errorf("Circuit breaker failed to stop restarts. Count=%d", count)
	}

	if triggers == 0 {
		t.Error("Circuit breaker metric not incremented")
	}
}

func TestSupervisor_RestartPolicies(t *testing.T) {
	helper := newFactoryHelper()

	sup := New("policy-sup", StrategyOneForOne,
		Spec{
			Name:          "on-failure-success",
			Factory:       helper.makeFactory("on-failure-success"),
			RestartPolicy: RestartOnFailure,
		},
		Spec{
			Name:          "on-failure-fail",
			Factory:       helper.makeFactory("on-failure-fail"),
			RestartPolicy: RestartOnFailure,
		},
		Spec{
			Name:          "never",
			Factory:       helper.makeFactory("never"),
			RestartPolicy: RestartNever,
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		cancel()
		<-sup.Wait()
	}()

	if err := sup.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	// 1. OnFailure with success (exit code 0/nil) -> Should NOT restart
	w1 := helper.getWorker("on-failure-success")
	w1.WaitChan <- nil
	close(w1.WaitChan)
	time.Sleep(50 * time.Millisecond)
	if helper.getCount("on-failure-success") > 1 {
		t.Error("on-failure-success restarted despite successful exit")
	}

	// 2. OnFailure with failure -> Should restart
	w2 := helper.getWorker("on-failure-fail")
	w2.WaitChan <- errors.New("boom")
	close(w2.WaitChan)
	time.Sleep(100 * time.Millisecond)
	if helper.getCount("on-failure-fail") < 2 {
		t.Error("on-failure-fail failed to restart")
	}

	// 3. Never -> Should NOT restart regardless of exit
	w3 := helper.getWorker("never")
	w3.WaitChan <- errors.New("boom")
	close(w3.WaitChan)
	time.Sleep(50 * time.Millisecond)
	if helper.getCount("never") > 1 {
		t.Error("never restarted despite RestartNever policy")
	}
}

func TestSupervisor_String(t *testing.T) {
	supervisor := New("test-supervisor", StrategyOneForOne)
	str := supervisor.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
	expected := "Supervisor(test-supervisor)"
	if str != expected {
		t.Errorf("Expected %q, got %q", expected, str)
	}
}
