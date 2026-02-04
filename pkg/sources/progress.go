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
	control.BaseSource
	interval time.Duration
}

// NewTickerSource creates a new source that emits tick events.
func NewTickerSource(interval time.Duration) *TickerSource {
	return &TickerSource{
		BaseSource: control.NewBaseSource("ticker", 10),
		interval:   interval,
	}
}

// Start begins the ticker loop.
func (s *TickerSource) Start(ctx context.Context) error {
	defer s.Close()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-ticker.C:
			if err := s.Emit(ctx, TickEvent{Time: t}); err != nil {
				return err
			}
		}
	}
}
