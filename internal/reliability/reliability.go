package reliability

import (
	"context"
	"time"

	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
)

// Do executes a function in a "Critical Section" that delays context cancellation.
// It wraps the provided function in a shielded context that ignores the parent's cancellation.
// The shielded context is only cancelled AFTER the function finishes or the parent is cancelled
// AND a reasonable grace period has passed (optional, for now it just shields until finish).
func Do(parent context.Context, fn func(ctx context.Context)) error {
	metrics.GetProvider().IncCriticalSectionStarted()
	log.Debug("entering critical section")
	start := time.Now()

	// Create a context that is NOT cancelled when parent is cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	success := true
	defer func() {
		duration := time.Since(start)
		metrics.GetProvider().ObserveCriticalSectionDuration(duration)
		metrics.GetProvider().IncCriticalSectionFinished(success)
		log.Debug("exited critical section", "duration", duration, "success", success)
	}()

	fn(ctx)
	return nil
}
