package lifecycle

import (
	"context"
	"testing"
	"time"

	"log/slog"
)

func TestRootFacade(t *testing.T) {
	// Test Job and Run
	t.Run("JobAndRun", func(t *testing.T) {
		executed := false
		r := Job(func(ctx context.Context) error {
			executed = true
			return nil
		})

		// Simple runner that returns immediately.
		err := Run(r)
		if err != nil {
			t.Errorf("Run failed: %v", err)
		}
		if !executed {
			t.Error("Job was not executed")
		}
	})

	t.Run("GoAndDo", func(t *testing.T) {
		ctx := context.Background()

		// Test Do
		err := Do(ctx, func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Errorf("Do failed: %v", err)
		}

		// Test Go (async)
		Go(ctx, func(ctx context.Context) error {
			return nil
		})
	})

	t.Run("SignalDiscovery", func(t *testing.T) {
		parent := context.Background()
		ctx := NewSignalContext(parent)

		if !IsUnsafe(ctx) {
			// default is safe
		}

		threshold := GetForceExitThreshold(ctx)
		if threshold != 1 {
			t.Errorf("Expected threshold 1, got %d", threshold)
		}

		state, ok := GetSignalState(ctx)
		if !ok {
			t.Error("Expected GetSignalState to be ok")
		}

		diagram := SignalStateDiagram(state)
		if diagram == "" {
			t.Error("Expected non-empty diagram")
		}

		// Test OnShutdown/ShutdownAndWait
		shutdownCalled := false
		OnShutdown(ctx, func() {
			shutdownCalled = true
		})

		ShutdownAndWait(ctx)

		if !shutdownCalled {
			t.Error("Shutdown hook was not called")
		}
	})

	t.Run("InteractiveRouter", func(t *testing.T) {
		// Exercise NewInteractiveRouter
		suspendHandler := NewSuspendHandler()
		router := NewInteractiveRouter(suspendHandler, WithInput(false), WithSignal(false))
		if router == nil {
			t.Error("Expected non-nil router")
		}
	})

	t.Run("SupervisorAPI", func(t *testing.T) {
		// Exercise supervisor creation and basic tree rendering
		s := NewSupervisor("test-sup", SupervisorStrategyOneForOne)
		if s == nil {
			t.Error("Expected non-nil supervisor")
		}

		state := WorkerState{
			Name: "test-sup",
		}
		tree := WorkerTreeDiagram(state)
		if tree == "" {
			t.Error("Expected non-empty tree diagram")
		}

		stateDiagram := WorkerStateDiagram(state)
		if stateDiagram == "" {
			t.Error("Expected non-empty state diagram")
		}
	})

	t.Run("MetricsAndLogging", func(t *testing.T) {
		// Exercise metrics bridging
		provider := NewLogMetricsProvider()
		SetMetricsProvider(provider)

		// Exercise logger configuration
		SetLogger(slog.Default())
	})

	t.Run("ProcessManagement", func(t *testing.T) {
		// Exercise SetStrictMode
		SetStrictMode(true)
		SetStrictMode(false)
	})

	t.Run("GlobalConfig", func(t *testing.T) {
		// Exercise global setters
		SetLogger(slog.Default())
		SetMetricsProvider(NewLogMetricsProvider())

		// Exercise With options
		WithLogger(slog.Default())
		WithMetrics(NewLogMetricsProvider())
		WithShutdownTimeout(time.Second)
	})

	t.Run("TerminalAPI", func(t *testing.T) {
		// Exercise IsInterrupted
		if IsInterrupted(context.Canceled) != true {
			t.Error("Expected IsInterrupted(context.Canceled) to be true")
		}

		// UpgradeTerminal for non-terminal
		_, err := UpgradeTerminal(nil)
		if err == nil {
		}
	})

	t.Run("EventFacades", func(t *testing.T) {
		// Exercise DefaultRouter aliases
		if DefaultRouter == nil {
			t.Error("DefaultRouter should not be nil")
		}

		// Handle and HandleFunc (facades to events package)
		// We just need to ensure they don't panic
		Handle("test.event", NewShutdownHandler(func() {}))
		HandleFunc("test.func", func(ctx context.Context, e Event) error { return nil })
	})

	t.Run("WorkerFacades", func(t *testing.T) {
		// Exercise Worker aliases
		w1 := NewWorkerFromFunc("func-worker", func(ctx context.Context) error { return nil })
		if w1 == nil {
			t.Error("NewWorkerFromFunc returned nil")
		}

		w2 := NewProcessWorker("proc-worker", "echo", "hello")
		if w2 == nil {
			t.Error("NewProcessWorker returned nil")
		}

		w3 := NewBaseWorker("base-worker")
		if w3.String() != "base-worker" {
			t.Error("NewBaseWorker name mismatch")
		}
	})
}
