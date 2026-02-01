package sources

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/aretw0/lifecycle/pkg/control"
)

// SignalEvent wraps an os.Signal as a control.Event.
type SignalEvent struct {
	Signal os.Signal
}

func (e SignalEvent) String() string {
	return fmt.Sprintf("Signal(%s)", e.Signal.String())
}

// OSSignalSource listens for operating system signals (SIGINT, SIGTERM, etc.).
type OSSignalSource struct {
	signals []os.Signal
	events  chan control.Event
}

// NewOSSignalSource creates a source that listens for the specified signals.
func NewOSSignalSource(signals ...os.Signal) *OSSignalSource {
	return &OSSignalSource{
		signals: signals,
		events:  make(chan control.Event),
	}
}

// Events returns the channel of events.
func (s *OSSignalSource) Events() <-chan control.Event {
	return s.events
}

// Start begins listening for signals and forwarding them to the events channel.
func (s *OSSignalSource) Start(ctx context.Context) error {
	defer close(s.events)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, s.signals...)
	defer signal.Stop(sigChan)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sig := <-sigChan:
			select {
			case s.events <- SignalEvent{Signal: sig}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
