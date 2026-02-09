package container

import (
	"context"
	"io"
)

// Status represents the lifecycle state of a container.
type Status string

const (
	StatusCreated Status = "Created"
	StatusPending Status = "Pending"
	StatusRunning Status = "Running"
	StatusStopped Status = "Stopped"
	StatusFailed  Status = "Failed"
)

// InspectData contains detailed runtime information about a container.
type InspectData struct {
	Image  string
	IP     string
	Ports  []string
	Labels map[string]string
}

// Container defines a generic interface for managing containerized workloads.
// This decouples the lifecycle supervisor from specific SDKs like Docker or Podman.
type Container interface {
	// Start initiates the container.
	Start(ctx context.Context) error

	// Stop requests the container to stop gracefully.
	Stop(ctx context.Context) error

	// Inspect returns detailed runtime information.
	Inspect(ctx context.Context) (InspectData, error)

	// Logs returns a reader for the container's logs (stdout/stderr).
	Logs(ctx context.Context) (io.ReadCloser, error)

	// ID returns the unique identifier of the container.
	ID() string

	// Status returns the current lifecycle status.
	Status() Status
}
