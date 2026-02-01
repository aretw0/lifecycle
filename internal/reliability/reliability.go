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
func Do(parent context.Context, fn func(ctx context.Context) error) error {
	metrics.GetProvider().IncCriticalSectionStarted()
	log.Debug("entering critical section")
	start := time.Now()

	// Create a context that is NOT cancelled when parent is cancelled
	// But PRESERVES values from parent (trace IDs, loggers, etc)
	// We use WithoutCancel (Go 1.21+) to detach cancellation.
	detached := context.WithoutCancel(parent)
	ctx, cancel := context.WithCancel(detached)
	defer cancel()

	var err error
	success := false
	defer func() {
		// Recover first to capture panic
		r := recover()

		// Always record duration and result
		duration := time.Since(start)
		metrics.GetProvider().ObserveCriticalSectionDuration(duration)
		metrics.GetProvider().IncCriticalSectionFinished(success)
		log.Debug("exited critical section", "duration", duration, "success", success, "error", err)

		// Re-panic if needed
		if r != nil {
			panic(r)
		}
	}()

	err = fn(ctx)
	if err == nil {
		success = true
	}
	return err
}
