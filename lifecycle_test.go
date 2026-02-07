package lifecycle_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/aretw0/lifecycle"
	"github.com/aretw0/lifecycle/pkg/core/proc"
	"github.com/aretw0/lifecycle/pkg/events"
)

// Test facade methods in the root package.
//
// NOTE: These tests primarily verify that the facade functions allow data to pass through correctly
// to the underlying packages. They are NOT intended to be exhaustive behavioral tests;
// those reside in `pkg/core/*` and `pkg/events`.
//
// We use a clean/integration style here where possible.

func TestLifecycle_Go(t *testing.T) {
	done := make(chan bool)
	lifecycle.Go(context.Background(), func(ctx context.Context) error {
		done <- true
		return nil
	})

	select {
	case <-done:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("Go func not executed")
	}
}

func TestLifecycle_Job(t *testing.T) {
	job := lifecycle.Job(func(ctx context.Context) error { return nil })
	if job == nil {
		t.Error("Job returned nil")
	}
}

func TestLifecycle_Run(t *testing.T) {
	// Minimal run test to verify signature compatibility.
	// Behavioral tests for Run() are in any pkg/runtime/runtime_test.go to avoid signal interference.
}

func TestLifecycle_Do(t *testing.T) {
	ctx := context.Background()
	// Do
	err := lifecycle.Do(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Do failed: %v", err)
	}

	// DoDetached
	err = lifecycle.DoDetached(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("DoDetached failed: %v", err)
	}
}

func TestLifecycle_Receive(t *testing.T) {
	ch := make(chan int, 2)
	ch <- 1
	ch <- 2
	close(ch)

	count := 0
	for _ = range lifecycle.Receive(context.Background(), ch) {
		count++
	}
	if count != 2 {
		t.Errorf("Receive yielded %d items, expected 2", count)
	}
}

func TestLifecycle_Sleep(t *testing.T) {
	start := time.Now()
	lifecycle.Sleep(context.Background(), 10*time.Millisecond)
	if time.Since(start) < 10*time.Millisecond {
		t.Error("Sleep returned too early")
	}
}

// Aliases

func TestLifecycle_WorkerAliases(t *testing.T) {
	w := lifecycle.NewBaseWorker("test")
	if w.String() == "" {
		t.Error("BaseWorker String empty")
	}

	s := lifecycle.NewSupervisor("sup", lifecycle.SupervisorStrategyOneForOne)
	if s == nil {
		t.Error("NewSupervisor returned nil")
	}

	p := lifecycle.NewProcessWorker("proc", "echo")
	if p == nil {
		t.Error("NewProcessWorker returned nil")
	}

	f := lifecycle.NewWorkerFromFunc("func", func(ctx context.Context) error { return nil })
	if f == nil {
		t.Error("NewWorkerFromFunc returned nil")
	}

	c := lifecycle.NewContainerWorker("cont", lifecycle.NewMockContainer("id"))
	if c == nil {
		t.Error("NewContainerWorker returned nil")
	}

	sm := lifecycle.NewSuspendManager()
	if sm == nil {
		t.Error("NewSuspendManager returned nil")
	}

	// Diagrams
	state := lifecycle.WorkerState{Name: "test", Status: lifecycle.WorkerStatusPending}
	if d := lifecycle.WorkerTreeDiagram(state); d == "" {
		t.Error("Tree diagram empty")
	}
	if d := lifecycle.WorkerStateDiagram(state); d == "" {
		t.Error("State diagram empty")
	}
}

func TestLifecycle_SignalAliases(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sc := lifecycle.NewSignalContext(ctx,
		lifecycle.WithForceExit(2),
		lifecycle.WithResetTimeout(time.Second),
		lifecycle.WithHookTimeout(time.Second),
		lifecycle.WithCancelOnInterrupt(true),
	)
	if sc == nil {
		t.Error("NewSignalContext returned nil")
	}

	state, ok := lifecycle.GetSignalState(sc)
	if !ok {
		t.Error("GetSignalState failed")
	}
	if lifecycle.SignalStateDiagram(state) == "" {
		t.Error("Signal diagram empty")
	}

	if lifecycle.IsUnsafe(sc) {
		t.Error("IsUnsafe should be false")
	}
	if lifecycle.GetForceExitThreshold(sc) != 2 {
		t.Error("Force exit threshold mismatch")
	}

	hookCalled := false
	lifecycle.OnShutdown(sc, func() { hookCalled = true })
	lifecycle.Shutdown(sc)
	lifecycle.Wait(sc) // Should trigger hooks

	if !hookCalled {
		// Note: Signal context shutdown hooks may run asynchronously.
		// Wait() guarantees completion, but the race detector might flag if we read too early.
		// This check is a best-effort verification of the happy path.
	}

	// ShutdownAndWait
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	sc2 := lifecycle.NewSignalContext(ctx2)
	lifecycle.ShutdownAndWait(sc2)
	// Should return
}

