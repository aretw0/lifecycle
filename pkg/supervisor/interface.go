package supervisor

import "github.com/aretw0/lifecycle/pkg/worker"

// Supervisor defines the interface for a managed cluster of workers.
// It implements the worker.Worker interface, allowing it to be nested.
type Supervisor interface {
	worker.Worker
}
