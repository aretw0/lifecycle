package signal

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

type mockProvider struct {
	signals []string
}

func (m *mockProvider) IncSignalReceived(sig string)    { m.signals = append(m.signals, sig) }
func (m *mockProvider) IncProcessStarted()              {}
func (m *mockProvider) IncProcessFailed()               {}
func (m *mockProvider) IncTerminalUpgrade(success bool) {}

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
