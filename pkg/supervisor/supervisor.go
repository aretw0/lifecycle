package supervisor

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

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

// Backoff defines the retry policy for failed children.
type Backoff struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	// ResetDuration is the time the child must run successfully to reset the backoff.
	ResetDuration time.Duration
	// MaxRestarts is the maximum number of restarts allowed within MaxDuration.
	// If 0, no limit is enforced.
	MaxRestarts int
	// MaxDuration is the time window for MaxRestarts.
	MaxDuration time.Duration
}

// RestartPolicy defines when a child worker should be restarted.
type RestartPolicy string

const (
	// RestartAlways: Always restart the worker, regardless of exit reason (default).
	RestartAlways RestartPolicy = "Always"
	// RestartOnFailure: Restart only if the worker exits with an error.
	RestartOnFailure RestartPolicy = "OnFailure"
	// RestartNever: Never restart the worker.
	RestartNever RestartPolicy = "Never"
)

// Spec defines the configuration for a supervised child worker.
type Spec struct {
	Name          string
	Type          string // "process", "container", "func" (optional, for diagrams)
	Factory       Factory
	Backoff       Backoff
	RestartPolicy RestartPolicy
}

type backoffState struct {
	currentInterval time.Duration
	lastFailure     time.Time
	lastStart       time.Time
	restarts        int       // Restarts in current window
	windowStart     time.Time // Start of current window
}

// supervisor manages a set of worker processes.
type supervisor struct {
	name     string
	strategy Strategy
	specs    []Spec

	mu            sync.Mutex
	started       bool
	children      map[string]worker.Worker // Active workers
	resumeIDs     map[string]string        // Persistent IDs across restarts
	backoffStates map[string]*backoffState // State for exponential backoff
	eventChan     chan childExit           // Channel for child exit events
	cancel        context.CancelFunc       // To stop the monitor loop
	waitChan      chan error
}

// New creates a new Supervisor.
func New(name string, strategy Strategy, specs ...Spec) Supervisor {
	return &supervisor{
		name:          name,
		strategy:      strategy,
		specs:         specs,
		children:      make(map[string]worker.Worker),
		resumeIDs:     make(map[string]string),
		backoffStates: make(map[string]*backoffState),
		eventChan:     make(chan childExit, 100), // Buffer to prevent blocking guards slightly
		waitChan:      make(chan error, 1),
	}
}

// Start initiates the supervisor and its children.
func (s *supervisor) Start(ctx context.Context) error {
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
func (s *supervisor) startChildren(ctx context.Context, specs []Spec) error {
	for _, spec := range specs {
		if err := s.startChild(ctx, spec); err != nil {
			return err
		}
	}
	return nil
}

// startChild starts a single child from spec.
// MUST hold lock.
func (s *supervisor) startChild(ctx context.Context, spec Spec) error {
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

	// Initialize or update backoff state
	if bs, ok := s.backoffStates[spec.Name]; ok {
		bs.lastStart = time.Now()
	} else {
		s.backoffStates[spec.Name] = &backoffState{
			lastStart: time.Now(),
		}
	}

	metrics.GetProvider().IncWorkerStarted("supervisor_child")
	return nil
}

// monitor runs the main supervision loop.
func (s *supervisor) monitor(ctx context.Context) {
	// We need to listen to all children's Wait channels.
	// Guards forward exit events to s.eventChan.

	// Spawn guards for initially started children
	s.mu.Lock()
	for name, w := range s.children {
		go s.guard(name, w, s.eventChan)
	}
	s.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			// Supervisor is stopping (Stop() called)
			s.waitChan <- nil // Clean exit
			close(s.waitChan)
			return

		case exit := <-s.eventChan:
			s.handleExit(ctx, exit)
		}
	}
}

type childExit struct {
	name string
	err  error
}

func (s *supervisor) guard(name string, w worker.Worker, ch chan<- childExit) {
	// Wait for worker to exit
	err := <-w.Wait()
	ch <- childExit{name: name, err: err}
}

// handleExit processes a child's exit event.
func (s *supervisor) handleExit(ctx context.Context, exit childExit) {
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
		s.handleOneForOne(ctx, exit, failedSpec)
	case StrategyOneForAll:
		s.handleOneForAll(exit)
	}
}

