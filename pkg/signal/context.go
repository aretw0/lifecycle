package signal

import (
	"context"
	"os"
	ossignal "os/signal"
	"sync"
	"syscall"

	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
)

// Context wraps a context and captures the signal that cancelled it.
type Context struct {
	context.Context
	Cancel func()
	start  sync.Once
	stop   sync.Once
	sigCh  chan os.Signal
	sigVal os.Signal
	mu     sync.Mutex
	opts   options
}

type options struct {
	interruptCancel    bool
	forceExitThreshold int
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

// Stop stops the signal monitoring and restores default behavior.
func (sc *Context) Stop() {
	sc.stop.Do(func() {
		ossignal.Stop(sc.sigCh)
	})
}

// NewContext creates a context that is cancelled on SIGTERM or SIGINT (standard termination).
// On the first signal received, it cancels the context to initiate graceful shutdown (if configured).
// If a second signal is received before the program exits, it performs an immediate os.Exit(1) (if configured).
func NewContext(parent context.Context, opts ...Option) *Context {
	o := options{
		interruptCancel:    true,
		forceExitThreshold: 2,
	}
	for _, opt := range opts {
		opt(&o)
	}

	ctx, cancel := context.WithCancel(parent)
	sc := &Context{
		Context: ctx,
		Cancel:  cancel,
		sigCh:   make(chan os.Signal, 1),
		opts:    o,
	}

	sc.start.Do(func() {
		// Capture standard termination signals.
		ossignal.Notify(sc.sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			count := 0
			for {
				select {
				case sig, ok := <-sc.sigCh:
					if !ok {
						return
					}
					count++
					sc.mu.Lock()
					isFirst := sc.sigVal == nil
					sc.sigVal = sig
					sc.mu.Unlock()

					log.Debug("received signal", "signal", sig.String(), "count", count, "first", isFirst)
					metrics.GetProvider().IncSignalReceived(sig.String())

					// Determine if we should cancel the context
					shouldCancel := false
					if sig == syscall.SIGTERM {
						shouldCancel = true
					} else if sig == os.Interrupt && sc.opts.interruptCancel {
						shouldCancel = true
					}

					if isFirst && shouldCancel {
						sc.Cancel()
					}

					// Check for Force Exit threshold
					if sc.opts.forceExitThreshold > 0 && count >= sc.opts.forceExitThreshold {
						log.Warn("force exit threshold reached, exiting immediately",
							"signal", sig.String(),
							"count", count)
						os.Exit(1)
					}
				case <-sc.Context.Done():
					// We continue looping even after Done() to catch the force exit signals during cleanup.
				}
			}
		}()
	})

	return sc
}

// Signal returns the signal that caused the context to be cancelled/interrupted, or nil.
func (sc *Context) Signal() os.Signal {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.sigVal
}
