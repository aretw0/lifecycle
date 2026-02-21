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
// until the debounce window expires. Use WithMaxWait to prevent starvation.

// DebounceOption configures the DebounceHandler.
type DebounceOption func(*debounce)

// WithMaxWait sets a maximum duration that events can be delayed before
// a flush is forced. This prevents "starvation" when events arrive continuously
// at an interval shorter than the debounce window.
func WithMaxWait(maxWait time.Duration) DebounceOption {
	return func(d *debounce) {
		d.maxWait = maxWait
	}
}

func DebounceHandler(next Handler, window time.Duration, mergeFunc func(a, b Event) Event, opts ...DebounceOption) Handler {
	if next == nil {
		panic("events: DebounceHandler missing next handler")
	}
	if window <= 0 {
		return next // No debouncing if window is zero or negative
	}

	d := &debounce{
		next:   next,
		window: window,
		merge:  mergeFunc,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

type debounce struct {
	next    Handler
	window  time.Duration
	merge   func(a, b Event) Event
	maxWait time.Duration

	mu       sync.Mutex
	timer    *time.Timer
	buffered Event
	firstHit time.Time
}

func (d *debounce) HandleEvent(ctx context.Context, e Event) error {
	d.mu.Lock()

	// 1. Merge or Replace the buffered event
	if d.buffered == nil || d.merge == nil {
		d.buffered = e // Replace with newest
	} else {
		d.buffered = d.merge(d.buffered, e) // Combine
	}

	// 2. Manage the timer
	var toFlush Event
	if d.timer != nil {
		// Event arrived before window closed
		if d.maxWait > 0 && time.Since(d.firstHit) >= d.maxWait {
			// Starvation prevention: MaxWait exceeded, force a synchronous flush
			d.timer.Stop()
			toFlush = d.buffered
			d.buffered = nil
			d.timer = nil
		} else {
			// Reset the timer for trailing edge
			d.timer.Stop()
			d.timer.Reset(d.window)
		}
	} else {
		// First event of a new burst
		d.firstHit = time.Now()
		d.timer = time.AfterFunc(d.window, func() {
			d.flush(ctx)
		})
	}
	d.mu.Unlock()

	// 3. Flush synchronously outside the lock if MaxWait triggers
	if toFlush != nil && ctx.Err() == nil {
		_ = d.next.HandleEvent(ctx, toFlush)
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
