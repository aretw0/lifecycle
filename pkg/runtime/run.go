package runtime

import (
	"context"
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

// Run executes the application logic with a managed SignalContext.
// It handles context creation, signal monitoring, and proper cleanup (Stop).
// It also ensures we Wait() for hooks if a signal triggered the shutdown.
// This is the recommended entry point for main().
func Run(fn func(context.Context) error, opts ...signal.Option) error {
	ctx := signal.NewContext(context.Background(), opts...)
	defer ctx.Stop()

	err := fn(ctx)

	// If shutdown was triggered by a signal, wait for hooks to complete.
	// We avoid calling Wait() on normal exit or manual stop, as it would block forever.
	reason := ctx.Reason()
	if reason == signal.ReasonInterrupt || reason == signal.ReasonTerminate {
		ctx.Wait()
	}

	return err
}
