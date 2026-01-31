package worker

import (
	"context"
	"strings"
)

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

	// State returns the current state of the worker for introspection.
	State() State
}

// Status represents the lifecycle state of a worker.
type Status string

const (
	StatusPending Status = "Pending"
	StatusRunning Status = "Running"
	StatusStopped Status = "Stopped"
	StatusFailed  Status = "Failed"
)

// Key returns the normalized lowercase representation of the status.
func (s Status) Key() string {
	return strings.ToLower(string(s))
}

// Type represents the kind of worker.
type Type string

const (
	TypeProcess    Type = "process"
	TypeContainer  Type = "container"
	TypeFunc       Type = "func"
	TypeSupervisor Type = "supervisor"
	TypeGoroutine  Type = "goroutine"
)

// String returns the capitalized representation of the type for logs.
func (t Type) String() string {
	switch t {
	case TypeProcess:
		return "Process"
	case TypeContainer:
		return "Container"
	case TypeFunc:
		return "Func"
	case TypeSupervisor:
		return "Supervisor"
	case TypeGoroutine:
		return "Goroutine"
	default:
		return string(t)
	}
}

// State represents a snapshot of the worker's status.
type State struct {
	Name     string
	Status   Status
	PID      int
	ExitCode int
	Error    error
	Metadata map[string]string
	Children []State
}
