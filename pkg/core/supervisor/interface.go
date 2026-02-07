package supervisor

import (
	"context"

	"github.com/aretw0/lifecycle/pkg/core/introspection"
	"github.com/aretw0/lifecycle/pkg/core/worker"
)

// Supervisor defines the interface for a managed cluster of workers.
// It implements the worker.Worker interface, allowing it to be nested.
type Supervisor interface {
	worker.Worker

	// Add executes a new worker under the supervisor.
	Add(Spec) error

	// Remove terminates and removes a worker from the supervisor.
	Remove(name string) error

	// Suspend pauses all suspendable children.
	Suspend(context.Context) error

	// Resume resumes all suspendable children.
	Resume(context.Context) error

	// Watch returns a channel that emits state changes.
	Watch(context.Context) <-chan introspection.StateChange[worker.State]
}
