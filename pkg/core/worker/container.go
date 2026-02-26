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
	*BaseWorker
	c container.Container

	// runtime cache for metrics
	image string
}

// NewContainerWorker creates a new Worker for a given container.
func NewContainerWorker(name string, c container.Container) *ContainerWorker {
	return &ContainerWorker{
		BaseWorker: NewBaseWorker(name),
		c:          c,
	}
}

func (cw *ContainerWorker) Start(ctx context.Context) error {
	cw.SetStatus(StatusPending)
	log.Info("starting container worker", "name", cw.String(), "container_id", cw.c.ID())

	// Fetch metadata early if possible
	inspect, err := cw.c.Inspect(ctx)
	if err == nil {
		cw.image = inspect.Image
	}

	if err := cw.c.Start(ctx); err != nil {
		cw.SetStatus(StatusFailed)
		metrics.GetProvider().IncContainerFailed(cw.image)
		log.Error("failed to start container", "name", cw.String(), "error", err)
		return err
	}

	cw.SetStatus(StatusRunning)
	metrics.GetProvider().IncContainerStarted(cw.image)

	// Monitor in background
	go func() {
		for {
			select {
			case <-ctx.Done():
				cw.Finish(ctx.Err())
				return
			default:
				status := cw.c.Status()

				if status == container.StatusStopped {
					cw.Finish(nil)
					return
				}

				if status == container.StatusFailed {
					cw.Finish(fmt.Errorf("container failed"))
					return
				}
				// Poll interval for mock/simple impls
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	return nil
}

func (cw *ContainerWorker) Stop(ctx context.Context) error {
	withLock(cw.BaseWorker, func() { cw.StopRequested = true })

	log.Info("stopping container worker", "name", cw.String(), "container_id", cw.c.ID())
	_ = cw.c.Stop(ctx)

	// Wait for quiescence using Base implementation
	return cw.BaseWorker.Stop(ctx)
}

func (cw *ContainerWorker) String() string {
	return fmt.Sprintf("ContainerWorker(%s, %s)", cw.BaseWorker.String(), cw.c.ID())
}

// State returns the current worker state.
func (cw *ContainerWorker) State() State {
	return cw.ExportState(func(s *State) {
		s.Metadata = make(map[string]string)

		// Use background context for state inspection with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		if inspect, err := cw.c.Inspect(ctx); err == nil {
			s.Metadata["image"] = inspect.Image
			s.Metadata["ip"] = inspect.IP
			s.Metadata["ports"] = fmt.Sprintf("%v", inspect.Ports)
			for k, v := range inspect.Labels {
				s.Metadata["label."+k] = v
			}
		}
		s.Metadata["type"] = "container"
	})
}
