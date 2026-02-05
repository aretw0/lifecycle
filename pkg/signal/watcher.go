package signal

import (
	"context"
	"time"

	"github.com/aretw0/lifecycle/pkg/introspection"
)

// Watch implements introspection.TypedWatcher[State] interface.
// Returns a type-safe channel that emits StateChange[State] events.
// The channel is closed when the provided context is cancelled.
func (sc *Context) Watch(ctx context.Context) <-chan introspection.StateChange[State] {
	ch := make(chan introspection.StateChange[State], 10)

	sc.watchersMu.Lock()
	sc.stateWatchers = append(sc.stateWatchers, ch)
	sc.watchersMu.Unlock()

	// Cleanup on context cancellation
	go func() {
		<-ctx.Done()
		sc.watchersMu.Lock()
		defer sc.watchersMu.Unlock()

		// Remove from watchers list
		for i, watcher := range sc.stateWatchers {
			if watcher == ch {
				sc.stateWatchers = append(sc.stateWatchers[:i], sc.stateWatchers[i+1:]...)
				break
			}
		}
		close(ch)
	}()

	return ch
}

// emitStateChange broadcasts typed state change to all watchers (non-blocking).
func (sc *Context) emitStateChange(old, new State) {
	sc.watchersMu.RLock()
	defer sc.watchersMu.RUnlock()

	if len(sc.stateWatchers) == 0 {
		return
	}

	// Deduplication: Only emit if runtime status changed
	// Now that we separated Config from Status, we can simply compare the Status struct
	if old.Status == new.Status {
		return
	}

	change := introspection.StateChange[State]{
		ComponentID:   "signal-context",
		ComponentType: "signal",
		OldState:      old,
		NewState:      new,
		Timestamp:     time.Now(),
	}

	for _, ch := range sc.stateWatchers {
		select {
		case ch <- change:
		default:
			// Skip slow consumers (non-blocking)
		}
	}
}
