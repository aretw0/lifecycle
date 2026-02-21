package runtime_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"github.com/aretw0/lifecycle/pkg/core/observe"
	"github.com/aretw0/lifecycle/pkg/core/runtime"
)

// TestGo_GlobalFallback verifies that Go uses the default tracker when
// called with a context not managed by Run.
func TestGo_GlobalFallback(t *testing.T) {
	var executed atomicBool
	Executed := &executed

	// Use a background context (no tracker)
	ctx := context.Background()

	// Start a goroutine
	runtime.Go(ctx, func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		Executed.Set(true)
		return nil
	})

	// Wait for it using the global waiter
	runtime.WaitForGlobal()

	if !Executed.Get() {
		t.Error("Global fallback did not wait for task")
	}
}

// TestGo_PanicSafe verifies that a panic in a background task is caught
// and does not crash the test runner.
func TestGo_PanicSafe(t *testing.T) {
	// Setup metrics to verify Do recorded the failure
	provider := &mockMetricsProvider{}
	metrics.SetProvider(provider)

	ctx := context.Background()
	var wg sync.WaitGroup

	// Manually inject a tracker to isolate this test from global
	ctx = runtime.WithTaskTracking(ctx, &wg)

	// runtime.Go adds to the WG automatically, so we just check Wait() behavior.
	runtime.Go(ctx, func(ctx context.Context) error {
		panic("boom")
	})

	// Wait for the task to finish (and recover)
	wg.Wait()

	// Verify metrics
	if provider.criticalSectionFinished != 1 {
		t.Errorf("Expected 1 critical section finished, got %d", provider.criticalSectionFinished)
	}
	if provider.lastSuccess {
		t.Error("Expected critical section to be marked as failed due to panic")
	}
}

func TestGo_OnGoroutinePanicked_StackCaptureEnabled(t *testing.T) {
	observer := &mockObserver{}
	observe.SetObserver(observer)
	defer observe.SetObserver(nil)

	var wg sync.WaitGroup
	ctx := runtime.WithTaskTracking(context.Background(), &wg)
	runtime.Go(ctx, func(ctx context.Context) error {
		panic("boom")
	}, runtime.WithStackCapture(true))

	wg.Wait()

	panickedCalls, stack := observer.snapshot()
	if panickedCalls != 1 {
		t.Fatalf("Expected 1 panic callback, got %d", panickedCalls)
	}
	if len(stack) == 0 {
		t.Error("Expected stack capture when enabled")
	}
}

func TestGo_OnGoroutinePanicked_StackCaptureDisabled(t *testing.T) {
	observer := &mockObserver{}
	observe.SetObserver(observer)
	defer observe.SetObserver(nil)

	var wg sync.WaitGroup
	ctx := runtime.WithTaskTracking(context.Background(), &wg)
	runtime.Go(ctx, func(ctx context.Context) error {
		panic("boom")
	}, runtime.WithStackCapture(false))

	wg.Wait()

	panickedCalls, stack := observer.snapshot()
	if panickedCalls != 1 {
		t.Fatalf("Expected 1 panic callback, got %d", panickedCalls)
	}
	if len(stack) != 0 {
		t.Error("Expected no stack capture when disabled")
	}
}

func TestGo_OnGoroutinePanicked_AutoDetectDebug(t *testing.T) {
	observer := &mockObserver{}
	observe.SetObserver(observer)
	defer observe.SetObserver(nil)

	logger := log.GetLogger()
	defer log.SetLogger(logger)

	debugLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	log.SetLogger(debugLogger)

	var wg sync.WaitGroup
	ctx := runtime.WithTaskTracking(context.Background(), &wg)
	runtime.Go(ctx, func(ctx context.Context) error {
		panic("boom")
	})

	wg.Wait()

	panickedCalls, stack := observer.snapshot()
	if panickedCalls != 1 {
		t.Fatalf("Expected 1 panic callback, got %d", panickedCalls)
	}
	if len(stack) == 0 {
		t.Error("Expected stack capture when debug logging is enabled")
	}
}

// TestGo_ContextPropagation verifies that values set in the context
// are passed down to the background Goroutine.
func TestGo_ContextPropagation(t *testing.T) {
	key := "trace-id"
	val := "12345"
	ctx := context.WithValue(context.Background(), key, val)

	var executed atomicBool
	Executed := &executed

	runtime.Go(ctx, func(ctx context.Context) error {
		v := ctx.Value(key)
		if v != val {
			t.Errorf("Expected context value %s, got %v", val, v)
		}
		Executed.Set(true)
		return nil
	})

	runtime.WaitForGlobal()

	if !Executed.Get() {
		t.Error("Task not executed")
	}
}

// TestSleep_Cancellation verifies that Sleep returns early if context is cancelled.
func TestSleep_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	start := time.Now()

	// Trigger cancel after a tiny delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	// Sleep for much longer
	err := runtime.Sleep(ctx, 1*time.Second)

	duration := time.Since(start)

	if err == nil {
		t.Error("Expected error from Sleep, got nil")
	}
	if duration > 100*time.Millisecond {
		t.Errorf("Sleep took too long: %v", duration)
	}
}

// -- Helpers --

type atomicBool struct {
	sync.Mutex
	val bool
}

func (a *atomicBool) Set(v bool) {
	a.Lock()
	defer a.Unlock()
	a.val = v
}

func (a *atomicBool) Get() bool {
	a.Lock()
	defer a.Unlock()
	return a.val
}

type mockMetricsProvider struct {
	metrics.NoOpProvider // Embed defaults to satisfy interface

	criticalSectionFinished int
	lastSuccess             bool
}

// Override only what we need to test
func (m *mockMetricsProvider) IncCriticalSectionFinished(success bool) {
	m.criticalSectionFinished++
	m.lastSuccess = success
}

type mockObserver struct {
	panickedCalls int
	recovered     any
	stack         []byte
	mu            sync.Mutex
}

func (m *mockObserver) OnProcessStarted(int)    {}
func (m *mockObserver) OnProcessFailed(error)   {}
func (m *mockObserver) LogDebug(string, ...any) {}
func (m *mockObserver) LogInfo(string, ...any)  {}
func (m *mockObserver) LogWarn(string, ...any)  {}
func (m *mockObserver) LogError(string, ...any) {}

func (m *mockObserver) OnGoroutinePanicked(recovered any, stack []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.panickedCalls++
	m.recovered = recovered
	m.stack = stack
}

func (m *mockObserver) snapshot() (int, []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stackCopy := make([]byte, len(m.stack))
	copy(stackCopy, m.stack)
	return m.panickedCalls, stackCopy
}