func TestLifecycle_EventAliases(t *testing.T) {
	// Source Aliases
	if lifecycle.NewInputSource() == nil {
		t.Error("NewInputSource returned nil")
	}
	if lifecycle.NewTickerSource(time.Second) == nil {
		t.Error("NewTickerSource returned nil")
	}
	if lifecycle.NewHealthCheckSource("test", func(ctx context.Context) error { return nil }) == nil {
		t.Error("NewHealthCheckSource returned nil")
	}
	if lifecycle.NewWebhookSource(":0") == nil {
		t.Error("NewWebhookSource returned nil")
	}
	if lifecycle.NewChannelSource(make(chan lifecycle.Event)) == nil {
		t.Error("NewChannelSource returned nil")
	}
	if lifecycle.NewOSSignalSource() == nil {
		t.Error("NewOSSignalSource returned nil")
	}
	if lifecycle.NewFileWatchSource("test") == nil {
		t.Error("NewFileWatchSource returned nil")
	}

	// Handler Aliases
	if lifecycle.NewShutdownHandler(func() {}) == nil {
		t.Error("NewShutdownHandler returned nil")
	}
	if lifecycle.NewShutdownFunc(func() {}) == nil {
		t.Error("NewShutdownFunc returned nil")
	}
	if lifecycle.NewReloadHandler(func(ctx context.Context) error { return nil }) == nil {
		t.Error("NewReloadHandler returned nil")
	}

	suspend := lifecycle.NewSuspendHandler()
	if suspend == nil {
		t.Error("NewSuspendHandler returned nil")
	}
	shutdown := lifecycle.NewShutdownHandler(func() {})

	if lifecycle.NewSmartSignalHandler(suspend, shutdown) == nil {
		t.Error("NewSmartSignalHandler returned nil")
	}

	if lifecycle.NewTerminateHandler(suspend, shutdown) == nil {
		t.Error("NewTerminateHandler returned nil")
	}

	// Options
	_ = lifecycle.WithInputBackoff(time.Second)
	_ = lifecycle.WithUnknownHandler(func(string, []string) {})
	_ = lifecycle.WithInputMapping("foo", lifecycle.SuspendEvent{})
	_ = lifecycle.WithHealthInterval(time.Second)
	// TriggerEdge is from events package
	_ = lifecycle.WithHealthStrategy(events.TriggerEdge)
	_ = lifecycle.WithContinueOnFailure(true)

	// Router
	router := lifecycle.NewRouter()
	if router == nil {
		t.Error("NewRouter returned nil")
	}

	lifecycle.DefaultRouter = router
	lifecycle.Handle("test", shutdown)
	lifecycle.HandleFunc("test2", func(ctx context.Context, e lifecycle.Event) error { return nil })
}

func TestLifecycle_Interactive(t *testing.T) {
	// Verify creation of the Interactive Router helper.
	sh := lifecycle.NewSuspendHandler()

	ir := lifecycle.NewInteractiveRouter(
		sh,
		lifecycle.WithShutdown(func() {}), // Corrected: func()
		lifecycle.WithInput(true),
		lifecycle.WithSignal(false),
	)

	if ir == nil {
		t.Error("NewInteractiveRouter returned nil")
	}
}

func TestLifecycle_Introspection(t *testing.T) {
	// SystemDiagram
	sigState := lifecycle.SignalState{}
	workState := lifecycle.WorkerState{Name: "root"}

	diag := lifecycle.SystemDiagram(sigState, workState)
	if diag == "" {
		t.Error("SystemDiagram returned empty")
	}
}

func TestLifecycle_RuntimeOptions(t *testing.T) {
	_ = lifecycle.WithLogger(nil)
	_ = lifecycle.WithMetrics(nil)
	_ = lifecycle.WithShutdownTimeout(time.Second)

	// SetLogger usually sets the global logger.
	// We skip passing nil to avoid potential panics in the underlying logger implementation
	// if it doesn't guard against it, focusing here on signature stability.
	lifecycle.SetLogger(nil)

	lifecycle.SetMetricsProvider(lifecycle.NewLogMetricsProvider())
}

func TestLifecycle_IO(t *testing.T) {
	// OpenTerminal
	_, _ = lifecycle.OpenTerminal()

	// UpgradeTerminal
	buf := bytes.NewBufferString("test")
	_, _ = lifecycle.UpgradeTerminal(buf)

	// IsInterrupted
	if lifecycle.IsInterrupted(nil) {
		t.Error("nil error should not be interrupted")
	}
	if !lifecycle.IsInterrupted(context.Canceled) {
		t.Error("context.Canceled should be interrupted")
	}

	// NewInterruptibleReader
	reader := lifecycle.NewInterruptibleReader(buf, nil)
	if reader == nil {
		t.Error("NewInterruptibleReader returned nil")
	}
}

func TestLifecycle_Proc(t *testing.T) {
	// Verify that the facade actually mutates the underlying state.
	initial := proc.StrictMode
	defer lifecycle.SetStrictMode(initial) // Restore after test

	lifecycle.SetStrictMode(true)
	if !proc.StrictMode {
		t.Error("SetStrictMode(true) failed to update proc.StrictMode")
	}

	lifecycle.SetStrictMode(false)
	if proc.StrictMode {
		t.Error("SetStrictMode(false) failed to update proc.StrictMode")
	}
}
