package runtime

import (
	"context"
	"sync"
	"time"

	"github.com/aretw0/lifecycle/pkg/signal"
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
// This is the recommended entry point for main().
func Run(r Runnable, opts ...signal.Option) error {
	var wg sync.WaitGroup
	sigCtx := signal.NewContext(context.Background(), opts...)
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
	wg.Wait()

	return err
}
