package signal

import (
	"context"
	"syscall"
	"testing"
	"time"
)

// TestCaveat_AsyncHooksRace demonstrates that hooks might not finish before main exits
// if they are run asynchronously without a wait mechanism.
func TestCaveat_AsyncHooksRace(t *testing.T) {
	ctx := NewContext(context.Background())
	defer ctx.Stop()

	hookStarted := make(chan struct{})
	hookFinished := make(chan struct{})

	// Register a slow hook
	ctx.OnShutdown(func() {
		close(hookStarted)
		time.Sleep(100 * time.Millisecond) // Simulate work
		close(hookFinished)
	})

	// Trigger signal
	ctx.sigCh <- syscall.SIGTERM

	// Wait for the Context to signal "Done"
	<-ctx.Done()

	// Standard pattern: Main waits for hooks
	ctx.Wait()

	// Check if hook has finished (it MUST have, because Wait returned)
	select {
	case <-hookFinished:
		// success
	default:
		t.Error("FAIL: Wait() returned but hooks are not done!")
	}
}

// TestCaveat_LateRegistration demonstrates that hooks added during shutdown are ignored.
func TestCaveat_LateRegistration(t *testing.T) {
	ctx := NewContext(context.Background())
	defer ctx.Stop()

	hook1Run := false
	hook2Run := false

	done := make(chan struct{})

	ctx.OnShutdown(func() {
		hook1Run = true
		// Register another hook dynamically
		ctx.OnShutdown(func() {
			hook2Run = true
		})
	})

	// Add a final hook to close the test channel
	ctx.OnShutdown(func() {
		close(done)
	})

	ctx.sigCh <- syscall.SIGTERM

	<-done
	// Wait a bit to ensure async weirdness settles
	time.Sleep(50 * time.Millisecond)

	if !hook1Run {
		t.Error("Hook 1 should have run")
	}
	if !hook2Run {
		t.Error("FAIL: Late registered hooks should be executed (LIFO)!")
	}
}
