package events

import (
	"context"
)

// ChannelSource adapts an external Go channel to the Source interface.
// Unlike other sources, it delegates Events() to the external channel
// rather than using BaseSource, since the channel is provided by the caller.
//
// Use this to bridge async systems, testing, or internal event loops
// into the lifecycle Control Plane.
type ChannelSource struct {
	ch <-chan Event
}

// NewChannelSource returns a new source that reads from the given channel.
// The caller retains ownership of the channel and is responsible for closing it.
func NewChannelSource(ch <-chan Event) *ChannelSource {
	return &ChannelSource{ch: ch}
}

// Events returns the external channel for the router to consume.
// This source does NOT use BaseSource because the channel is externally managed.
func (s *ChannelSource) Events() <-chan Event {
	return s.ch
}

// Start blocks until the context is cancelled.
// Since the channel is external, we simply wait for the context to finish.
// The caller is responsible for managing the channel lifecycle.
func (s *ChannelSource) Start(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
