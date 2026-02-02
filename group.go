package lifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle/pkg/metrics"
	"github.com/aretw0/lifecycle/pkg/runtime"
	"golang.org/x/sync/errgroup"
)

// Group is a wrapper around errgroup.Group that adds lifecycle safety features:
// - Panic Recovery: Captures panics in goroutines and returns them as errors.
// - Observability: Tracks started/finished goroutines via pkg/metrics.
// - Context Propagation: Inherits context from the parent.
type Group struct {
	g   *errgroup.Group
	ctx context.Context
}

// NewGroup creates a new Group derived from the given context.
// It acts like errgroup.WithContext.
func NewGroup(ctx context.Context) (*Group, context.Context) {
	g, ctx := errgroup.WithContext(ctx)
	return &Group{g: g, ctx: ctx}, ctx
}

// SetLimit limits the number of active goroutines in this group to at most n.
// A negative value indicates no limit.
func (g *Group) SetLimit(n int) {
	g.g.SetLimit(n)
}

// Go calls the given function in a new goroutine.
// It blocks until the new goroutine can be added without the number of
// active goroutines in the group exceeding the configured limit.
//
// If the function triggers a panic, it is recovered, logged, and returned as an error,
// which will cancel the Group's context.
func (g *Group) Go(fn func(ctx context.Context) error) {
	// We wrap the function to handle panics and metrics.
	// We also verify backpressure by measuring how long it takes to schedule.
	start := time.Now()

	// Signal that we are attempting to schedule (entering wait queue)
	metrics.GetProvider().IncGoroutineWaiting()

	// We need to track Goroutine semantics explicitly for Group,
	// because runtime.Do tracks "Critical Operations" which is slightly different.
	metrics.GetProvider().IncGoroutineStarted()
	defer metrics.GetProvider().IncGoroutineFinished()

	g.g.Go(func() (err error) {
		metrics.GetProvider().DecGoroutineWaiting()

		defer func() {
			if r := recover(); r != nil {
				// runtime.Do() already logged the panic.
				// We recover it here to return it as an error to the errgroup,
				// which will cancel the context.
				metrics.GetProvider().IncGoroutinePanicked()
				err = fmt.Errorf("panic in lifecycle.Group: %v", r)
			}
		}()

		// Use runtime.Do for execution safety
		return runtime.Do(g.ctx, fn)
	})

	// Measure backpressure (Time spent blocked by SetLimit)
	// Note: errgroup.Go returns when the goroutine is scheduled.
	if d := time.Since(start); d > 100*time.Microsecond {
		metrics.GetProvider().ObserveGoroutineBlockDuration(d)
	}
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
func (g *Group) Wait() error {
	return g.g.Wait()
}
