package introspection

import (
	"context"
	"reflect"
	"sync"
	"time"
)

// AggregateWatchers combines multiple typed watchers into a unified snapshot stream.
func AggregateWatchers(ctx context.Context, watchers ...interface{}) <-chan StateSnapshot {
	out := make(chan StateSnapshot, 64)
	var wg sync.WaitGroup

	for _, w := range watchers {
		compType := inferComponentType(w)
		if compType == "" {
			continue
		}

		v := reflect.ValueOf(w)
		watchMethod := v.MethodByName("Watch")
		if !watchMethod.IsValid() {
			continue
		}

		results := watchMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
		if len(results) == 0 {
			continue
		}

		ch := results[0]
		wg.Add(1)
		go func(componentType string, changeChan reflect.Value) {
			defer wg.Done()

			for {
				val, ok := changeChan.Recv()
				if !ok {
					return
				}

				snapshot := StateSnapshot{
					ComponentID:   val.FieldByName("ComponentID").String(),
					ComponentType: componentType,
					Timestamp:     val.FieldByName("Timestamp").Interface().(time.Time),
					Payload:       val.FieldByName("NewState").Interface(),
				}

				select {
				case out <- snapshot:
				case <-ctx.Done():
					return
				}
			}
		}(compType, ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// inferComponentType determines the component type from a watcher using reflection.
func inferComponentType(watcher interface{}) string {
	if watcher == nil {
		return ""
	}

	// Check if it implements Component interface
	if c, ok := watcher.(Component); ok {
		return c.ComponentType()
	}

	return ""
}

// AggregateEvents combines multiple event sources into a unified event stream.
func AggregateEvents(ctx context.Context, sources ...EventSource) <-chan ComponentEvent {
	out := make(chan ComponentEvent, 64)
	var wg sync.WaitGroup

	for _, src := range sources {
		wg.Add(1)
		go func(source EventSource) {
			defer wg.Done()
			for event := range source.Events(ctx) {
				select {
				case out <- event:
				case <-ctx.Done():
					return
				}
			}
		}(src)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
