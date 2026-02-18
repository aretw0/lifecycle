package signal

import (
	"context"
	"os"
	ossignal "os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aretw0/introspection"
	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
)

type contextKey struct{}

var signalContextKey = contextKey{}

// FromContext retrieves the signal.Context from a possibly wrapped context.
func FromContext(ctx context.Context) (*Context, bool) {
	if sc, ok := ctx.Value(signalContextKey).(*Context); ok {
		return sc, true
	}
	// Fallback to type assertion if not using the key yet
	sc, ok := ctx.(*Context)
	return sc, ok
}

// Reason describes why the context was cancelled.
type Reason string

const (
	ReasonNone         Reason = "None"
	ReasonInterrupt    Reason = "Signal:Interrupt" // SIGINT (Ctrl+C)
	ReasonTerminate    Reason = "Signal:Terminate" // SIGTERM
	ReasonManualStop   Reason = "Manual:Stop"      // Explicit Stop() call
	ReasonManualCancel Reason = "Manual:Cancel"    // Context Cancel() called
	ReasonTimeout      Reason = "System:Timeout"   // Shutdown timed out
)

func (r Reason) String() string {
	return string(r)
}

// Context wraps a context and captures the signal that cancelled it.
type Context struct {
	context.Context
	cancel      func()
	start       sync.Once
	stop        sync.Once
	sigCh       chan os.Signal
	sigVal      os.Signal
	reason      Reason
	mu          sync.Mutex
	opts        options
	hooks       []func()
	hooksDone   chan struct{}
	signalCount int
	lastSignal  time.Time

	// StateWatchers (Event-Driven Introspection)
	stateWatchers []chan introspection.StateChange[State]
	watchersMu    sync.RWMutex
	stopped       bool
	hooksOnce     sync.Once
}

// Cancel terminates the context manually.
func (sc *Context) Cancel() {
	oldState := sc.State()

	sc.mu.Lock()
	if sc.reason == ReasonNone {
		sc.reason = ReasonManualCancel
	}
	sc.mu.Unlock()
	sc.cancel()

	newState := sc.State()
	sc.emitStateChange(oldState, newState)

	go sc.runHooks()
}

// Shutdown is an alias for Cancel for consistency with other components.
func (sc *Context) Shutdown() {
	sc.Cancel()
}

// ShutdownWait initiates a graceful shutdown and blocks until all hooks have completed.
// It is a shorthand for Shutdown() followed by Wait().
func (sc *Context) ShutdownWait() {
	sc.Shutdown()
	sc.Wait()
}

// ComponentType returns the component type for introspection.
func (sc *Context) ComponentType() string {
	return "signal"
}

// ResetSignalCount resets the signal counter and clears the last received signal.
// This is useful for "Smart Handlers" that successfully handle a signal (e.g. Suspend)
// and want to reset the "Force Exit" threshold.
func (sc *Context) ResetSignalCount() {
	oldState := sc.State()

	sc.mu.Lock()
	sc.signalCount = 0
	sc.sigVal = nil
	sc.mu.Unlock()

	log.Debug("signal count reset")

	newState := sc.State()
	sc.emitStateChange(oldState, newState)
}

// OnShutdown registers a function to be called when the context receives a shutdown signal.
// Hooks are executed in LIFO (Last-In-First-Out) order, simulating `defer`.
// Execution happens asynchronously after the context is cancelled.
// Call Wait() to block until all hooks (including those registered dynamically) have finished.
func (sc *Context) OnShutdown(f func()) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.hooks = append(sc.hooks, f)
}

type options struct {
	forceExitThreshold int
	resetTimeout       time.Duration
	hookTimeout        time.Duration
	cancelOnInterrupt  bool
}

// Option is a functional option for configuring signal behavior.
type Option func(*options)

// WithForceExit configures the threshold of signals required to trigger an immediate os.Exit(1).
// Threshold values:
// 1 (Default): SIGINT cancels context immediately. SIGTERM always cancels.
// n >= 2: SIGINT is captured (signalCount increments), os.Exit(1) at n-th signal.
// 0 (Unsafe): Automatic os.Exit(1) is disabled for SIGINT. SIGTERM still cancels context.
func WithForceExit(threshold int) Option {
	return func(o *options) {
		o.forceExitThreshold = threshold
	}
}

// WithResetTimeout configures the duration after which the signal count resets.
// If a signal is received after this duration from the previous one, it is treated as a new sequence.
// Default is 5 seconds.
func WithResetTimeout(d time.Duration) Option {
	return func(o *options) {
		o.resetTimeout = d
	}
}

// WithHookTimeout configures the duration after which a running hook produces a warning log.
// Default is 5 seconds.
func WithHookTimeout(d time.Duration) Option {
	return func(o *options) {
		o.hookTimeout = d
	}
}

