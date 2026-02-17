package runtime

// Benchmarks for lifecycle runtime components.
//
// Note: Some benchmarks (stack capture, observer) intentionally trigger panics
// to measure recovery overhead. This generates error logs during execution.
// To suppress logs: go test -bench=. ./pkg/core/runtime/ 2>/dev/null (Unix)
//                   go test -bench=. ./pkg/core/runtime/ 2>$null (Windows)

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/observe"
	"github.com/aretw0/lifecycle/pkg/core/supervisor"
	"github.com/aretw0/lifecycle/pkg/core/worker"
)

// BenchmarkGoVsRawGoroutine measures the overhead of lifecycle.Go compared to raw goroutines.
func BenchmarkGoVsRawGoroutine(b *testing.B) {
	b.Run("RawGoroutine", func(b *testing.B) {
		var wg sync.WaitGroup
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = simpleWork(ctx)
			}()
		}
		wg.Wait()
	})

	b.Run("LifecycleGo", func(b *testing.B) {
		ctx := context.Background()
		ctx = WithTaskTracking(ctx, &sync.WaitGroup{})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			task := Go(ctx, simpleWork)
			task.Wait()
		}
	})

	b.Run("LifecycleGoParallel", func(b *testing.B) {
		ctx := context.Background()
		var wg sync.WaitGroup
		ctx = WithTaskTracking(ctx, &wg)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				Go(ctx, simpleWork)
			}
		})
		wg.Wait()
	})
}

// BenchmarkGoWithStackCapture measures stack capture overhead in different modes.
func BenchmarkGoWithStackCapture(b *testing.B) {
	b.Run("Disabled", func(b *testing.B) {
		ctx := context.Background()
		var wg sync.WaitGroup
		ctx = WithTaskTracking(ctx, &wg)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Go(ctx, panicWork, WithStackCapture(false))
		}
		wg.Wait()
	})

	b.Run("Enabled", func(b *testing.B) {
		ctx := context.Background()
		var wg sync.WaitGroup
		ctx = WithTaskTracking(ctx, &wg)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Go(ctx, panicWork, WithStackCapture(true))
		}
		wg.Wait()
	})

	b.Run("AutoDetect", func(b *testing.B) {
		ctx := context.Background()
		var wg sync.WaitGroup
		ctx = WithTaskTracking(ctx, &wg)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Go(ctx, panicWork) // Default: auto-detect
		}
		wg.Wait()
	})
}

// BenchmarkGoWithObserver measures observer invocation overhead.
func BenchmarkGoWithObserver(b *testing.B) {
	b.Run("NoObserver", func(b *testing.B) {
		observe.SetObserver(nil)
		ctx := context.Background()
		var wg sync.WaitGroup
		ctx = WithTaskTracking(ctx, &wg)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Go(ctx, panicWork, WithStackCapture(true))
		}
		wg.Wait()
	})

	b.Run("WithObserver", func(b *testing.B) {
		observe.SetObserver(&benchmarkObserver{})
		defer observe.SetObserver(nil)

		ctx := context.Background()
		var wg sync.WaitGroup
		ctx = WithTaskTracking(ctx, &wg)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Go(ctx, panicWork, WithStackCapture(true))
		}
		wg.Wait()
	})
}

// BenchmarkDoExecution measures lifecycle.Do overhead.
func BenchmarkDoExecution(b *testing.B) {
	b.Run("SimpleWork", func(b *testing.B) {
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Do(ctx, simpleWork)
		}
	})

	b.Run("PanicWork", func(b *testing.B) {
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Do(ctx, panicWork)
		}
	})
}

// BenchmarkSupervisorTraversal measures State() overhead on supervision trees.
func BenchmarkSupervisorTraversal(b *testing.B) {
	b.Run("10Workers", func(b *testing.B) {
		sup := createSupervisorTree(10)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		_ = sup.Start(ctx)
		defer sup.Stop(ctx)

		// Wait for all workers to start
		time.Sleep(50 * time.Millisecond)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = sup.State()
		}
	})

	b.Run("100Workers", func(b *testing.B) {
		sup := createSupervisorTree(100)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		_ = sup.Start(ctx)
		defer sup.Stop(ctx)

		// Wait for all workers to start
		time.Sleep(100 * time.Millisecond)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = sup.State()
		}
	})

	b.Run("1000Workers", func(b *testing.B) {
		sup := createSupervisorTree(1000)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		_ = sup.Start(ctx)
		defer sup.Stop(ctx)

		// Wait for all workers to start
		time.Sleep(500 * time.Millisecond)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = sup.State()
		}
	})
}

// BenchmarkMemoryFootprint measures memory allocation patterns.
func BenchmarkMemoryFootprint(b *testing.B) {
	b.Run("GoAllocation", func(b *testing.B) {
		ctx := context.Background()
		var wg sync.WaitGroup
		ctx = WithTaskTracking(ctx, &wg)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Go(ctx, simpleWork)
		}
		wg.Wait()
	})

	b.Run("SupervisorAllocation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = createSupervisorTree(10)
		}
	})
}

// Helper functions

func simpleWork(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Simulate minimal work
		_ = 1 + 1
		return nil
	}
}

func panicWork(ctx context.Context) error {
	panic("benchmark panic")
}

// benchmarkObserver is a minimal observer for benchmarking.
type benchmarkObserver struct{}

func (o *benchmarkObserver) OnGoroutinePanicked(recovered any, stack []byte) {
	// Minimal processing for benchmark
	_ = recovered
	_ = stack
}

func (o *benchmarkObserver) OnProcessStarted(pid int)  {}
func (o *benchmarkObserver) OnProcessFailed(err error) {}
func (o *benchmarkObserver) LogDebug(msg string, args ...any) {}
func (o *benchmarkObserver) LogInfo(msg string, args ...any)  {}
func (o *benchmarkObserver) LogWarn(msg string, args ...any)  {}
func (o *benchmarkObserver) LogError(msg string, args ...any) {}

// createSupervisorTree creates a supervisor with N workers for benchmarking.
func createSupervisorTree(n int) worker.Worker {
	specs := make([]supervisor.Spec, n)
	for i := 0; i < n; i++ {
		idx := i
		specs[i] = supervisor.Spec{
			Name: fmt.Sprintf("worker-%d", idx),
			Factory: func() (worker.Worker, error) {
				return &benchmarkWorker{name: fmt.Sprintf("worker-%d", idx)}, nil
			},
			RestartPolicy: supervisor.RestartNever,
		}
	}

	return supervisor.New("benchmark-supervisor", supervisor.StrategyOneForOne, specs...)
}

// benchmarkWorker is a minimal worker implementation for benchmarking.
type benchmarkWorker struct {
	name   string
	ctx    context.Context
	cancel context.CancelFunc
	wait   chan error
}

func (w *benchmarkWorker) Start(ctx context.Context) error {
	w.ctx, w.cancel = context.WithCancel(ctx)
	w.wait = make(chan error, 1)

	go func() {
		<-w.ctx.Done()
		w.wait <- nil
		close(w.wait)
	}()

	return nil
}

func (w *benchmarkWorker) Stop(ctx context.Context) error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

func (w *benchmarkWorker) Wait() <-chan error {
	return w.wait
}

func (w *benchmarkWorker) String() string {
	return w.name
}

func (w *benchmarkWorker) State() worker.State {
	return worker.State{
		Name:   w.name,
		Status: worker.StatusRunning,
		Metadata: map[string]string{
			worker.MetadataType: string(worker.TypeFunc),
		},
	}
}
