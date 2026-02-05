package introspection

import "context"

// WatcherAdapter converts typed state changes to snapshots for aggregation.
// This allows TypedWatcher[S] instances to participate in cross-domain aggregation.
type WatcherAdapter[S any] struct {
	componentType string
	watcher       TypedWatcher[S]
}

// NewWatcherAdapter creates an adapter for the given typed watcher.
func NewWatcherAdapter[S any](componentType string, w TypedWatcher[S]) *WatcherAdapter[S] {
	return &WatcherAdapter[S]{
		componentType: componentType,
		watcher:       w,
	}
}

// Snapshots converts the typed state change stream into snapshot envelopes.
func (a *WatcherAdapter[S]) Snapshots(ctx context.Context) <-chan StateSnapshot {
	ch := make(chan StateSnapshot, 10)

	go func() {
		defer close(ch)

		for change := range a.watcher.Watch(ctx) {
			select {
			case ch <- StateSnapshot{
				ComponentID:   change.ComponentID,
				ComponentType: a.componentType,
				Timestamp:     change.Timestamp,
				Payload:       change.NewState,
			}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}