// WithCancelOnInterrupt controls whether SIGINT cancels the context.
//
// true (default): SIGINT immediately cancels context (traditional behavior)
//   - Use for: CLIs, servers, batch jobs
//   - SIGINT #1: Context cancelled (graceful shutdown)
//   - SIGINT #2+: Force exit based on threshold
//
// false: SIGINT only emits events, does NOT cancel context
//   - Use for: REPLs, interactive shells, suspendable processes
//   - SIGINT #1-N: Emits control events (ClearLine, Suspend, etc.)
//   - SIGINT #N: Force exit when threshold reached
//   - User must explicitly cancel context via router/handler
//
// Note: SIGTERM always cancels context regardless of this setting.
func WithCancelOnInterrupt(enabled bool) Option {
	return func(o *options) {
		o.cancelOnInterrupt = enabled
	}
}

// Stop stops the signal monitoring and restores default behavior.
// It also ensures Wait() unblocks if called.
func (sc *Context) Stop() {
	sc.stop.Do(func() {
		sc.mu.Lock()
		if sc.reason == ReasonNone {
			sc.reason = ReasonManualStop
		}
		sc.stopped = true
		sc.mu.Unlock()
		// Restore default signal handling
		ossignal.Stop(sc.sigCh)
		close(sc.sigCh)
	})
}

// Wait blocks until all registered hooks have completed execution.
// This is essential to prevent the main function from exiting before cleanup is done.
func (sc *Context) Wait() {
	<-sc.hooksDone
}

// NewContext creates a context that is cancelled on SIGTERM or SIGINT (standard termination).
// On the first signal received, it cancels the context to initiate graceful shutdown (if configured).
// If a second signal is received before the program exits, it performs an immediate os.Exit(1) (if configured).
func NewContext(parent context.Context, opts ...Option) *Context {
	o := options{
		forceExitThreshold: 1, // Default: Ctrl+C cancels context immediately
		resetTimeout:       5 * time.Second,
		hookTimeout:        5 * time.Second,
		cancelOnInterrupt:  true, // Default: SIGINT cancels context (backward compatible)
	}
	for _, opt := range opts {
		opt(&o)
	}

	ctx, cancel := context.WithCancel(parent)
	sc := &Context{
		sigCh:     make(chan os.Signal, 1),
		hooksDone: make(chan struct{}),
		reason:    ReasonNone,
		opts:      o,
		cancel:    cancel,
	}

	// Double-bind: the context contains the sc, and the sc wraps the context.
	sc.Context = context.WithValue(ctx, signalContextKey, sc)

	sc.start.Do(func() {
		ossignal.Notify(sc.sigCh, os.Interrupt, syscall.SIGTERM)
		go sc.monitor()
	})

	return sc
}

// monitor runs the signal monitoring loop.
func (sc *Context) monitor() {
	doneCh := sc.Context.Done()
	for {
		select {
		case sig, ok := <-sc.sigCh:
			if !ok {
				return
			}
			sc.mu.Lock()
			sc.signalCount++
			count := sc.signalCount
			sc.mu.Unlock()

			sc.handleSignal(sig, count)

		case <-doneCh:
			// Turn off the Done case to prevent busy loop,
			// but continue monitoring signals for Force Exit logic.
			doneCh = nil
		}
	}
}

// handleSignal processes a single signal.
func (sc *Context) handleSignal(sig os.Signal, count int) {
	sc.mu.Lock()
	isFirst := sc.sigVal == nil

	// Reset logic
	if !sc.lastSignal.IsZero() && time.Since(sc.lastSignal) > sc.opts.resetTimeout {
		sc.signalCount = 1
		count = 1
		log.Debug("signal count reset due to timeout")
	}
	sc.lastSignal = time.Now()
	sc.sigVal = sig
	sc.mu.Unlock()

	log.Debug("received signal", "signal", sig.String(), "count", count, "first", isFirst)
	metrics.GetProvider().IncSignalReceived(sig.String())

	// Determine cancellation reason early for introspection accuracy
	if sc.shouldCancel(sig) {
		sc.mu.Lock()
		if sc.reason == ReasonNone {
			switch sig {
			case os.Interrupt:
				sc.reason = ReasonInterrupt
			case syscall.SIGTERM:
				sc.reason = ReasonTerminate
			}
		}
		sc.mu.Unlock()
	}

	// Cancellation Execution
	// We cancel on the very first compatible signal if cancellation is allowed.
	if isFirst && sc.shouldCancel(sig) {
		sc.Cancel()
		go sc.runHooks()
		return
	}

	// Emit state change for signal reception (only if not cancelling)
	oldState := sc.State()
	oldState.SignalCount-- // Restore old count for diff
	newState := sc.State()
	sc.emitStateChange(oldState, newState)

	// Force Exit Logic (Escalation)
	if sc.opts.forceExitThreshold > 0 {
		threshold := sc.opts.forceExitThreshold
		// Interactive Offset:
		// If cancellation on interrupt is disabled, we assume the signals are
		// delegated to the application (e.g., Suspend -> Quit).
		// We add a +2 buffer to ensure both the primary and fallback software
		// handlers get a chance to execute before the hardware kill switch fires.
		if !sc.opts.cancelOnInterrupt {
			threshold += 2
		}

		if count >= threshold {
			log.Warn("force exit threshold reached, exiting immediately",
				"signal", sig.String(),
				"count", count)
			metrics.GetProvider().IncForceExitTriggered()
			os.Exit(1)
		}
	}
}

