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

	// Open channels to track hook execution
	hook1Done := make(chan struct{})
	hook2Done := make(chan struct{})
	done := make(chan struct{})

	ctx.OnShutdown(func() {
		close(hook1Done)
		// Register another hook dynamically
		ctx.OnShutdown(func() {
			close(hook2Done)
			close(done)
		})
	})

	// Trigger signal
	ctx.sigCh <- syscall.SIGTERM

	<-done // só libera após hook2 rodar

	select {
	case <-hook1Done:
		// ok
	default:
		t.Error("Hook 1 should have run")
	}
	select {
	case <-hook2Done:
		// ok
	default:
		t.Error("FAIL: Late registered hooks should be executed (LIFO)!")
	}
}
