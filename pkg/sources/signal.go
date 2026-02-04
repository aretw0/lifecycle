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
	control.BaseSource
	signals []os.Signal
}

// NewOSSignalSource creates a source that listens for the specified signals.
func NewOSSignalSource(signals ...os.Signal) *OSSignalSource {
	return &OSSignalSource{
		BaseSource: control.NewBaseSource("ossignal", 10),
		signals:    signals,
	}
}

// Start begins listening for signals and forwarding them to the events channel.
func (s *OSSignalSource) Start(ctx context.Context) error {
	defer s.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, s.signals...)
	defer signal.Stop(sigChan)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sig := <-sigChan:
			if err := s.Emit(ctx, SignalEvent{Signal: sig}); err != nil {
				return err
			}
		}
	}
}
