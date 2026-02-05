package main

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/aretw0/lifecycle"
	"github.com/aretw0/lifecycle/examples/suspend/shared"
	"github.com/aretw0/lifecycle/pkg/core/worker"
)

// Generator produces raw materials using SuspendGate for simplified suspension.
type Generator struct {
	lifecycle.BaseWorker
	output chan int
	store  *shared.Store
	gate   *worker.SuspendGate
}

func NewGenerator(output chan int, store *shared.Store) *Generator {
	return &Generator{
		BaseWorker: lifecycle.NewBaseWorker("Generator"),
		output:     output,
		store:      store,
		gate:       worker.NewSuspendGate(),
	}
}

func (g *Generator) Start(ctx context.Context) error {
	return g.StartFunc(ctx, g.Run)
}

func (g *Generator) Run(ctx context.Context) error {
	slog.Info("[GENERATOR] Started.")
	for {
		// Use the Gate to handle suspension boilerplate
		if err := g.gate.Check(ctx); err != nil {
			return err
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

func (g *Generator) Suspend(ctx context.Context) error { return g.gate.Suspend(ctx) }
func (g *Generator) Resume(ctx context.Context) error  { g.gate.Resume(); return nil }

// Worker processes materials using SuspendGate for simplified suspension.
type Worker struct {
	lifecycle.BaseWorker
	input chan int
	store *shared.Store
	gate  *worker.SuspendGate
}

func NewWorker(input chan int, store *shared.Store) *Worker {
	return &Worker{
		BaseWorker: lifecycle.NewBaseWorker("Worker"),
		input:      input,
		store:      store,
		gate:       worker.NewSuspendGate(),
	}
}

func (w *Worker) Start(ctx context.Context) error {
	return w.StartFunc(ctx, w.Run)
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		if err := w.gate.Check(ctx); err != nil {
			return err
		}

		select {
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

func (w *Worker) Suspend(ctx context.Context) error { return w.gate.Suspend(ctx) }
func (w *Worker) Resume(ctx context.Context) error  { w.gate.Resume(); return nil }

func main() {
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



