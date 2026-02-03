// Package sources provides standard implementations of control.Source.
//
// It includes:
//   - OSSignalSource: Wraps os/signal for system interruptions implementation.
//   - InputSource: Reads commands from Stdin (robust against Ctrl+C).
//   - ChannelSource: Bridges internal Go channels to the Router.
//   - WebhookSource: (Skeleton) HTTP-triggered events.
//   - HealthCheckSource: (Skeleton) Probe-based failure detection.
//   - FileWatchSource: (Skeleton) Filesystem observation.
package sources
