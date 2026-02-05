package signal

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"github.com/aretw0/lifecycle/pkg/core/metrics/mock"
)

func TestSignalContext_Graceful(t *testing.T) {
	ctx := NewContext(context.Background())
	defer ctx.Stop()

	// Simulate a signal by sending to the internal channel
	ctx.sigCh <- syscall.SIGTERM

	select {
	case <-ctx.Done():
		// Success
		if ctx.Signal() != syscall.SIGTERM {
			t.Errorf("Expected signal SIGTERM, got %v", ctx.Signal())
		}
		if ctx.Reason() != ReasonTerminate {
			t.Errorf("Expected reason Terminate, got %v", ctx.Reason())
		}
	case <-time.After(1 * time.Second):
		t.Error("Context was not cancelled after signal")
	}
}

func TestSignalContext_Reason(t *testing.T) {
	t.Run("Interrupt", func(t *testing.T) {
		ctx := NewContext(context.Background())
		defer ctx.Stop()
		ctx.sigCh <- os.Interrupt
		<-ctx.Done()
		if ctx.Reason() != ReasonInterrupt {
			t.Errorf("Expected ReasonInterrupt, got %v", ctx.Reason())
		}
	})

	t.Run("ManualStop", func(t *testing.T) {
		ctx := NewContext(context.Background())
		ctx.Stop()
		if ctx.Reason() != ReasonManualStop {
			t.Errorf("Expected ReasonManualStop after Stop(), got %v", ctx.Reason())
		}
	})

	t.Run("ManualCancel", func(t *testing.T) {
		ctx := NewContext(context.Background())
		ctx.Cancel()
		// Wait for context to be done to ensure cancellation propagated
		<-ctx.Done()
		if ctx.Reason() != ReasonManualCancel {
			t.Errorf("Expected ReasonManualCancel after Cancel(), got %v", ctx.Reason())
		}
	})
}

func TestSignalContext_Options(t *testing.T) {
	t.Run("WithForceExit_0_WithCancelOnInterrupt_false", func(t *testing.T) {
		ctx := NewContext(context.Background(),
			WithForceExit(0),
			WithCancelOnInterrupt(false), // Explicit: Don't cancel
		)
		defer ctx.Stop()

		ctx.sigCh <- os.Interrupt

		select {
		case <-ctx.Done():
			t.Error("Context should NOT be cancelled by Interrupt when WithForceExit(0) and WithCancelOnInterrupt(false)")
		case <-time.After(100 * time.Millisecond):
			// Success
		}
	})

	t.Run("With_Default", func(t *testing.T) {
		// We can't easily test os.Exit in a unit test without subprocesses,
		// but we can test the handleSignal logic directly.
		sc := &Context{
			Context: context.Background(),
		}

		// This is just to verify no panic and basic first signal logic
		sc.handleSignal(os.Interrupt, 1)
		if sc.Signal() != os.Interrupt {
			t.Error("Signal was not recorded")
		}
	})
}

func TestSignalContext_Stop(t *testing.T) {
	ctx := NewContext(context.Background())
	ctx.Stop() // Should close the channel and stop the goroutine

	// Verify channel is closed
	_, ok := <-ctx.sigCh
	if ok {
		t.Error("sigCh should be closed after Stop()")
	}
}

func TestShouldCancel(t *testing.T) {
	// Test with cancelOnInterrupt=true (default)
	sc := &Context{opts: options{cancelOnInterrupt: true}}

	if !sc.shouldCancel(syscall.SIGTERM) {
		t.Error("SIGTERM should always cancel")
	}
	if !sc.shouldCancel(os.Interrupt) {
		t.Error("SIGINT should cancel when cancelOnInterrupt is true")
	}

	// Test with cancelOnInterrupt=false
	sc.opts.cancelOnInterrupt = false
	if !sc.shouldCancel(syscall.SIGTERM) {
		t.Error("SIGTERM should always cancel regardless of cancelOnInterrupt")
	}
	if sc.shouldCancel(os.Interrupt) {
		t.Error("SIGINT should NOT cancel when cancelOn Interrupt is false")
	}
}

func TestSignalContext_Hooks(t *testing.T) {
	ctx := NewContext(context.Background())
	defer ctx.Stop()

	var execution []string
	// We rely on Wait() to ensure all hooks are done.

	// Hook A (First registered, Last executed)
	ctx.OnShutdown(func() {
		execution = append(execution, "A")
	})
	// Hook B
	ctx.OnShutdown(func() {
		execution = append(execution, "B")
	})
	// Hook C (Last registered, First executed)
	ctx.OnShutdown(func() {
		execution = append(execution, "C")
	})

	// Trigger signal
	ctx.sigCh <- syscall.SIGTERM

	// Wait for hooks
	ctx.Wait()

	if len(execution) != 3 {
		t.Fatalf("Expected 3 hooks, got %d", len(execution))
	}
	if execution[0] != "C" || execution[1] != "B" || execution[2] != "A" {
		t.Errorf("Expected order [C B A], got %v", execution)
	}
}

