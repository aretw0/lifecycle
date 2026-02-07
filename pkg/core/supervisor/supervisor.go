package supervisor

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/introspection"
	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"github.com/aretw0/lifecycle/pkg/core/worker"
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
	stopping      bool                     // In process of shutting down
	children      map[string]worker.Worker // Active workers
	resumeIDs     map[string]string        // Persistent IDs across restarts
	backoffStates map[string]*backoffState // State for exponential backoff
	lastResults   map[string]worker.Status // Final status of finished workers
	stopRequested map[string]bool          // Track workers asked to stop
	eventChan     chan childExit           // Channel for child exit events
	cancel        context.CancelFunc       // To stop the monitor loop
	waitChan      chan error

	// StateWatchers
	stateWatchers []chan introspection.StateChange[worker.State]
	watchersMu    sync.RWMutex
	wg            sync.WaitGroup
	guardsWg      sync.WaitGroup
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
		lastResults:   make(map[string]worker.Status),
		stopRequested: make(map[string]bool),
		eventChan:     make(chan childExit, len(specs)+10), // Buffer to avoid blocking
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

	// Create a context for the monitor loop.
	// We derive it from the startup context to support "Context-Driven Shutdown".
	monitorCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.started = true

	// Initial start of all children
	if err := s.startChildren(ctx, s.specs); err != nil {
		s.stopAll(ctx) // Cleanup whatever started
		s.started = false
		return err
	}

	// Start monitor loop
	s.wg.Add(1)
	go s.monitor(monitorCtx)

	s.emitStateChange(worker.State{Name: s.name, Status: worker.StatusCreated}, s.stateLocked())

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
	defer s.wg.Done()

	// We need to listen to all children's Wait channels.
	// Guards forward exit events to s.eventChan.

	// Spawn guards for initially started children
	s.mu.Lock()
	for name, w := range s.children {
		s.guardsWg.Add(1)
		go s.guard(name, w, s.eventChan)
	}
	s.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			// Supervisor is stopping (Context cancelled or Stop() called)

			// 1. Ensure visual feedback if stopped via context
			s.mu.Lock()
			if !s.stopping {
				oldState := s.stateLocked()
				s.stopping = true
				s.emitStateChange(oldState, s.stateLocked())
			}
			// 2. Wait for all child exit events to be sent while draining channel
			// We must drain because guards might block on eventChan if full.
			// Use a background context with timeout for stopAll to avoid hanging forever
			// if context cancellation failed to stop workers.
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
			stopErr := s.stopAll(stopCtx)
			stopCancel()
			s.mu.Unlock()

			if stopErr != nil {
				log.Warn("error stopping children during context shutdown", "error", stopErr)
			}

			// 2. Wait for all child exit events to be sent while draining channel
			// We must drain because guards might block on eventChan if full.
			guardsDone := make(chan struct{})
			go func() {
				s.guardsWg.Wait()
				close(guardsDone)
			}()

			for {
				select {
				case exit := <-s.eventChan:
					s.handleExit(context.Background(), exit)
				case <-guardsDone:
					// All guards finished, do a final non-blocking drain
					for {
						select {
						case exit := <-s.eventChan:
							s.handleExit(context.Background(), exit)
						default:
							goto finish
						}
					}
				}
			}

		case exit := <-s.eventChan:
			s.handleExit(ctx, exit)
		}
	}

finish:
	s.mu.Lock()
	oldState := s.stateLocked()
	s.started = false
	s.stopping = false
	s.emitStateChange(oldState, s.stateLocked())
	s.mu.Unlock()

	s.waitChan <- nil // Clean exit
	close(s.waitChan)
}

type childExit struct {
	name string
	err  error
}

func (s *supervisor) guard(name string, w worker.Worker, ch chan<- childExit) {
	// Wait for worker to exit
	err := <-w.Wait()
	ch <- childExit{name: name, err: err}
	s.guardsWg.Done()
}