func (sc *Context) shouldCancel(sig os.Signal) bool {
	if sig == syscall.SIGTERM {
		return true
	}
	// SIGINT only cancels if explicitly enabled
	// This allows interactive applications (REPL, Suspend) to handle SIGINT without context cancellation
	return sig == os.Interrupt && sc.opts.cancelOnInterrupt
}

// Signal returns the signal that caused the context to be cancelled/interrupted, or nil.
func (sc *Context) Signal() os.Signal {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.sigVal
}

// Reason returns the reason why the context was cancelled.
func (sc *Context) Reason() Reason {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.reason
}

// IsUnsafe returns true if the context is configured to never force exit (threshold 0).
func (sc *Context) IsUnsafe() bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.opts.forceExitThreshold == 0
}

// ForceExitThreshold returns the number of signals required to trigger os.Exit(1).
func (sc *Context) ForceExitThreshold() int {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.opts.forceExitThreshold
}

// State represents the configuration state of the SignalContext.
// Config represents immutable signal handler configuration.
// These values are set at initialization and never change during runtime.
type Config struct {
	ForceExitThreshold int
	HookTimeout        time.Duration
}

// Status represents dynamic signal handler runtime state.
// These values change as the handler processes signals and manages lifecycle.
type Status struct {
	Received      os.Signal
	Reason        Reason
	Stopping      bool // Context is cancelled, hooks may be running
	SignalCount   int
	ResetDeadline time.Time
	Enabled       bool // Monitoring loop is active (shielding escalation)
	Stopped       bool // All shutdown hooks have finished
}

// State combines configuration and runtime status for introspection.
// This struct is used for type-safe state watching and diagram generation.
type State struct {
	Config
	Status
}

// State returns a snapshot of the current configuration.
func (sc *Context) State() State {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	stopping := false
	select {
	case <-sc.Context.Done():
		stopping = true
	default:
	}

	stopped := false
	select {
	case <-sc.hooksDone:
		stopped = true
	default:
	}

	var deadline time.Time
	if !sc.lastSignal.IsZero() {
		deadline = sc.lastSignal.Add(sc.opts.resetTimeout)
	}

	return State{
		Config: Config{
			ForceExitThreshold: sc.opts.forceExitThreshold,
			HookTimeout:        sc.opts.hookTimeout,
		},
		Status: Status{
			Received:      sc.sigVal,
			Reason:        sc.reason,
			Stopping:      stopping,
			SignalCount:   sc.signalCount,
			ResetDeadline: deadline,
			Enabled:       !sc.stopped,
			Stopped:       stopped,
		},
	}
}

// runHooks executes registered hooks in LIFO order using a Pop-Loop strategy.
// This allows hooks to register *new* hooks that will be executed immediately (LIFO).
// runHooks executes registered hooks in LIFO order using a Pop-Loop strategy.
// This allows hooks to register *new* hooks that will be executed immediately (LIFO).
func (sc *Context) runHooks() {
	sc.hooksOnce.Do(func() {
		oldState := sc.State()
		defer func() {
			close(sc.hooksDone)
			newState := sc.State()
			sc.emitStateChange(oldState, newState)
		}()

		for {
			sc.mu.Lock()
			if len(sc.hooks) == 0 {
				sc.mu.Unlock()
				return
			}
			// Pop the last hook
			lastIdx := len(sc.hooks) - 1
			h := sc.hooks[lastIdx]
			sc.hooks = sc.hooks[:lastIdx]
			sc.mu.Unlock()

			safeRunLoop(h, sc.opts.hookTimeout)
		}
	})
}

func safeRunLoop(f func(), timeout time.Duration) {
	start := time.Now()
	done := make(chan struct{})

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic in release hook", "panic", r)
				metrics.GetProvider().IncHookPanicked()
			}
			close(done)
		}()
		f()
	}()

	// Stalled Hook Detection
	// We don't kill the hook (unsafe), but we warn if it takes too long.
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-done:
		metrics.GetProvider().IncHookExecuted()
		metrics.GetProvider().ObserveHookDuration(time.Since(start))
	case <-timer.C:
		log.Warn("hook execution stalled", "elapsed", time.Since(start), "timeout", timeout)
		// We still wait for it, because we can't kill it safely.
		// The main `ctx.Wait()` will block forever if this hangs forever, which is intended (Fail-Closed).
		<-done
		metrics.GetProvider().IncHookExecuted()
		metrics.GetProvider().ObserveHookDuration(time.Since(start))
	}
}
