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

// It also recovers from panics to prevent crashing the entire application.
// Go starts a goroutine that is tracked by the lifecycle.
// If the context contains a TaskTracker (injected by Run), it adds to the WaitGroup.
// It uses Do() internally, providing metrics and panic recovery for the background task.
func Go(ctx context.Context, fn func(context.Context) error) {
	wg, ok := ctx.Value(taskTrackerKey{}).(*sync.WaitGroup)
	if ok {
		wg.Add(1)
	}

	go func() {
		if ok {
			defer wg.Done()
		}

		// Use Do for observability and panic recovery.
		// We ignore the error return here as it's an async background task.
		// (Errors should be logged inside fn or by the metrics in Do)
		_ = Do(ctx, func(c context.Context) error {
			// Check for panic in Do? No, Do re-panics.
			// So we need another recovery here?
			// Actually, Do re-panics. If we want Go to be safe ("prevent crashing application"),
			// we must recover *outside* Do.
			// But Do logs the panic.

			// Let's modify Do to NOT re-panic? Or handle it here.
			// "It also recovers from panics to prevent crashing the entire application."
			// If Do re-panics, we crash.

			// Wait, reliability.Do re-panicked.
			// If reliability.Do re-panics, then it wasn't strictly "Crash Prevention" for the app,
			// unless called inside another recover block.

			// Let's wrap Do in a safe block.
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("lifecycle: background task panic: %v\n", r)
				}
			}()

			return fn(c)
		})
	}()
}
