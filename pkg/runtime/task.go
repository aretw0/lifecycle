package runtime

import (
	"context"
	"fmt"
	"sync"
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
func Go(ctx context.Context, fn func(context.Context) error) {
	wg, ok := ctx.Value(taskTrackerKey{}).(*sync.WaitGroup)
	if !ok {
		// Fallback to global tracker if not managed by Run()
		// This ensures we still track/wait, but the user must call WaitForGlobal manually
		// if they want to wait for these specific detached tasks.
		wg = &defaultTracker
	}

	wg.Add(1)

	go func() {
		defer wg.Done()

		// Top-level recovery for the background task.
		// logic.Do re-panics to allow bubbling, so we MUST catch it here
		// to prevent the application from crashing.
		defer func() {
			if r := recover(); r != nil {
				// We log specifically for background tasks, as they have no caller to return error to.
				fmt.Printf("lifecycle: background task panic: %v\n", r)
			}
		}()

		// Use Do for observability and panic capturing (metrics).
		// We ignore the error return here as it's an async background task.
		_ = Do(ctx, fn)
	}()
}
