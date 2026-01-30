package worker

import (
	"context"
	"fmt"

	"github.com/aretw0/lifecycle/pkg/container"
)

// ContainerWorker is a Worker that manages a container via the container.Container interface.
type ContainerWorker struct {
	c    container.Container
	name string
}

// NewContainerWorker creates a new Worker for a given container.
func NewContainerWorker(name string, c container.Container) *ContainerWorker {
	return &ContainerWorker{
		c:    c,
		name: name,
	}
}

func (cw *ContainerWorker) Start(ctx context.Context) error {
	return cw.c.Start(ctx)
}

func (cw *ContainerWorker) Stop(ctx context.Context) error {
	return cw.c.Stop(ctx)
}

func (cw *ContainerWorker) Wait() <-chan error {
	// For containers, we might need a background poller or a block on Stop.
	// In this reference implementation, we return a closed channel if Stopped.
	ch := make(chan error, 1)
	go func() {
		for {
			status := cw.c.Status()
			if status == container.StatusStopped {
				ch <- nil
				close(ch)
				return
			}
			if status == container.StatusFailed {
				ch <- fmt.Errorf("container failed")
				close(ch)
				return
			}
			// Busy wait for mock? In a real impl, this would be a long-poll to Docker.
		}
	}()
	return ch
}

func (cw *ContainerWorker) String() string {
	return fmt.Sprintf("ContainerWorker(%s, %s)", cw.name, cw.c.ID())
}

func (cw *ContainerWorker) State() State {
	status := StatusPending
	switch cw.c.Status() {
	case container.StatusRunning:
		status = StatusRunning
	case container.StatusStopped:
		status = StatusStopped
	case container.StatusFailed:
		status = StatusFailed
	}

	return State{
		Name:   cw.name,
		Status: status,
	}
}
