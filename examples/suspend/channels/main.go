package main

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/aretw0/lifecycle"
	"github.com/aretw0/lifecycle/examples/suspend/shared"
)

// Generator produces raw materials using channels for suspension (v2.x Style).
type Generator struct {
	lifecycle.BaseWorker
	output chan int
	store  *shared.Store

	suspend chan struct{}
	resume  chan struct{}
	paused  chan struct{}
}

func NewGenerator(output chan int, store *shared.Store) *Generator {
	return &Generator{
		BaseWorker: lifecycle.NewBaseWorker("Generator"),
		output:     output,
		store:      store,
		suspend:    make(chan struct{}),
		resume:     make(chan struct{}),
		paused:     make(chan struct{}),
	}
}

func (g *Generator) Start(ctx context.Context) error {
	return g.StartFunc(ctx, g.Run)
}

func (g *Generator) Run(ctx context.Context) error {
	slog.Info("[GENERATOR] Started.")
	for {
		// Check for suspension
		select {
		case <-g.suspend:
			slog.Info("[GENERATOR] Entering quiescence...")
			select {
			case g.paused <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			}

			slog.Info("[GENERATOR] Paused. Waiting for resume signal...")
			select {
			case <-g.resume:
				slog.Info("[GENERATOR] Resuming production...")
			case <-ctx.Done():
				return ctx.Err()
			}
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		g.store.Mu.Lock()
		if g.store.State.ItemsProduced >= shared.TargetGoal {
			g.store.Mu.Unlock()
			return nil
		}
		g.store.State.ItemsProduced++
		item := g.store.State.ItemsProduced
		g.store.Mu.Unlock()

		slog.Info("[GENERATOR] Produced item", "id", item)

		select {
		case g.output <- item:
		case <-ctx.Done():
			return ctx.Err()
		}

		time.Sleep(2000 * time.Millisecond)
	}
}

func (g *Generator) Suspend(ctx context.Context) error {
	select {
	case g.suspend <- struct{}{}:
		// Block until confirmed
		select {
		case <-g.paused:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *Generator) Resume(ctx context.Context) error {
	select {
	case g.resume <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// Worker processes materials using channels for suspension.
type Worker struct {
	lifecycle.BaseWorker
	input chan int
	store *shared.Store

	suspend chan struct{}
	resume  chan struct{}
	paused  chan struct{}
}

func NewWorker(input chan int, store *shared.Store) *Worker {
	return &Worker{
		BaseWorker: lifecycle.NewBaseWorker("Worker"),
		input:      input,
		store:      store,
		suspend:    make(chan struct{}),
		resume:     make(chan struct{}),
		paused:     make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) error {
	return w.StartFunc(ctx, w.Run)
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		select {
		case <-w.suspend:
			slog.Info("[WORKER] Entering quiescence...")
			select {
			case w.paused <- struct{}{}:
				// Signal Suspend() that we are now paused
			case <-ctx.Done():
				return ctx.Err()
			}

			select {
			case <-w.resume:
				slog.Info("[WORKER] Resuming...")
			case <-ctx.Done():
				return ctx.Err()
			}

		case item, ok := <-w.input:
			if !ok {
				return nil
			}
			slog.Info("[WORKER] Processing item...", "id", item)
			time.Sleep(1500 * time.Millisecond)

			w.store.Mu.Lock()
			w.store.State.ItemsProcessed++
			count := w.store.State.ItemsProcessed
			w.store.Mu.Unlock()

			slog.Info("[WORKER] Finished item", "id", item)
			if count >= shared.TargetGoal {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (w *Worker) Suspend(ctx context.Context) error {
	select {
	case w.suspend <- struct{}{}:
		// Wait for confirmation
		select {
		case <-w.paused:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) Resume(ctx context.Context) error {
	select {
	case w.resume <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func main() {
	// Set path relative to this main.go
	path := filepath.Join("examples", "suspend", "channels", shared.StateFile)
	store := shared.NewStore(path)
	store.Load()

	suspendHandler := lifecycle.NewSuspendHandler()
	sharedChan := make(chan int)

	sup := lifecycle.NewSupervisor("factory-channels", lifecycle.SupervisorStrategyOneForOne,
		lifecycle.SupervisorSpec{
			Name: "watchdog",
			Factory: func() (lifecycle.Worker, error) {
				return shared.NewWatchdog(), nil
			},
			RestartPolicy: lifecycle.RestartAlways,
		},
		lifecycle.SupervisorSpec{
			Name: "generator",
			Factory: func() (lifecycle.Worker, error) {
				return NewGenerator(sharedChan, store), nil
			},
			RestartPolicy: lifecycle.RestartOnFailure,
		},
		lifecycle.SupervisorSpec{
			Name: "worker",
			Factory: func() (lifecycle.Worker, error) {
				return NewWorker(sharedChan, store), nil
			},
			RestartPolicy: lifecycle.RestartOnFailure,
		},
		lifecycle.SupervisorSpec{
			Name: "blocker",
			Factory: func() (lifecycle.Worker, error) {
				return shared.NewBlocker(), nil
			},
			RestartPolicy: lifecycle.RestartAlways,
		},
	)

	shared.RunFactory(sup, store, suspendHandler)
}
