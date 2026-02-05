package runtime

import (
	"context"
	"log/slog"
	"os"
	stdruntime "runtime"
	"sync"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"github.com/aretw0/lifecycle/pkg/core/signal"
)

// Sleep pauses the current goroutine for at least the duration d.
// Unlike time.Sleep, it returns immediately if the context is cancelled.
// Returns nil if the duration passed, or ctx.Err() if cancelled.
func Sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// option structs
type loggerOpt struct{ l *slog.Logger }
type metricsOpt struct{ p metrics.Provider }
type shutdownTimeoutOpt struct{ d time.Duration }

// WithLogger returns an option to configure the global logger.
func WithLogger(l *slog.Logger) any {
	return loggerOpt{l: l}
}

// WithMetrics returns an option to configure the global metrics provider.
func WithMetrics(p metrics.Provider) any {
	return metricsOpt{p: p}
}

// WithShutdownTimeout returns an option to configure the diagnostic timeout during shutdown.
// If the application doesn't finish within this duration, it dumps goroutine stacks.
// Default is 2s.
func WithShutdownTimeout(d time.Duration) any {
	return shutdownTimeoutOpt{d: d}
}

// Runnable defines a long-running process that can be started with a context.
type Runnable interface {
	Start(ctx context.Context) error
}

// RunnableFunc is a function adapter that implements Runnable.
type RunnableFunc func(context.Context) error

// Start calls the underlying function.
func (f RunnableFunc) Start(ctx context.Context) error {
	return f(ctx)
}

// Job creates a Runnable from a function.
// It is an alias for RunnableFunc, providing a cleaner API for v1-style CLIs.
func Job(fn func(context.Context) error) Runnable {
	return RunnableFunc(fn)
}

// Run executes the application logic with a managed SignalContext.
// It accepts a Runnable (Job, Router, Supervisor) and manages its lifecycle.
// It accepts generic options (e.g. SignalOption, WithLogger, WithMetrics) to configure the runtime.
// This is the recommended entry point for main().
func Run(r Runnable, opts ...any) error {
	var sigOpts []signal.Option
	diagTimeout := 2 * time.Second

	// Apply configuration options
	for _, opt := range opts {
		switch v := opt.(type) {
		case signal.Option:
			sigOpts = append(sigOpts, v)
		case loggerOpt:
			log.SetLogger(v.l)
		case metricsOpt:
			metrics.SetProvider(v.p)
		case shutdownTimeoutOpt:
			diagTimeout = v.d
		}
	}

	var wg sync.WaitGroup
	sigCtx := signal.NewContext(context.Background(), sigOpts...)
	defer sigCtx.Stop()
	defer sigCtx.Stop()

	// Inject the task tracker into the context passed to the application
	appCtx := WithTaskTracking(sigCtx, &wg)

	err := r.Start(appCtx)

	// Explicitly cancel the context to signal background workers (lifecycle.Go) to stop.
	// This ensures that when the main function exits, the application shuts down cleanly.
	sigCtx.Cancel()

	// If shutdown was triggered by a signal, wait for hooks to complete.
	// We avoid calling Wait() on normal exit or manual stop, as it would block forever.
	reason := sigCtx.Reason()
	if reason == signal.ReasonInterrupt || reason == signal.ReasonTerminate {
		sigCtx.Wait()
	}

	// Wait for all background tasks to finish cleaning up
	waitForTasks(&wg, diagTimeout)

	return err
}

func waitForTasks(wg *sync.WaitGroup, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All good
	case <-time.After(timeout):
		log.Warn("shutdown is taking too long, potential background task leak", "timeout", timeout)
		log.Warn("Dumping goroutine stacks to help diagnose the hang:")

		buf := make([]byte, 1024*1024)
		n := stdruntime.Stack(buf, true)
		_, _ = os.Stderr.Write(buf[:n])

		// Wait indefinitely for tasks to eventually finish
		<-done
	}
}



