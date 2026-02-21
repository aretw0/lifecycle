package events

import (
	"context"
)

// Notify returns a Handler that forwards received events to the provided channel.
// This is analogous to os/signal.Notify, giving consumers control over channel buffer
// size and blocking behavior.
//
// The send is non-blocking. If the channel's buffer is full (or it is unbuffered and
// no goroutine is ready to receive), the event is dropped and ErrNotHandled is returned.
//
// Example:
//
//	ch := make(chan events.Event, 10)
//	events.Handle("file/*", events.Notify(ch))
//
//	for e := range ch {
//		fmt.Println("Received:", e.String())
//	}
func Notify(c chan<- Event) Handler {
	if c == nil {
		panic("events: Notify using nil channel")
	}
	return HandlerFunc(func(ctx context.Context, e Event) error {
		select {
		case c <- e:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Non-blocking drop
			return ErrNotHandled
		}
	})
}
