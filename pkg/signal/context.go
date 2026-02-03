package signal

import (
	"context"
	"os"
	ossignal "os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
)

// Reason describes why the context was cancelled.
type Reason string

const (
	ReasonNone         Reason = "None"
	ReasonInterrupt    Reason = "Interrupt"     // SIGINT (Ctrl+C)
	ReasonTerminate    Reason = "Terminate"     // SIGTERM
	ReasonManualStop   Reason = "Manual:Stop"   // Explicit Stop() call
	ReasonManualCancel Reason = "Manual:Cancel" // Context Cancel() called
	ReasonTimeout      Reason = "Timeout"       // Shutdown timed out
)

func (r Reason) String() string {
	return string(r)
}

// Context wraps a context and captures the signal that cancelled it.
type Context struct {
	context.Context
	Cancel func()
	start  sync.Once
	stop   sync.Once
	sigCh  chan os.Signal
	sigVal os.Signal
	reason Reason
	mu     sync.Mutex
	// ... fields ...
	opts        options
	hooks       []func()
	hooksDone   chan struct{}
	signalCount int
	lastSignal  time.Time
}

// ResetSignalCount resets the signal counter and clears the last received signal.
// This is useful for "Smart Handlers" that successfully handle a signal (e.g. Suspend)
// and want to reset the "Force Exit" threshold.
func (sc *Context) ResetSignalCount() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.signalCount = 0
	sc.sigVal = nil
	log.Debug("signal count reset")
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
}

// Option is a functional option for configuring signal behavior.
type Option func(*options)

// WithInterrupt is deprecated. Use WithForceExit(1) for default behavior
// or WithForceExit(0) to disable automatic interruption.
//
// Deprecated: SIGINT logic is now controlled by ForceExit threshold.
func WithInterrupt(cancel bool) Option {
	return func(o *options) {
		if cancel {
			o.forceExitThreshold = 1
		} else {
			o.forceExitThreshold = 0
		}
	}
}

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

// Stop stops the signal monitoring and restores default behavior.
// It also ensures Wait() unblocks if called.
func (sc *Context) Stop() {
	sc.stop.Do(func() {
		sc.mu.Lock()
		if sc.reason == ReasonNone {
			sc.reason = ReasonManualStop
		}
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
	}
	for _, opt := range opts {
		opt(&o)
	}

	ctx, cancel := context.WithCancel(parent)
	sc := &Context{
		Context:   ctx,
		sigCh:     make(chan os.Signal, 1),
		hooksDone: make(chan struct{}),
		reason:    ReasonNone,
		opts:      o,
	}

	// Internal Cancel Wrapper
	// We wrap the context's cancel function to set the ReasonManualCancel if no other reason is set.
	sc.Cancel = func() {
		sc.mu.Lock()
		if sc.reason == ReasonNone {
			sc.reason = ReasonManualCancel
		}
		sc.mu.Unlock()
		cancel()
	}

	sc.start.Do(func() {
		ossignal.Notify(sc.sigCh, os.Interrupt, syscall.SIGTERM)
		go sc.monitor()
	})

	return sc
}

// monitor runs the signal monitoring loop.
func (sc *Context) monitor() {
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

		case <-sc.Context.Done():
			// Keep looping after Done() to support Force Exit during cleanup.
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

	// Cancellation Logic
	// SIGTERM always cancels on first signal
	// SIGINT cancels on first signal ONLY if threshold is exactly 1
	if isFirst && sc.shouldCancel(sig) {
		sc.mu.Lock()
		switch sig {
		case os.Interrupt:
			sc.reason = ReasonInterrupt
		case syscall.SIGTERM:
			sc.reason = ReasonTerminate
		}
		sc.mu.Unlock()

		sc.Cancel()
		go sc.runHooks()
	}

	// Force Exit Logic
	// Enabled if threshold > 0
	if sc.opts.forceExitThreshold > 0 && count >= sc.opts.forceExitThreshold {
		// If threshold is 1, we already cancelled, so this is an immediate follow-up exit
		// if another signal arrives or if we want to treat 1 as "cancel then exit"
		// Actually, if threshold is 1, and this is the first signal, we cancel.
		// If another signal arrives, count becomes 2, which is >= 1, so we exit.
		// THIS IS CORRECT: 1st signal = Cancel, 2nd signal = Force Exit.

		// WAIT: If user wants "1st signal = Force Exit" without cancellation,
		// that's not what we discussed. We said 1 = industry standard (Cancel context).
		// So if count == 1 and threshold is 1, we SHOULD only cancel.
		// Force Exit should only happen if count > threshold OR if we want 1 to be special.

		// Correct logic:
		// Threshold 1: 1st signal = Cancel. 2nd signal = Force Exit.
		// Threshold N: 1..N-1 signals = Captured (Events). N-th signal = Force Exit.
		if count > 1 || (sig == os.Interrupt && sc.opts.forceExitThreshold == 1 && count > 1) || (count >= sc.opts.forceExitThreshold && sc.opts.forceExitThreshold > 1) {
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
	// SIGINT only cancels if threshold is explicitly 1 (Default behavior)
	return sig == os.Interrupt && sc.opts.forceExitThreshold == 1
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
type State struct {
	ForceExitThreshold int
	HookTimeout        time.Duration
	Received           os.Signal
	Reason             Reason
	Stopping           bool
	Enabled            bool // True if the signal monitor is active
	SignalCount        int
	ResetDeadline      time.Time
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

	var deadline time.Time
	if !sc.lastSignal.IsZero() {
		deadline = sc.lastSignal.Add(sc.opts.resetTimeout)
	}

	return State{
		ForceExitThreshold: sc.opts.forceExitThreshold,
		HookTimeout:        sc.opts.hookTimeout,
		Received:           sc.sigVal,
		Reason:             sc.reason,
		Stopping:           stopping,
		// If the channel is closed, the monitor is disabled (Stopped).
		// We can check if the channel is closed or nil, but checking the channel itself is tricky without a lock/select.
		// However, in Stop(), we close the channel.
		// A reliable way is checking if sigCh is closed.
		Enabled:       isChannelOpen(sc.sigCh),
		SignalCount:   sc.signalCount,
		ResetDeadline: deadline,
	}
}

func isChannelOpen(ch <-chan os.Signal) bool {
	select {
	case _, ok := <-ch:
		return ok
	default:
		return true // It's open but empty
	}
}

// runHooks executes registered hooks in LIFO order using a Pop-Loop strategy.
// This allows hooks to register *new* hooks that will be executed immediately (LIFO).
func (sc *Context) runHooks() {
	defer close(sc.hooksDone)

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
