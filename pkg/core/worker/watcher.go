package worker

import (
	"context"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/introspection"
)

// Watch implements introspection.TypedWatcher[State] interface.
// Returns a type-safe channel that emits StateChange[State] events.
// The channel is closed when the provided context is cancelled.
func (b *BaseWorker) Watch(ctx context.Context) <-chan introspection.StateChange[State] {
	ch := make(chan introspection.StateChange[State], 10) // Buffered to avoid blocking

	b.watchersMu.Lock()
	b.stateWatchers = append(b.stateWatchers, ch)
	b.watchersMu.Unlock()

	// Cleanup on context cancellation
	go func() {
		<-ctx.Done()

		b.watchersMu.Lock()
		defer b.watchersMu.Unlock()

		// Remove this watcher
		for i, watcher := range b.stateWatchers {
			if watcher == ch {
				b.stateWatchers = append(b.stateWatchers[:i], b.stateWatchers[i+1:]...)
				break
			}
		}
		close(ch)
	}()

	return ch
}

// emitStateChange broadcasts a type-safe state change to all watchers (non-blocking).
func (b *BaseWorker) emitStateChange(old, new State) {
	change := introspection.StateChange[State]{
		ComponentID:   b.name,
		ComponentType: "worker",
		OldState:      old,
		NewState:      new,
		Timestamp:     time.Now(),
	}

	b.watchersMu.RLock()
	watchers := make([]chan introspection.StateChange[State], len(b.stateWatchers))
	copy(watchers, b.stateWatchers)
	b.watchersMu.RUnlock()

	// Non-blocking send to avoid slow consumers blocking the worker
	for _, ch := range watchers {
		select {
		case ch <- change:
		default:
			// Skip if watcher is slow
		}
	}
}



