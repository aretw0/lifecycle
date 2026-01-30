package worker

import "context"

// Worker defines the interface for a managed unit of work (process, goroutine, container).
//
// The lifecycle is:
//  1. Start(ctx) -> Non-blocking.
//  2. Wait()     -> Returns channel that closes when work finishes.
//  3. Stop(ctx)  -> Graceful termination request.
type Worker interface {
	// Start initiates the worker. It must be non-blocking.
	// The context can be used to control the startup phase.
	Start(context.Context) error

	// Stop requests the worker to stop.
	// It should respect the provided context for timeout/cancellation of the stop request.
	Stop(context.Context) error

	// Wait returns a channel that is closed when the worker has exited.
	// The error associated with the exit (if any) is sent on the channel.
	Wait() <-chan error

	// String returns a human-readable description/ID of the worker.
	String() string
}
