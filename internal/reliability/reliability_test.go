package reliability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/metrics"
	"github.com/aretw0/lifecycle/pkg/metrics/mock"
)

func TestDo_ReturnsError(t *testing.T) {
	// Setup Mock Metrics
	mp := mock.New()
	metrics.SetProvider(mp)

	expectedErr := errors.New("critical failure")
	ctx := context.Background()

	err := Do(ctx, func(c context.Context) error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	// Verify Metrics
	if mp.CriticalSectionStarted != 1 {
		t.Errorf("expected 1 critical section start, got %d", mp.CriticalSectionStarted)
	}
	if mp.CriticalSectionFinished != 1 {
		t.Errorf("expected 1 critical section finish, got %d", mp.CriticalSectionFinished)
	}
	if mp.CriticalSectionSuccesses != 0 {
		t.Error("expected 0 successes (since error returned)")
	}
	if mp.CriticalSectionFailures != 1 {
		t.Error("expected 1 failure")
	}
}

func TestDo_Success(t *testing.T) {
	mp := mock.New()
	metrics.SetProvider(mp)
	ctx := context.Background()

	err := Do(ctx, func(c context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if mp.CriticalSectionSuccesses != 1 {
		t.Error("expected critical section to be marked as success")
	}
}

func TestDo_Panics(t *testing.T) {
	mp := mock.New()
	metrics.SetProvider(mp)
	ctx := context.Background()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic to propagate")
		}

		// Verify metrics captured BEFORE re-panic
		if mp.CriticalSectionFinished != 1 {
			t.Errorf("expected 1 critical section finish, got %d", mp.CriticalSectionFinished)
		}
		if mp.CriticalSectionFailures != 1 {
			t.Error("expected critical section to be marked as failure")
		}
		if mp.CriticalSectionDuration == 0 {
			t.Error("expected duration to be recorded")
		}
	}()

	Do(ctx, func(c context.Context) error {
		time.Sleep(1 * time.Millisecond) // Ensure duration > 0
		panic("boom")
	})
}

func TestDo_PreservesValues(t *testing.T) {
	metrics.SetProvider(mock.New())
	key := "user-id"
	val := "12345"
	ctx := context.WithValue(context.Background(), key, val)

	err := Do(ctx, func(c context.Context) error {
		if v := c.Value(key); v != val {
			return errors.New("context value lost")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected context values to be preserved: %v", err)
	}
}
