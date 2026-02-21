package events

import (
	"context"
	"sync"
	"time"
)

// DebounceHandler wraps a Handler, buffering events that occur within a specified
// time window and emitting only a single aggregated event after the window passes
// without new events.
//
// If mergeFunc is nil, all intermediate events are dropped and only the MOST
// RECENT event is emitted. This is the idiomatic behavior for file-system debouncing
// where you only care about "the file is stable now".
//
// If you need to aggregate data (e.g. collecting multiple changed file paths into a batch),
// provide a custom mergeFunc that combines the previously buffered event (a)
// with the new arrival (b).
//
// Note: This spawns a background goroutine per unique event series to manage the timer
// until the debounce window expires.
func DebounceHandler(next Handler, window time.Duration, mergeFunc func(a, b Event) Event) Handler {
	if next == nil {
		panic("events: DebounceHandler missing next handler")
	}
	if window <= 0 {
		return next // No debouncing if window is zero or negative
	}

	return &debounce{
		next:   next,
		window: window,
		merge:  mergeFunc,
	}
}

type debounce struct {
	next   Handler
	window time.Duration
	merge  func(a, b Event) Event

	mu       sync.Mutex
	timer    *time.Timer
	buffered Event
}

func (d *debounce) HandleEvent(ctx context.Context, e Event) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 1. Merge or Replace the buffered event
	if d.buffered == nil || d.merge == nil {
		d.buffered = e // Replace with newest
	} else {
		d.buffered = d.merge(d.buffered, e) // Combine
	}

	// 2. Manage the timer
	if d.timer != nil {
		// Event arrived before window closed; reset the timer
		d.timer.Stop()
		d.timer.Reset(d.window)
	} else {
		// First event of a new burst; start the timer
		// We spawn a lightweight goroutine managed via core/runtime package helpers
		// or directly via time.AfterFunc to ensure it doesn't block the caller.

		// time.AfterFunc executes the callback in its own goroutine
		d.timer = time.AfterFunc(d.window, func() {
			d.flush(ctx)
		})
	}

	return nil
}

// flush extracts the buffered event and sends it to the next handler.
func (d *debounce) flush(ctx context.Context) {
	d.mu.Lock()
	e := d.buffered
	d.buffered = nil
	d.timer = nil
	d.mu.Unlock()

	if e != nil {
		// Note: The context passed from the original event might be cancelled
		// by the time the timer fires. Thus, we should check it.
		if ctx.Err() == nil {
			_ = d.next.HandleEvent(ctx, e)
		}
	}
}
