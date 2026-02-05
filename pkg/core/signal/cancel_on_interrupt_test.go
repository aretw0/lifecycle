package signal

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestSignalContext_CancelOnInterrupt(t *testing.T) {
	t.Run("Enabled_CancelsOnFirstSIGINT", func(t *testing.T) {
		ctx := NewContext(context.Background(),
			WithCancelOnInterrupt(true),
			WithForceExit(2),
		)
		defer ctx.Stop()

		ctx.sigCh <- os.Interrupt
		select {
		case <-ctx.Done():
			// Success - context cancelled
			if ctx.Reason() != ReasonInterrupt {
				t.Errorf("Expected ReasonInterrupt, got %v", ctx.Reason())
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Context should be cancelled on first SIGINT when enabled")
		}
	})

	t.Run("Disabled_DoesNotCancel", func(t *testing.T) {
		ctx := NewContext(context.Background(),
			WithCancelOnInterrupt(false),
			WithForceExit(2),
		)
		defer ctx.Stop()

		ctx.sigCh <- os.Interrupt
		select {
		case <-ctx.Done():
			t.Error("Context should NOT be cancelled when disabled")
		case <-time.After(100 * time.Millisecond):
			// Success - context not cancelled
		}
	})

	t.Run("Disabled_SIGTERMStillCancels", func(t *testing.T) {
		ctx := NewContext(context.Background(),
			WithCancelOnInterrupt(false),
		)
		defer ctx.Stop()

		ctx.sigCh <- syscall.SIGTERM
		select {
		case <-ctx.Done():
			// Success - SIGTERM always cancels
			if ctx.Reason() != ReasonTerminate {
				t.Errorf("Expected ReasonTerminate, got %v", ctx.Reason())
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("SIGTERM should always cancel context")
		}
	})

	t.Run("Default_IsEnabled", func(t *testing.T) {
		// Test that default behavior is cancelOnInterrupt=true
		ctx := NewContext(context.Background())
		defer ctx.Stop()

		ctx.sigCh <- os.Interrupt
		select {
		case <-ctx.Done():
			// Success - default should cancel
		case <-time.After(100 * time.Millisecond):
			t.Error("Default behavior should cancel on SIGINT")
		}
	})
}



