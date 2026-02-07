package main

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/aretw0/lifecycle"
	"github.com/aretw0/lifecycle/examples/suspend/shared"
)

// Generator produces raw materials using sync.Cond for suspension.
type Generator struct {
	*lifecycle.BaseWorker
	output chan int
	paused bool
	mu     sync.Mutex
	cond   *sync.Cond
	store  *shared.Store
}

func NewGenerator(output chan int, store *shared.Store) *Generator {
	g := &Generator{
		BaseWorker: lifecycle.NewBaseWorker("Generator"),
		output:     output,
		store:      store,
	}
	g.cond = sync.NewCond(&g.mu)
	return g
}

func (g *Generator) Start(ctx context.Context) error {
	return g.StartFunc(ctx, g.Run)
}

func (g *Generator) Run(ctx context.Context) error {
	slog.Info("[GENERATOR] Started.")
	for {
		g.mu.Lock()
		for g.paused {
			slog.Info("[GENERATOR] Paused. Waiting for signal...")
			g.cond.Wait()
			slog.Info("[GENERATOR] Resuming production...")
		}
		g.mu.Unlock()

		select {
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
	g.mu.Lock()
	defer g.mu.Unlock()
	g.paused = true
	return nil
}

func (g *Generator) Resume(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.paused = false
	g.cond.Broadcast()
	return nil
}

// Worker processes materials using sync.Cond for suspension with strict quiescence.
type Worker struct {
	*lifecycle.BaseWorker
	input    <-chan int
	store    *shared.Store
	paused   bool
	pauseReq bool
	mu       sync.Mutex
	cond     *sync.Cond
}

func NewWorker(input <-chan int, store *shared.Store) *Worker {
	w := &Worker{
		BaseWorker: lifecycle.NewBaseWorker("Worker"),
		input:      input,
		store:      store,
	}
	w.cond = sync.NewCond(&w.mu)
	return w
}

func (w *Worker) Start(ctx context.Context) error {
	return w.StartFunc(ctx, w.Run)
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		w.mu.Lock()
		if w.pauseReq {
			slog.Info("[WORKER] Entering quiescence...")
			w.paused = true
			w.cond.Broadcast() // Notify Suspend() that we are paused
		}
		for w.paused {
			w.cond.Wait()
		}
		w.mu.Unlock()

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

func (w *Worker) Suspend(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	slog.Info("[WORKER] Suspend requested. Waiting for quiescence...")
	w.pauseReq = true

	// Block until the worker loop confirms it is paused
	for !w.paused {
		// Note: sync.Cond.Wait() is not context-aware.
		// This is one reason why SuspendGate (channels) is preferred.
		w.cond.Wait()
	}
	slog.Info("[WORKER] Quiescence reached.")
	return nil
}

func (w *Worker) Resume(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	slog.Info("[WORKER] Resuming processing.")
	w.pauseReq = false
	w.paused = false
	w.cond.Broadcast()
	return nil
}

func main() {
	path := filepath.Join("examples", "suspend", "cond", shared.StateFile)
	store := shared.NewStore(path)
	store.Load()

	suspendHandler := lifecycle.NewSuspendHandler()
	sharedChan := make(chan int)

	sup := lifecycle.NewSupervisor("factory-cond", lifecycle.SupervisorStrategyOneForOne,
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
