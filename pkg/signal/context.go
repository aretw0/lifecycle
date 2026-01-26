package signal

import (
	"context"
	"os"
	ossignal "os/signal"
	"sync"
	"syscall"
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
}

// NewContext creates a context that is cancelled on SIGTERM (standard termination).
// It captures SIGINT (Interrupt) separately to allow the state machine to handle it.
//
// Usage:
//
//	ctx := signal.NewContext(context.Background())
//	defer ctx.Cancel()
//
//	select {
//	case <-ctx.Done():
//	    // Check if it was a signal or just a cancel
//	    if sig := ctx.Signal(); sig != nil {
//	        fmt.Println("Received signal:", sig)
//	    }
//	}
func NewContext(parent context.Context) *Context {
	ctx, cancel := context.WithCancel(parent)
	sc := &Context{
		Context: ctx,
		Cancel:  cancel,
		sigCh:   make(chan os.Signal, 1),
	}

	sc.start.Do(func() {
		// We only cancel the context on SIGTERM.
		// SIGINT is captured but DOES NOT automatically cancel the context here,
		// because we usually want the application to handle it gracefully (e.g. "Do you want to quit?").
		ossignal.Notify(sc.sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			select {
			case sig := <-sc.sigCh:
				sc.mu.Lock()
				sc.sigVal = sig
				sc.mu.Unlock()
				if sig == syscall.SIGTERM {
					sc.Cancel()
				}
			case <-sc.Context.Done():
				// Context cancelled elsewhere
			}
			sc.stop.Do(func() {
				ossignal.Stop(sc.sigCh)
			})
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
