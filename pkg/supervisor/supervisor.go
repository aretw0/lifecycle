package supervisor

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
	"github.com/aretw0/lifecycle/pkg/worker"
	"github.com/google/uuid"
)

// Strategy defines how the supervisor handles child failures.
type Strategy string

const (
	// StrategyOneForOne: If a child process terminates, only that process is restarted.
	StrategyOneForOne Strategy = "OneForOne"
	// StrategyOneForAll: If a child process terminates, all other child processes are terminated,
	// and then all child processes are restarted.
	StrategyOneForAll Strategy = "OneForAll"
)

// Factory is a function that creates a new worker instance.
type Factory func() (worker.Worker, error)

// Spec defines the configuration for a supervised child worker.
type Spec struct {
	Name    string
	Factory Factory
}

// Supervisor manages a set of worker processes.
type Supervisor struct {
	name     string
	strategy Strategy
	specs    []Spec

	mu        sync.Mutex
	started   bool
	children  map[string]worker.Worker // Active workers
	resumeIDs map[string]string        // Persistent IDs across restarts
	cancel    context.CancelFunc       // To stop the monitor loop
	waitChan  chan error
}

// New creates a new Supervisor.
func New(name string, strategy Strategy, specs ...Spec) *Supervisor {
	return &Supervisor{
		name:      name,
		strategy:  strategy,
		specs:     specs,
		children:  make(map[string]worker.Worker),
		resumeIDs: make(map[string]string),
		waitChan:  make(chan error, 1),
	}
}

// Start initiates the supervisor and its children.
func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("supervisor %s already started", s.name)
	}

	log.Info("starting supervisor", "name", s.name, "strategy", s.strategy, "children", len(s.specs))

	// Create a detached context for the monitor loop to ensure it runs independently
	// of the startup context, but can be cancelled by Stop().
	monitorCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.started = true

	// Initial start of all children
	if err := s.startChildren(ctx, s.specs); err != nil {
		s.stopAll(ctx) // Cleanup whatever started
		s.started = false
		return err
	}

	// Start monitor loop
	go s.monitor(monitorCtx)

	return nil
}

// startChildren starts a list of specs.
// MUST hold lock.
func (s *Supervisor) startChildren(ctx context.Context, specs []Spec) error {
	for _, spec := range specs {
		if err := s.startChild(ctx, spec); err != nil {
			return err
		}
	}
	return nil
}

// startChild starts a single child from spec.
// MUST hold lock.
func (s *Supervisor) startChild(ctx context.Context, spec Spec) error {
	w, err := spec.Factory()
	if err != nil {
		return fmt.Errorf("failed to create worker %s: %w", spec.Name, err)
	}

	if err := w.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker %s: %w", spec.Name, err)
	}

	// Handover Protocol: Initialize or retrieve Resume ID
	resumeID, ok := s.resumeIDs[spec.Name]
	if !ok {
		resumeID = uuid.New().String()
		s.resumeIDs[spec.Name] = resumeID
	}

	// Inject Resume ID if possible
	if injector, ok := w.(worker.EnvInjector); ok {
		injector.SetEnv(worker.EnvResumeID, resumeID)
	}

	s.children[spec.Name] = w
	metrics.GetProvider().IncWorkerStarted("supervisor_child")
	return nil
}

// monitor runs the main supervision loop.
func (s *Supervisor) monitor(ctx context.Context) {
	// We need to listen to all children's Wait channels.
	// A simple way is to spawn a "guard" goroutine for each child that forwards exit events.

	eventChan := make(chan childExit, len(s.specs)*2)

	// Spawn guards for initially started children
	s.mu.Lock()
	for name, w := range s.children {
		go s.guard(name, w, eventChan)
	}
	s.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			// Supervisor is stopping (Stop() called)
			s.waitChan <- nil // Clean exit
			close(s.waitChan)
			return

		case exit := <-eventChan:
			s.handleExit(ctx, exit, eventChan)
		}
	}
}

type childExit struct {
	name string
	err  error
}

func (s *Supervisor) guard(name string, w worker.Worker, ch chan<- childExit) {
	// Wait for worker to exit
	err := <-w.Wait()
	ch <- childExit{name: name, err: err}
}

