package events

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

// SignalEvent represents an OS signal.
type SignalEvent struct {
	Signal os.Signal
}

func (e SignalEvent) String() string {
	// Standardize on lowercase for matching
	return fmt.Sprintf("Signal(%s)", e.Signal)
}

// SignalSource listens for OS signals and emits them as Events.
type SignalSource struct {
	BaseSource
	signals []os.Signal
}

// NewSignalSource creates a source that listens for the given signals.
func NewSignalSource(sigs ...os.Signal) *SignalSource {
	return &SignalSource{
		BaseSource: NewBaseSource("signal", 10),
		signals:    sigs,
	}
}

// NewOSSignalSource is an alias for NewSignalSource for backward compatibility.
func NewOSSignalSource(sigs ...os.Signal) *SignalSource {
	return NewSignalSource(sigs...)
}

func (s *SignalSource) Start(ctx context.Context) error {
	defer s.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, s.signals...)
	defer signal.Stop(sigChan)

	for {
		select {
		case <-ctx.Done():
			return nil
		case sig := <-sigChan:
			event := SignalEvent{Signal: sig}
			if err := s.Emit(ctx, event); err != nil {
				return err
			}
		}
	}
}