// handleExit processes a child's exit event.
func (s *supervisor) handleExit(ctx context.Context, exit childExit) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find spec
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
		log.Debug("handleExit: spec not found", "name", exit.name)
		return
	}

	// Determine if the exit is "expected" (graceful shutdown or requested stop)
	isExpected := s.stopRequested[exit.name] || s.stopping || (exit.err != nil && errors.Is(exit.err, context.Canceled))

	if isExpected {
		log.Debug("child worker exited", "supervisor", s.name, "child", exit.name, "error", exit.err)
	} else {
		log.Warn("child worker exited", "supervisor", s.name, "child", exit.name, "error", exit.err)
	}

	// Capture old state for emission
	oldState := s.stateLocked()

	// Remove from active children
	delete(s.children, exit.name)

	// Save last status
	// Determine final status
	if exit.err != nil {
		s.lastResults[exit.name] = worker.StatusFailed
	} else if s.stopRequested[exit.name] || s.stopping {
		s.lastResults[exit.name] = worker.StatusStopped
	} else {
		s.lastResults[exit.name] = worker.StatusFinished
	}

	// Remove from stop requested (capture first to check if we should apply strategy)
	wasRequested := s.stopRequested[exit.name]
	delete(s.stopRequested, exit.name)

	// Capture state after child removal for emission
	stateAfterExit := s.stateLocked()

	// Emit transition: Initial -> After Exit
	s.emitStateChange(oldState, stateAfterExit)

	// Apply Strategy ONLY if the exit was NOT requested by the supervisor.
	// If it was requested (e.g. during a StrategyOneForAll mass restart),
	// we don't want to trigger the strategy again, as that causes an infinite loop.
	if !wasRequested && !s.stopping && s.started {
		switch s.strategy {
		case StrategyOneForOne:
			s.handleOneForOne(ctx, exit, failedSpec)
		case StrategyOneForAll:
			s.handleOneForAll(exit)
		}
	}

	// Capture state after strategy (e.g. restart) and emit transition
	s.emitStateChange(stateAfterExit, s.stateLocked())
}