// handleExit processes a child's exit event.
func (s *Supervisor) handleExit(ctx context.Context, exit childExit, eventChan chan<- childExit) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If supervisor is stopping, ignore exits
	if ctx.Err() != nil {
		return
	}

	log.Warn("child worker exited", "supervisor", s.name, "child", exit.name, "error", exit.err)

	// Determine spec for this child
	var failedSpec Spec
	found := false
	for _, spec := range s.specs {
		if spec.Name == exit.name {
			failedSpec = spec
			found = true
			break
		}
	}
	if !found {
		return
	}

	// Remove from active children
	delete(s.children, exit.name)

	// Apply Strategy
	switch s.strategy {
	case StrategyOneForOne:
		metrics.GetProvider().IncSupervisorRestart(s.name, string(StrategyOneForOne))
		metrics.GetProvider().IncChildRestart(s.name, exit.name)

		// Restart only this child
		// Using Background context as restart is a fresh lifecycle event.
		// Future: Support restart backoff and context timeouts.
		restartCtx := context.Background()
		if err := s.startChild(restartCtx, failedSpec); err != nil {
			log.Error("failed to restart child", "child", exit.name, "error", err)
		} else {
			// Handover Protocol: Inject previous exit code if possible
			if injector, ok := s.children[exit.name].(worker.EnvInjector); ok {
				exitCode := "0"
				if exit.err != nil {
					// Simplified: assume -1 if error.
					// In a more complex impl, we'd extract the actual code from exit.err if it's an ExitError
					exitCode = "-1"
				}
				injector.SetEnv(worker.EnvPrevExit, exitCode)
			}

			// Re-guard
			go s.guard(exit.name, s.children[exit.name], eventChan)
		}

	case StrategyOneForAll:
		metrics.GetProvider().IncSupervisorRestart(s.name, string(StrategyOneForAll))
		// Stop all other children
		log.Info("Restarting all children due to failure", "trigger", exit.name)
		s.stopAll(context.Background()) // Synchronously stop others

		// Restart all
		if err := s.startChildren(context.Background(), s.specs); err != nil {
			log.Error("failed to restart all children", "error", err)
		} else {
			// Re-guard all
			for name, w := range s.children {
				go s.guard(name, w, eventChan)
			}
		}
	}
}

// Stop stops the supervisor and all its children.
func (s *Supervisor) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	// 1. Stop the monitor loop first to prevent restarts during shutdown
	if s.cancel != nil {
		s.cancel()
	}
	s.started = false

	// 2. Stop all children
	return s.stopAll(ctx)
}

// stopAll terminates all children in reverse order.
// Assumes caller holds lock.
func (s *Supervisor) stopAll(ctx context.Context) error {
	var errs []error

	// Iterate specs in reverse order to respect dependencies (LIFO)
	for i := len(s.specs) - 1; i >= 0; i-- {
		name := s.specs[i].Name
		if child, ok := s.children[name]; ok {
			log.Debug("stopping child", "supervisor", s.name, "child", name)
			if err := child.Stop(ctx); err != nil {
				errs = append(errs, fmt.Errorf("child %s: %w", name, err))
			}
			delete(s.children, name)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Wait returns the channel to wait for supervisor exit.
func (s *Supervisor) Wait() <-chan error {
	return s.waitChan
}

// String returns the supervisor name.
func (s *Supervisor) String() string {
	return fmt.Sprintf("Supervisor(%s)", s.name)
}

// State returns the snapshot of the supervisor's state.
func (s *Supervisor) State() worker.State {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := worker.StatusRunning
	if !s.started {
		status = worker.StatusStopped
	}

	// We iterate over specs to preserve order, or children map?
	// Specs order is better for visualization.
	childrenState := make([]worker.State, 0, len(s.specs))

	for _, spec := range s.specs {
		if child, ok := s.children[spec.Name]; ok {
			childrenState = append(childrenState, child.State())
		} else {
			// Report missing children as Pending to report the full configuration scope,
			// even if the process is temporarily down or failed.
			childrenState = append(childrenState, worker.State{
				Name:   spec.Name,
				Status: worker.StatusPending,
			})
		}
	}

	return worker.State{
		Name:     s.name,
		Status:   status,
		Children: childrenState,
	}
}
