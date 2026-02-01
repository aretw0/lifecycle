package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle/pkg/control"
)

// HealthEvent represents a health probe failure or state change.
type HealthEvent struct {
	Status string
	Error  error
}

func (e HealthEvent) String() string {
	if e.Error != nil {
		return fmt.Sprintf("Health(status=%s, err=%v)", e.Status, e.Error)
	}
	return fmt.Sprintf("Health(status=%s)", e.Status)
}

// HealthCheckSource monitors system health and emits events on failure.
// TODO: Integrate with a real probe mechanism.
type HealthCheckSource struct {
	Interval time.Duration
	events   chan control.Event
}

func NewHealthCheckSource(interval time.Duration) *HealthCheckSource {
	return &HealthCheckSource{
		Interval: interval,
		events:   make(chan control.Event),
	}
}

func (s *HealthCheckSource) Events() <-chan control.Event {
	return s.events
}

func (s *HealthCheckSource) Start(ctx context.Context) error {
	defer close(s.events)
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// TODO: Run probe logic here.
		}
	}
}