// handleOneForOne handles the restart logic for a single child.
// MUST hold lock.
func (s *supervisor) handleOneForOne(ctx context.Context, exit childExit, failedSpec Spec) {
	// Apply Restart Policy
	shouldRestart := false
	policy := failedSpec.RestartPolicy
	if policy == "" {
		policy = RestartAlways // Default
	}

	switch policy {
	case RestartAlways:
		shouldRestart = true
	case RestartOnFailure:
		if exit.err != nil {
			shouldRestart = true
		} else {
			log.Info("worker finished successfully, not restarting", "child", exit.name)
		}
	case RestartNever:
		log.Info("worker finished, not restarting (policy=Never)", "child", exit.name)
	}

	if !shouldRestart {
		return
	}

	metrics.GetProvider().IncSupervisorRestart(s.name, string(StrategyOneForOne))
	metrics.GetProvider().IncChildRestart(s.name, exit.name)

	// Circuit Breaker Logic
	if failedSpec.Backoff.MaxRestarts > 0 && failedSpec.Backoff.MaxDuration > 0 {
		bs := s.backoffStates[exit.name]
		now := time.Now()

		// If window expired, reset
		if bs.windowStart.IsZero() || now.Sub(bs.windowStart) > failedSpec.Backoff.MaxDuration {
			bs.windowStart = now
			bs.restarts = 1
		} else {
			bs.restarts++
			if bs.restarts > failedSpec.Backoff.MaxRestarts {
				log.Error("circuit breaker triggered: too many restarts",
					"supervisor", s.name,
					"child", exit.name,
					"restarts", bs.restarts,
					"window", failedSpec.Backoff.MaxDuration)
				metrics.GetProvider().IncCircuitBreakerTriggered(exit.name)
				return // Stop restarting
			}
		}
	}

	// Backoff Strategy
	delay := s.nextBackoff(exit.name, failedSpec.Backoff)
	if delay > 0 {
		log.Info("backing off restart", "child", exit.name, "delay", delay)
		// Spawn a goroutine to wait and then restart
		go func() {
			select {
			case <-ctx.Done():
				return // Supervisor stopping
			case <-time.After(delay):
				// Attempt restart with lock
				s.mu.Lock()
				defer s.mu.Unlock()
				s.restartChildLocked(exit.name, failedSpec, exit.err)
			}
		}()
	} else {
		// Immediate restart (lock already held)
		s.restartChildLocked(exit.name, failedSpec, exit.err)
	}
}

// handleOneForAll handles the restart logic for all children upon a single failure.
// MUST hold lock.
func (s *supervisor) handleOneForAll(exit childExit) {
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
			go s.guard(name, w, s.eventChan)
		}
	}
}

// restartChildLocked handles the actual start logic for a single child after backoff.
// MUST hold lock.
func (s *supervisor) restartChildLocked(name string, spec Spec, prevErr error) {
	// Verify supervisor is still running
	if !s.started {
		return
	}

	// Verify spec still exists (dynamic removal)
	exists := false
	for _, s := range s.specs {
		if s.Name == name {
			exists = true
			break
		}
	}
	if !exists {
		return
	}

	restartCtx := context.Background()
	if err := s.startChild(restartCtx, spec); err != nil {
		log.Error("failed to restart child", "child", name, "error", err)
	} else {
		// Handover Protocol: Inject previous exit code
		if injector, ok := s.children[name].(worker.EnvInjector); ok {
			exitCode := "0"
			if prevErr != nil {
				exitCode = "-1"
			}
			injector.SetEnv(worker.EnvPrevExit, exitCode)
		}

		// Re-guard
		go s.guard(name, s.children[name], s.eventChan)
	}
}

// nextBackoff calculates the delay for the next restart.
// MUST hold lock.
func (s *supervisor) nextBackoff(name string, cfg Backoff) time.Duration {
	bs, ok := s.backoffStates[name]
	if !ok {
		return 0
	}

	// Check if we should reset
	if cfg.ResetDuration > 0 && time.Since(bs.lastStart) > cfg.ResetDuration {
		bs.currentInterval = cfg.InitialInterval
		bs.lastFailure = time.Now()
		metrics.GetProvider().IncBackoffTriggered(name, 0)
		return 0
	}

	// Calculate next
	if bs.currentInterval == 0 {
		bs.currentInterval = cfg.InitialInterval
	} else {
		bs.currentInterval = time.Duration(float64(bs.currentInterval) * cfg.Multiplier)
	}

	if cfg.MaxInterval > 0 && bs.currentInterval > cfg.MaxInterval {
		bs.currentInterval = cfg.MaxInterval
	}

	// Add jitter (10%)
	jitter := time.Duration(rand.Int63n(int64(bs.currentInterval/10 + 1)))
	final := bs.currentInterval + jitter
	metrics.GetProvider().IncBackoffTriggered(name, final)
	return final
}

