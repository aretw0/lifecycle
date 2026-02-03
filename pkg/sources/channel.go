package sources

import (
	"context"

	"github.com/aretw0/lifecycle/pkg/control"
)

// ChannelSource adapts a Go channel to the Source interface.
// It is useful for bridging async systems, testing, or internal event loops
// into the lifecycle Control Plane.
type ChannelSource struct {
	Ch <-chan control.Event
}

// NewChannelSource returns a new source that reads from the given channel.
func NewChannelSource(ch <-chan control.Event) *ChannelSource {
	return &ChannelSource{Ch: ch}
}

// Events returns the read-only channel for the router to consume.
func (s *ChannelSource) Events() <-chan control.Event {
	return s.Ch
}

// Start blocks until the context is cancelled.
// Since the channel is external, we simply wait for the context to finish.
func (s *ChannelSource) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
