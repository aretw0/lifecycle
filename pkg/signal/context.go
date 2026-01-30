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

// Context wraps a context and captures the signal that cancelled it.
type Context struct {
	context.Context
	Cancel    func()
	start     sync.Once
	stop      sync.Once
	sigCh     chan os.Signal
	sigVal    os.Signal
	mu        sync.Mutex
	opts      options
	hooks     []func()
	hooksDone chan struct{}
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
	interruptCancel    bool
	forceExitThreshold int
	hookTimeout        time.Duration
}

// Option is a functional option for configuring signal behavior.
type Option func(*options)

// WithInterrupt configures whether SIGINT (Ctrl+C) should cancel the context.
// Default is true.
func WithInterrupt(cancel bool) Option {
	return func(o *options) {
		o.interruptCancel = cancel
	}
}

// WithForceExit configures the threshold of signals required to trigger an immediate os.Exit(1).
// Set to 0 to disable forced exit. Default is 2.
func WithForceExit(threshold int) Option {
	return func(o *options) {
		o.forceExitThreshold = threshold
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
		interruptCancel:    true,
		forceExitThreshold: 2,
		hookTimeout:        5 * time.Second,
	}
	for _, opt := range opts {
		opt(&o)
	}

	ctx, cancel := context.WithCancel(parent)
	sc := &Context{
		Context:   ctx,
		Cancel:    cancel,
		sigCh:     make(chan os.Signal, 1),
		hooksDone: make(chan struct{}),
		opts:      o,
	}

	sc.start.Do(func() {
		ossignal.Notify(sc.sigCh, os.Interrupt, syscall.SIGTERM)
		go sc.monitor()
	})

	return sc
}

// monitor runs the signal monitoring loop.
func (sc *Context) monitor() {
	count := 0
	for {
		select {
		case sig, ok := <-sc.sigCh:
			if !ok {
				return
			}
			count++
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
	sc.sigVal = sig
	sc.mu.Unlock()

	log.Debug("received signal", "signal", sig.String(), "count", count, "first", isFirst)
	metrics.GetProvider().IncSignalReceived(sig.String())

	if isFirst && sc.shouldCancel(sig) {
		sc.Cancel()
		go sc.runHooks()
	}

	if sc.opts.forceExitThreshold > 0 && count >= sc.opts.forceExitThreshold {
		log.Warn("force exit threshold reached, exiting immediately",
			"signal", sig.String(),
			"count", count)
		metrics.GetProvider().IncForceExitTriggered()
		os.Exit(1)
	}
}

// shouldCancel determines if the given signal should cancel the context.
func (sc *Context) shouldCancel(sig os.Signal) bool {
	if sig == syscall.SIGTERM {
		return true
	}
	if sig == os.Interrupt && sc.opts.interruptCancel {
		return true
	}
	return false
}

// Signal returns the signal that caused the context to be cancelled/interrupted, or nil.
func (sc *Context) Signal() os.Signal {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.sigVal
}

// State represents the configuration state of the SignalContext.
type State struct {
	InterruptCancel    bool
	ForceExitThreshold int
	HookTimeout        time.Duration
	Received           os.Signal
	Stopping           bool
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

	return State{
		InterruptCancel:    sc.opts.interruptCancel,
		ForceExitThreshold: sc.opts.forceExitThreshold,
		HookTimeout:        sc.opts.hookTimeout,
		Received:           sc.sigVal,
		Stopping:           stopping,
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
