package runtime

import (
	"context"
	"iter"
)

// Receive creates a push iterator that yields values from the channel until
// the context is cancelled or the channel is closed.
// It eliminates the need for manual select loops when consuming channels,
// preventing deadlocks and "orphaned receiver" bugs.
//
// Usage:
//
//	for msg := range runtime.Receive(ctx, ch) {
//	    process(msg)
//	}
func Receive[V any](ctx context.Context, ch <-chan V) iter.Seq[V] {
	return func(yield func(V) bool) {
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-ch:
				if !ok {
					return
				}
				if !yield(v) {
					return
				}
			}
		}
	}
}
