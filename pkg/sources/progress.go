package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle/pkg/control"
)

// TickEvent represents a periodic time tick.
type TickEvent struct {
	Time time.Time
}

func (e TickEvent) String() string {
	return fmt.Sprintf("Tick(%s)", e.Time.Format(time.RFC3339))
}

// TickerSource emits events at a regular interval.
type TickerSource struct {
	interval time.Duration
	events   chan control.Event
}

// NewTickerSource creates a new source that emits tick events.
func NewTickerSource(interval time.Duration) *TickerSource {
	return &TickerSource{
		interval: interval,
		events:   make(chan control.Event),
	}
}

// Events returns the channel of events.
func (s *TickerSource) Events() <-chan control.Event {
	return s.events
}

// Start begins the ticker loop.
func (s *TickerSource) Start(ctx context.Context) error {
	defer close(s.events)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-ticker.C:
			select {
			case s.events <- TickEvent{Time: t}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
