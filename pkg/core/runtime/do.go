package runtime

import (
	"context"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
)

// Do executes a function with Managed Reliability (Panic Recovery + Observability).
// Unlike the deprecated internal/reliability.Do, this function RESPECTS context cancellation.
// For a "Critical Section" that survives cancellation, use DoDetached.
func Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	metrics.GetProvider().IncCriticalSectionStarted()
	log.Debug("starting managed operation")
	start := time.Now()

	success := false
	defer func() {
		// Recover first to capture panic
		r := recover()

		// Always record duration and result
		duration := time.Since(start)
		metrics.GetProvider().ObserveCriticalSectionDuration(duration)
		metrics.GetProvider().IncCriticalSectionFinished(success)
		log.Debug("managed operation finished", "duration", duration, "success", success, "error", err)

		// Re-panic if needed (after logging/metrics)
		if r != nil {
			// We panic with the same value to allow higher-level recovery if needed,
			// though usually Do is the top-level safeguard.
			panic(r)
		}
	}()

	err = fn(ctx)
	if err == nil {
		success = true
	}
	return err
}

// DoDetached executes a function in a "Critical Section" that delays context cancellation.
// It wraps the provided function in a shielded context that ignores the parent's cancellation.
// (Uses context.WithoutCancel available in Go 1.21+)
func DoDetached(parent context.Context, fn func(ctx context.Context) error) error {
	// Create a context that is NOT cancelled when parent is cancelled
	// But PRESERVES values from parent (trace IDs, loggers, etc)
	detached := context.WithoutCancel(parent)

	// Ensure isolation: The detached context is valid ONLY for the duration of this call.
	// This prevents leaks if the user passes this context to children that wait on Done().
	ctx, cancel := context.WithCancel(detached)
	defer cancel()

	return Do(ctx, fn)
}