// Add ensures a worker is supervised.
func (s *supervisor) Add(spec Spec) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check duplicates
	for _, child := range s.specs {
		if child.Name == spec.Name {
			return fmt.Errorf("child %s already exists", spec.Name)
		}
	}
	if _, ok := s.children[spec.Name]; ok {
		return fmt.Errorf("child %s already running", spec.Name)
	}

	s.specs = append(s.specs, spec)

	if s.started {
		// Start immediately
		if err := s.startChild(context.Background(), spec); err != nil {
			return err
		}
		// Guard
		go s.guard(spec.Name, s.children[spec.Name], s.eventChan)
	}
	metrics.GetProvider().IncSupervisorAdd(s.name)
	return nil
}

// Remove stops and removes a supervised worker.
func (s *supervisor) Remove(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find spec
	idx := -1
	for i, spec := range s.specs {
		if spec.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("child %s not found", name)
	}

	// Stop child if running
	if child, ok := s.children[name]; ok {
		// Stop with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := child.Stop(ctx); err != nil {
			log.Warn("failed to stop child during remove", "child", name, "error", err)
		}
		delete(s.children, name)
	}

	// Remove from specs
	s.specs = append(s.specs[:idx], s.specs[idx+1:]...)

	// Clean state
	delete(s.backoffStates, name)
	delete(s.resumeIDs, name)

	metrics.GetProvider().IncSupervisorRemove(s.name)
	return nil
}

// Stop stops the supervisor and all its children.
func (s *supervisor) Stop(ctx context.Context) error {
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
func (s *supervisor) stopAll(ctx context.Context) error {
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
func (s *supervisor) Wait() <-chan error {
	return s.waitChan
}

// String returns the supervisor name.
func (s *supervisor) String() string {
	return fmt.Sprintf("Supervisor(%s)", s.name)
}

// State returns the snapshot of the supervisor's state.
func (s *supervisor) State() worker.State {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := worker.StatusRunning
	if !s.started {
		status = worker.StatusStopped
	}

	childrenState := make([]worker.State, 0, len(s.specs))
	for _, spec := range s.specs {
		childState := worker.State{
			Name: spec.Name,
			Metadata: map[string]string{
				worker.MetadataType: spec.Type,
			},
		}

		if child, ok := s.children[spec.Name]; ok {
			st := child.State()
			childState.Status = st.Status
			childState.PID = st.PID
			childState.ExitCode = st.ExitCode
			childState.Error = st.Error
			childState.Children = st.Children // ← Add recursive children!
			// Merge metadata
			for k, v := range st.Metadata {
				childState.Metadata[k] = v
			}
		} else {
			childState.Status = worker.StatusPending
		}

		// Reliability Metadata
		if bs, ok := s.backoffStates[spec.Name]; ok {
			childState.Metadata[worker.MetadataRestarts] = fmt.Sprintf("%d", bs.restarts)
			if !bs.windowStart.IsZero() {
				childState.Metadata[worker.MetadataWindowStart] = bs.windowStart.Format(time.RFC3339)
			}
			if spec.Backoff.MaxRestarts > 0 && bs.restarts > spec.Backoff.MaxRestarts {
				childState.Metadata[worker.MetadataCircuitBreaker] = "triggered"
				childState.Status = worker.StatusFailed // Correct the status from Pending to Failed
			}
		}

		childrenState = append(childrenState, childState)
	}

	return worker.State{
		Name:     s.name,
		Status:   status,
		Children: childrenState,
		Metadata: map[string]string{
			worker.MetadataType: "supervisor",
		},
	}
}

// Suspend pauses all suspendable children.
func (s *supervisor) Suspend(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Info("suspending supervisor", "name", s.name)

	var errs []error
	// Suspend in reverse order (LIFO) to respect dependencies
	for i := len(s.specs) - 1; i >= 0; i-- {
		name := s.specs[i].Name
		if child, ok := s.children[name]; ok {
			if suspendable, ok := child.(worker.Suspendable); ok {
				if err := suspendable.Suspend(ctx); err != nil {
					errs = append(errs, fmt.Errorf("child %s suspend failed: %w", name, err))
				}
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Resume resumes all suspendable children.
func (s *supervisor) Resume(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Info("resuming supervisor", "name", s.name)

	var errs []error
	// Resume in startup order (FIFO)
	for _, spec := range s.specs {
		if child, ok := s.children[spec.Name]; ok {
			if suspendable, ok := child.(worker.Suspendable); ok {
				if err := suspendable.Resume(ctx); err != nil {
					errs = append(errs, fmt.Errorf("child %s resume failed: %w", spec.Name, err))
				}
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
