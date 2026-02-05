package worker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFromFunc(t *testing.T) {
	// Good case
	w := FromFunc("good", func(ctx context.Context) error {
		return nil
	})

	ctx := context.Background()
	if err := w.Start(ctx); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-w.Wait():
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for good worker")
	}

	// Bad case
	w = FromFunc("bad", func(ctx context.Context) error {
		return errors.New("oops")
	})

	if err := w.Start(ctx); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-w.Wait():
		if err == nil || err.Error() != "oops" {
			t.Errorf("Expected oops error, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for bad worker")
	}

	// Cancel case
	w = FromFunc("cancel", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	if err := w.Start(ctx); err != nil {
		t.Fatal(err)
	}

	if err := w.Stop(ctx); err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	select {
	case err := <-w.Wait():
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for cancelled worker")
	}
}



