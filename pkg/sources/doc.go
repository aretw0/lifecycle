// Package sources provides standard implementations of control.Source.
//
// It includes:
//   - OSSignalSource: Wraps os/signal for system interruptions implementation.
//   - WebhookSource: (Skeleton) HTTP-triggered events.
//   - HealthCheckSource: (Skeleton) Probe-based failure detection.
//   - FileWatchSource: (Skeleton) Filesystem observation.
package sources