func TestSignalContext_Hooks_Panic(t *testing.T) {
	ctx := NewContext(context.Background())
	defer ctx.Stop()

	var execution []string

	// Hook A (Final)
	ctx.OnShutdown(func() {
		execution = append(execution, "A")
	})
	// Hook B (Panics)
	ctx.OnShutdown(func() {
		panic("oops")
	})
	// Hook C (First)
	ctx.OnShutdown(func() {
		execution = append(execution, "C")
	})

	ctx.sigCh <- syscall.SIGTERM
	ctx.Wait()

	// B should have panicked but A should still run.
	if len(execution) != 2 {
		t.Errorf("Expected 2 successful hooks (A, C), got %v", execution)
	}
	if execution[0] != "C" || execution[1] != "A" {
		t.Errorf("Expected order [C A] (skipping B), got %v", execution)
	}
}

func TestSignalContext_DynamicHooks(t *testing.T) {
	ctx := NewContext(context.Background())
	defer ctx.Stop()

	var execution []string

	// Hook A
	ctx.OnShutdown(func() {
		execution = append(execution, "A")
		// Register Hook B dynamically
		// LIFO Logic: Since this is added *during* the loop which pops elements,
		// it is appended to the list and becomes the next "last" element.
		// Thus, it runs immediately after this hook returns.
		ctx.OnShutdown(func() {
			execution = append(execution, "B")
		})
	})

	ctx.sigCh <- syscall.SIGTERM
	ctx.Wait()

	if len(execution) != 2 {
		t.Fatalf("Expected 2 hooks, got %v", execution)
	}
	// A runs first (popped). Inside A, B is added. A finishes. Loop checks length... finds B. Pops B.
	if execution[0] != "A" || execution[1] != "B" {
		t.Errorf("Expected [A B], got %v", execution)
	}
}

func TestSignalContext_HookTimeout(t *testing.T) {
	// Set a very short timeout for testing
	timeout := 10 * time.Millisecond
	ctx := NewContext(context.Background(), WithHookTimeout(timeout))
	defer ctx.Stop()

	// Use a mock provider to detect if stats were emitted (indirect way to verify flow)
	// Ideally we'd capture logs, but since we can't easily Mock slog in this setup without dependency inject changes,
	// we will rely on the fact that stalled detection *eventually* emits metrics when it finishes.
	// But we wait... the stall log happens on timer tick.
	// For this test, we just want to ensure it doesn't BLOCK forever or panic.

	// Use centralized mock
	mp := mock.New()
	metrics.SetProvider(mp)

	hookDone := make(chan struct{})

	ctx.OnShutdown(func() {
		// Sleep longer than the timeout
		time.Sleep(50 * time.Millisecond)
		close(hookDone)
	})

	ctx.sigCh <- syscall.SIGTERM

	// Wait for completion (slow hook)
	ctx.Wait()

	// Verify it actually finished
	select {
	case <-hookDone:
		// success
	default:
		t.Error("Hook should have finished eventually")
	}

	// Verify metric was recorded (meaning the timeout logic didn't abort execution flow)
	found := false
	mp.Mu.Lock()
	for _, s := range mp.Signals {
		if s == "HookExecuted" {
			found = true
			break
		}
	}
	mp.Mu.Unlock()
	if !found {
		t.Error("Expected HookExecuted metric even after stall warning")
	}
}

func TestSignalContext_State_NoDestructiveRead(t *testing.T) {
	ctx := NewContext(context.Background())
	defer ctx.Stop()

	// Put a signal in the channel
	sig := os.Interrupt
	ctx.sigCh <- sig

	// Call State() - this used to call isChannelOpen which would consume the signal
	_ = ctx.State()

	// Verify the signal is still there
	select {
	case s := <-ctx.sigCh:
		if s != sig {
			t.Errorf("Expected signal %v, got %v", sig, s)
		}
	default:
		t.Error("Signal was consumed by State()")
	}
}

func TestSignalContext_WithResetTimeout(t *testing.T) {
	timeout := 100 * time.Millisecond
	opt := WithResetTimeout(timeout)

	ctx := NewContext(context.Background(), opt)
	defer ctx.Stop()

	// Verify option was applied by checking context is created successfully
	state := ctx.State()
	if state == (State{}) {
		t.Error("State() returned empty state after WithResetTimeout")
	}
}

func TestSignalContext_IsUnsafe(t *testing.T) {
	t.Run("SafeWithDefaultThreshold", func(t *testing.T) {
		ctx := NewContext(context.Background())
		defer ctx.Stop()

		// Default threshold should be safe (not 0)
		unsafe := ctx.IsUnsafe()
		if unsafe {
			t.Errorf("Expected IsUnsafe() to be false for default threshold, got true")
		}
	})

	t.Run("UnsafeWhenThresholdIsZero", func(t *testing.T) {
		ctx := NewContext(context.Background(), WithForceExit(0))
		defer ctx.Stop()

		// Force exit threshold of 0 means unsafe
		unsafe := ctx.IsUnsafe()
		if !unsafe {
			t.Errorf("Expected IsUnsafe() to be true when ForceExit(0), got false")
		}
	})
}

func TestSignalContext_ForceExitThreshold(t *testing.T) {
	ctx := NewContext(context.Background())
	defer ctx.Stop()

	// Get the force exit threshold
	threshold := ctx.ForceExitThreshold()
	if threshold == 0 {
		t.Error("ForceExitThreshold() returned 0, expected non-zero")
	}
}
