package lifecycle

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
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

	g.g.Go(func() (err error) {
		// We are running, so we are no longer waiting.
		metrics.GetProvider().DecGoroutineWaiting()
		metrics.GetProvider().IncGoroutineStarted()

		defer metrics.GetProvider().IncGoroutineFinished()

		defer func() {

			if r := recover(); r != nil {
				metrics.GetProvider().IncGoroutinePanicked()
				stack := string(debug.Stack())
				log.Error("lifecycle.Group: panic recovered", "panic", r)
				// fmt.Printf("DEBUG STACK:\n%s\n", stack) // Optional: dump stack to stdout if logs are swallowed
				err = fmt.Errorf("panic in lifecycle.Group: %v\nstack: %s", r, stack)
			}
		}()

		return fn(g.ctx)
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