// handleOneForOne handles the restart logic for a single child.
// MUST hold lock.
func (s *supervisor) handleOneForOne(ctx context.Context, exit childExit, failedSpec Spec) {
	if !s.shouldRestart(failedSpec, s.lastResults[exit.name]) {
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

// shouldRestart determines if a child should be restarted based on its exit status.
// MUST hold lock.
func (s *supervisor) shouldRestart(spec Spec, status worker.Status) bool {
	if !s.started || s.stopping {
		return false
	}
	policy := spec.RestartPolicy
	if policy == "" {
		policy = RestartAlways
	}

	switch policy {
	case RestartAlways:
		return true
	case RestartOnFailure:
		return status == worker.StatusFailed
	case RestartNever:
		return false
	default:
		return true
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
			s.guardsWg.Add(1)
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
		s.guardsWg.Add(1)
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
		s.guardsWg.Add(1)
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
	if !s.started || s.stopping {
		s.mu.Unlock()
		return nil
	}

	// 1. Mark as stopping to prevent restarts and show visual feedback
	oldState := s.stateLocked()
	s.stopping = true
	s.emitStateChange(oldState, s.stateLocked())
	s.mu.Unlock()

	// 2. Stop all children
	// stopAll will release/reacquire lock while waiting for children
	s.mu.Lock()
	err := s.stopAll(ctx)
	s.mu.Unlock()

	// 3. Final cleanup
	s.guardsWg.Wait() // Wait for all guards to finish sending events

	s.mu.Lock()
	if s.cancel != nil {
		s.cancel() // Request monitor loop to exit
	}
	s.mu.Unlock()

	s.wg.Wait() // Synchronously wait for monitor (and its event draining) to finish

	s.mu.Lock()
	s.started = false
	s.stopping = false

	s.emitStateChange(worker.State{Name: s.name, Status: worker.StatusStopping}, s.stateLocked())
	s.mu.Unlock()

	return err
}

// stopAll terminates all children in reverse order.
// Assumes caller holds lock.
func (s *supervisor) stopAll(ctx context.Context) error {
	var errs []error

	// Iterate specs in reverse order to respect dependencies (LIFO)
	// We use names instead of raw children map to ensure order
	for i := len(s.specs) - 1; i >= 0; i-- {
		name := s.specs[i].Name
		if child, ok := s.children[name]; ok {
			log.Debug("stopping child", "supervisor", s.name, "child", name)
			s.stopRequested[name] = true

			// Request stop
			if err := child.Stop(ctx); err != nil {
				errs = append(errs, fmt.Errorf("child %s: %w", name, err))
			}

			// Synchronously wait for child to finish (respecting context)
			// IMPORTANT: Release lock while waiting to allow monitor loop to handleExit
			s.mu.Unlock()
			select {
			case <-child.Wait():
				// Clean exit or error already handled by guard/handleExit
			case <-ctx.Done():
				errs = append(errs, fmt.Errorf("child %s: stop timeout: %w", name, ctx.Err()))
			}
			s.mu.Lock()

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
	return s.stateLocked()
}

// ComponentType returns the component type for introspection.
func (s *supervisor) ComponentType() string {
	return "supervisor"
}

// stateLocked returns the snapshot of the supervisor's state.
// MUST hold lock.
func (s *supervisor) stateLocked() worker.State {
	status := worker.StatusRunning
	if s.stopping {
		// Supervisor is stopping. But are all children already stopped?
		// If yes → Stopped. If no → Stopping.
		allTerminal := true
		for _, spec := range s.specs {
			var childStatus worker.Status
			if child, ok := s.children[spec.Name]; ok {
				childStatus = child.State().Status
			} else if result, ok := s.lastResults[spec.Name]; ok {
				childStatus = result
			} else {
				childStatus = worker.StatusCreated
			}

			// Running or Stopping means not yet terminal
			if childStatus == worker.StatusRunning || childStatus == worker.StatusStopping {
				allTerminal = false
				break
			}
		}

		if allTerminal {
			status = worker.StatusStopped
		} else {
			status = worker.StatusStopping
		}
	} else if !s.started {
		// If we haven't started and haven't finished anything, we are Created.
		// Otherwise (after Stop), we are Stopped.
		if len(s.lastResults) == 0 && len(s.children) == 0 {
			status = worker.StatusCreated
		} else {
			status = worker.StatusStopped
		}
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
			// Only override status to Stopping if child is still actively Running
			// If child is already Stopped/Finished/Failed, preserve that terminal state
			if s.stopping && childState.Status == worker.StatusRunning {
				childState.Status = worker.StatusStopping
			}
		} else if result, ok := s.lastResults[spec.Name]; ok {
			// It finished. Should it restart?
			// Only show Pending if the supervisor is actually running.
			// If supervisor is stopped, we treat the result as final for this lifecycle.
			should := s.shouldRestart(spec, result)
			if should && s.started {
				childState.Status = worker.StatusPending
			} else {
				childState.Status = result
			}
		} else if _, ok := s.backoffStates[spec.Name]; ok && s.started {
			childState.Status = worker.StatusPending
		} else {
			childState.Status = worker.StatusCreated
		}

		// Reliability Metadata
		if bs, ok := s.backoffStates[spec.Name]; ok {
			childState.Metadata[worker.MetadataRestarts] = fmt.Sprintf("%d", bs.restarts)
			if !bs.windowStart.IsZero() {
				childState.Metadata[worker.MetadataWindowStart] = bs.windowStart.Format(time.RFC3339)
			}
			// If circuit breaker is triggered, force status to Failed
			if spec.Backoff.MaxRestarts > 0 && bs.restarts > spec.Backoff.MaxRestarts {
				childState.Metadata[worker.MetadataCircuitBreaker] = "triggered"
				childState.Status = worker.StatusFailed
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

// Watch returns a channel for state change events.
func (s *supervisor) Watch(ctx context.Context) <-chan introspection.StateChange[worker.State] {
	ch := make(chan introspection.StateChange[worker.State], 10)

	s.watchersMu.Lock()
	s.stateWatchers = append(s.stateWatchers, ch)
	s.watchersMu.Unlock()

	go func() {
		<-ctx.Done()
		s.watchersMu.Lock()
		defer s.watchersMu.Unlock()

		for i, watcher := range s.stateWatchers {
			if watcher == ch {
				s.stateWatchers = append(s.stateWatchers[:i], s.stateWatchers[i+1:]...)
				break
			}
		}
		close(ch)
	}()

	return ch
}

func (s *supervisor) emitStateChange(old, new worker.State) {
	s.watchersMu.RLock()
	defer s.watchersMu.RUnlock()

	if len(s.stateWatchers) == 0 {
		return
	}

	// Deduplication: Only emit if there's a visual change in the entire tree
	if statesEqual(old, new) {
		return
	}

	change := introspection.StateChange[worker.State]{
		ComponentID:   s.name,
		ComponentType: "supervisor",
		OldState:      old,
		NewState:      new,
		Timestamp:     time.Now(),
	}

	for _, ch := range s.stateWatchers {
		select {
		case ch <- change:
		default:
		}
	}
}

func statesEqual(a, b worker.State) bool {
	if a.Status != b.Status {
		return false
	}
	if len(a.Children) != len(b.Children) {
		return false
	}
	for i := range a.Children {
		if !statesEqual(a.Children[i], b.Children[i]) {
			return false
		}
	}
	return true
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
