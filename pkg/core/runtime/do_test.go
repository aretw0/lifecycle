package runtime

import (
	"context"
	"errors"
	"testing"
)

func TestDo(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		executed := false
		err := Do(context.Background(), func(ctx context.Context) error {
			executed = true
			return nil
		})
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
		if !executed {
			t.Error("Function was not executed")
		}
	})

	t.Run("Error", func(t *testing.T) {
		expectedErr := errors.New("failed")
		err := Do(context.Background(), func(ctx context.Context) error {
			return expectedErr
		})
		if err != expectedErr {
			t.Errorf("Expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("PanicRecovery", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic to propagate, but it was suppressed")
			}
		}()

		_ = Do(context.Background(), func(ctx context.Context) error {
			panic("oops")
		})
	})
}

func TestDoDetached(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel() // Cancel parent immediately

	executed := false
	err := DoDetached(parent, func(ctx context.Context) error {
		// Verify context is NOT cancelled despite parent being cancelled
		select {
		case <-ctx.Done():
			t.Error("Detached context should not be cancelled")
		default:
			executed = true
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if !executed {
		t.Error("Function was not executed")
	}
}

func TestDo_ExecutionDespiteCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel upfront

	executed := false
	err := Do(ctx, func(ctx context.Context) error {
		executed = true
		return nil
	})

	if err != nil {
		t.Errorf("Expected nil error despite cancellation (reliability block), got %v", err)
	}
	if !executed {
		t.Error("Function should have been executed (Do is a reliability block)")
	}
}
