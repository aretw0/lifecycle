package signal

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/metrics"
)

type mockProvider struct {
	signals []string
}

func (m *mockProvider) IncSignalReceived(sig string)        { m.signals = append(m.signals, sig) }
func (m *mockProvider) IncProcessStarted()                  {}
func (m *mockProvider) IncProcessFailed()                   {}
func (m *mockProvider) IncTerminalUpgrade(success bool)     {}
func (m *mockProvider) IncHookExecuted()                    { m.signals = append(m.signals, "HookExecuted") }
func (m *mockProvider) IncHookPanicked()                    {}
func (m *mockProvider) ObserveHookDuration(d time.Duration) {}

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
	case <-time.After(1 * time.Second):
		t.Error("Context was not cancelled after signal")
	}
}

func TestSignalContext_Options(t *testing.T) {
	t.Run("WithInterrupt_False", func(t *testing.T) {
		ctx := NewContext(context.Background(), WithInterrupt(false))
		defer ctx.Stop()

		ctx.sigCh <- os.Interrupt

		select {
		case <-ctx.Done():
			t.Error("Context should NOT be cancelled by Interrupt when WithInterrupt(false)")
		case <-time.After(100 * time.Millisecond):
			// Success
		}
	})

	t.Run("WithForceExit_Default", func(t *testing.T) {
		// We can't easily test os.Exit in a unit test without subprocesses,
		// but we can test the handleSignal logic directly.
		sc := &Context{
			Context: context.Background(),
			Cancel:  func() {},
			opts: options{
				forceExitThreshold: 2,
			},
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
	sc := &Context{opts: options{interruptCancel: true}}

	if !sc.shouldCancel(syscall.SIGTERM) {
		t.Error("SIGTERM should always cancel")
	}
	if !sc.shouldCancel(os.Interrupt) {
		t.Error("SIGINT should cancel when interruptCancel is true")
	}

	sc.opts.interruptCancel = false
	if sc.shouldCancel(os.Interrupt) {
		t.Error("SIGINT should NOT cancel when interruptCancel is false")
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

	mp := &mockProvider{}
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
	for _, s := range mp.signals {
		if s == "HookExecuted" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected HookExecuted metric even after stall warning")
	}
}
