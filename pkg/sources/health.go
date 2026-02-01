package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle/pkg/control"
)

// HealthEvent represents a health probe status.
type HealthEvent struct {
	Name   string
	Status string // "UP", "DOWN", etc
	Error  error
}

func (e HealthEvent) String() string {
	if e.Status == "DOWN" {
		return fmt.Sprintf("health/%s/down", e.Name)
	}
	return fmt.Sprintf("health/%s/up", e.Name)
}

// CheckFunc is a function that checks the health of a component.
type CheckFunc func(ctx context.Context) error

// TriggerStrategy defines when events are emitted.
type TriggerStrategy string

const (
	// TriggerEdge emits events only when the status changes.
	TriggerEdge TriggerStrategy = "EDGE"
	// TriggerLevel emits events on every check interval (Heartbeat).
	TriggerLevel TriggerStrategy = "LEVEL"
)

// HealthCheckSource runs a periodic health check.
type HealthCheckSource struct {
	Name     string
	Interval time.Duration
	Check    CheckFunc
	Strategy TriggerStrategy
	events   chan control.Event
}

// NewHealthCheckSource creates a new health monitor.
// TODO: Functional options pattern
func NewHealthCheckSource(name string, interval time.Duration, check CheckFunc) *HealthCheckSource {
	return &HealthCheckSource{
		Name:     name,
		Interval: interval,
		Check:    check,
		Strategy: TriggerEdge, // Default
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

	var lastStatus string

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			err := s.Check(ctx)
			// TODO: Use constants for status
			status := "UP"
			if err != nil {
				status = "DOWN"
			}

			// Determine if we should emit
			shouldEmit := false
			if s.Strategy == TriggerLevel {
				shouldEmit = true
			} else {
				// Edge Trigger
				if status != lastStatus {
					shouldEmit = true
				}
			}

			if shouldEmit {
				lastStatus = status
				s.emit(ctx, HealthEvent{Name: s.Name, Status: status, Error: err})
			}
		}
	}
}

func (s *HealthCheckSource) emit(ctx context.Context, e HealthEvent) {
	select {
	case s.events <- e:
	case <-ctx.Done():
	}
}
