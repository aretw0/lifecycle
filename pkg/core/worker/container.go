package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/container"
	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
)

// ContainerWorker is a Worker that manages a container via the container.Container interface.
type ContainerWorker struct {
	c    container.Container
	name string

	// runtime cache for metrics
	image string
}

// NewContainerWorker creates a new Worker for a given container.
func NewContainerWorker(name string, c container.Container) *ContainerWorker {
	return &ContainerWorker{
		c:    c,
		name: name,
	}
}

func (cw *ContainerWorker) Start(ctx context.Context) error {
	log.Info("starting container worker", "name", cw.name, "container_id", cw.c.ID())

	// Fetch metadata early if possible
	inspect, err := cw.c.Inspect(ctx)
	if err == nil {
		cw.image = inspect.Image
	}

	if err := cw.c.Start(ctx); err != nil {
		metrics.GetProvider().IncContainerFailed(cw.image)
		log.Error("failed to start container", "name", cw.name, "error", err)
		return err
	}

	metrics.GetProvider().IncContainerStarted(cw.image)
	return nil
}

func (cw *ContainerWorker) Stop(ctx context.Context) error {
	log.Info("stopping container worker", "name", cw.name, "container_id", cw.c.ID())
	start := time.Now()

	err := cw.c.Stop(ctx)
	if err != nil {
		log.Warn("container stop returned error", "name", cw.name, "error", err)
	}

	duration := time.Since(start)
	metrics.GetProvider().ObserveContainerDuration(cw.image, duration)
	metrics.GetProvider().IncContainerStopped(cw.image)

	return err
}

func (cw *ContainerWorker) Wait() <-chan error {
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
				metrics.GetProvider().IncContainerFailed(cw.image)
				err := fmt.Errorf("container failed")
				log.Error("container worker failure", "name", cw.name, "container_id", cw.c.ID())
				ch <- err
				close(ch)
				return
			}
			// Poll interval for mock/simple impls
			time.Sleep(100 * time.Millisecond)
		}
	}()
	return ch
}

func (cw *ContainerWorker) String() string {
	return fmt.Sprintf("ContainerWorker(%s, %s)", cw.name, cw.c.ID())
}

func (cw *ContainerWorker) State() State {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	status := StatusCreated
	switch cw.c.Status() {
	case container.StatusPending:
		status = StatusPending
	case container.StatusRunning:
		status = StatusRunning
	case container.StatusStopped:
		status = StatusStopped
	case container.StatusFailed:
		status = StatusFailed
	}

	metadata := make(map[string]string)
	if inspect, err := cw.c.Inspect(ctx); err == nil {
		metadata["image"] = inspect.Image
		metadata["ip"] = inspect.IP
		metadata["ports"] = fmt.Sprintf("%v", inspect.Ports)
		for k, v := range inspect.Labels {
			metadata["label."+k] = v
		}
	}
	metadata["type"] = "container"

	return State{
		Name:     cw.name,
		Status:   status,
		Metadata: metadata,
	}
}



