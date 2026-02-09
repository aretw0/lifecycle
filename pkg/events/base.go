package events

import (
	"context"
	"sync"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/metrics"
)

// BaseSource provides default implementation for Source.Events() method.
// Embed this in your source types to avoid repeating the events channel boilerplate.
//
// Example:
//
//	type MySource struct {
//	    BaseSource
//	    // custom fields...
//	}
//
//	func NewMySource() *MySource {
//	    return &MySource{
//	        BaseSource: NewBaseSource("my-source", 10), // name and buffer size
//	    }
//	}
//
//	func (s *MySource) Start(ctx context.Context) error {
//	    for {
//	        event := // ... create event
//	        if err := s.Emit(ctx, event); err != nil {
//	            return err
//	        }
//	    }
//	}
//
// The embedding provides Events() implementation automatically.
type BaseSource struct {
	name   string
	events chan Event
}

// NewBaseSource creates a BaseSource with the specified name and buffer size.
// A buffer of 10-100 is recommended for most sources to prevent blocking.
func NewBaseSource(name string, bufferSize int) BaseSource {
	return BaseSource{
		name:   name,
		events: make(chan Event, bufferSize),
	}
}

// Events returns the read-only events channel.
// This method is automatically available via embedding.
func (b *BaseSource) Events() <-chan Event {
	return b.events
}

// Emit sends an event to the events channel.
// This is a helper method for source implementations.
// It blocks if the channel buffer is full, providing backpressure.
// It returns an error if the context is cancelled while waiting.
func (b *BaseSource) Emit(ctx context.Context, e Event) error {
	start := time.Now()

	select {
	case b.events <- e:
		// Success: If we were blocked, record the duration
		if d := time.Since(start); d > 100*time.Microsecond {
			metrics.GetProvider().ObserveEventBlockDuration(b.name, d)
		}
		metrics.GetProvider().IncEventEmitted(b.name)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Buffer is full: enter waiting state
		metrics.GetProvider().IncEventWaiting(b.name)
		defer metrics.GetProvider().DecEventWaiting(b.name)

		select {
		case b.events <- e:
			duration := time.Since(start)
			metrics.GetProvider().ObserveEventBlockDuration(b.name, duration)
			metrics.GetProvider().IncEventEmitted(b.name)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// TryEmit attempts to send an event without blocking.
// Returns true if the event was sent, false if the buffer was full.
func (b *BaseSource) TryEmit(e Event) bool {
	select {
	case b.events <- e:
		metrics.GetProvider().IncEventEmitted(b.name)
		return true
	default:
		return false
	}
}

// Close closes the events channel.
// Call this when the source is done emitting
func (b *BaseSource) Close() {
	close(b.events)
}

// Once wraps a handler to ensure it only executes its logic exactly once.
// This is useful for shutdown or cleanup handlers that involve closing channels
// or other non-idempotent operations.
//
// Example:
//
//	Handle("command/quit", Once(HandlerFunc(func(ctx context.Context, _ Event) error {
//	    close(quitCh)
//	    return nil
//	})))
func Once(h Handler) Handler {
	var once sync.Once
	return HandlerFunc(func(ctx context.Context, e Event) error {
		var err error
		once.Do(func() {
			err = h.HandleEvent(ctx, e)
		})
		return err
	})
}
