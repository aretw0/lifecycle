package runtime

import (
	"context"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/log"
)

// BlockWithTimeout blocks until the done channel is closed or the timeout expires.
// It returns nil if the operation completed (channel closed), or context.DeadlineExceeded
// if the timeout occurred.
//
// Usage:
//
//	go func() {
//	    DoCleanup()
//	    close(done)
//	}()
//	if err := lifecycle.BlockWithTimeout(done, 5*time.Second); err != nil {
//	    // Force exit or log error
//	}
func BlockWithTimeout(done <-chan struct{}, timeout time.Duration) error {
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		log.Warn("timeout reached while blocking", "timeout", timeout)
		return context.DeadlineExceeded
	}
}



