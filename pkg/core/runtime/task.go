package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"

	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"github.com/aretw0/lifecycle/pkg/core/observe"
)

type taskTrackerKey struct{}

// WithTaskTracking returns a context that tracks goroutines using the provided WaitGroup.
// This is used by Run to inject a wait group into the context.
func WithTaskTracking(ctx context.Context, wg *sync.WaitGroup) context.Context {
	return context.WithValue(ctx, taskTrackerKey{}, wg)
}

var (
	defaultTracker sync.WaitGroup
	// ensure defaultTracker is ready to use (sync.WaitGroup is 0 value safe)
)

// WaitForGlobal waits for all goroutines started with Go() that used the fallback global tracker.
// This is useful if you are using Go() outside of Run().
func WaitForGlobal() {
	defaultTracker.Wait()
}

// Go starts a goroutine that is tracked by the lifecycle.
// If the context contains a TaskTracker (injected by Run), it adds to that WaitGroup.
// If not, it falls back to a global TaskTracker (accessible via WaitForGlobal), ensuring "Safe by Default".
// It also recovers from panics to prevent crashing the entire application.
//
// The returned Task handle allows waiting for completion and checking errors.
// By default, errors are discarded unless an ErrorHandler is provided via WithErrorHandler.
func Go(ctx context.Context, fn func(context.Context) error, opts ...GoOption) Task {
	cfg := &goConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	wg, ok := ctx.Value(taskTrackerKey{}).(*sync.WaitGroup)
	if !ok {
		// Fallback to global tracker if not managed by Run()
		// This ensures we still track/wait, but the user must call WaitForGlobal manually
		// if they want to wait for these specific detached tasks.
		wg = &defaultTracker
	}

	wg.Add(1)
	metrics.GetProvider().IncGoroutineStarted()

	handle := &taskHandle{
		done: make(chan struct{}),
	}

	go func() {
		defer wg.Done()
		defer metrics.GetProvider().IncGoroutineFinished()
		defer close(handle.done)

		// Top-level recovery for the background task.
		// logic.Do re-panics to allow bubbling, so we MUST catch it here
		// to prevent the application from crashing.
		defer func() {
			if r := recover(); r != nil {
				// We log specifically for background tasks, as they have no caller to return error to.
				log.Error("background task panic", "recover", r)
				metrics.GetProvider().IncGoroutinePanicked()

				// Stack capture: Three-mode behavior (Stable as of v1.6.0)
				//   1. WithStackCapture(true):  Always capture stack bytes
				//   2. WithStackCapture(false): Never capture stack bytes
				//   3. Unset (default):         Auto-detect via slog.LevelDebug; capture only in debug mode
				// This reduces production overhead (no stack capture in prod unless explicitly needed)
				// while enabling detailed diagnostics in development.
				// See docs/LIMITATIONS.md for performance impact and behavior details.
				logger := log.GetLogger()
				debugEnabled := logger.Enabled(context.Background(), slog.LevelDebug)
				capture := debugEnabled
				if cfg.stackCapture != nil {
					capture = *cfg.stackCapture
				}

				var stack []byte
				if capture {
					stack = debug.Stack()
				}
				if obs := observe.GetObserver(); obs != nil {
					obs.OnGoroutinePanicked(r, stack)
				}
				if capture && debugEnabled {
					log.Debug("panic stack trace", "stack", string(stack))
				}

				// Capture panic as error
				handle.err = fmt.Errorf("panic: %v", r)
				if cfg.errorHandler != nil {
					cfg.errorHandler(handle.err)
				}
			}
		}()

		// Use Do for observability and panic capturing (metrics).
		// We ignore the error return here as it's an async background task.
		err := Do(ctx, fn)
		handle.err = err

		if err != nil && cfg.errorHandler != nil {
			cfg.errorHandler(err)
		}
	}()

	return handle
}
