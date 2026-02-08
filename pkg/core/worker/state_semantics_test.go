package worker_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/worker"
)

func TestFuncWorker_StateSemantics(t *testing.T) {
	// 1. Natural Completion -> Finished
	t.Run("Natural Completion", func(t *testing.T) {
		w := worker.FromFunc("natural", func(ctx context.Context) error {
			return nil
		})

		_ = w.Start(context.Background())
		<-w.Wait()

		state := w.State()
		if state.Status != worker.StatusFinished {
			t.Errorf("expected Finished, got %s", state.Status)
		}
	})

	// 2. Requested Stop -> Stopped
	t.Run("Requested Stop", func(t *testing.T) {
		w := worker.FromFunc("requested", func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})

		_ = w.Start(context.Background())
		_ = w.Stop(context.Background())
		<-w.Wait()

		state := w.State()
		if state.Status != worker.StatusStopped {
			t.Errorf("expected Stopped, got %s", state.Status)
		}
	})

	// 3. Error -> Failed
	t.Run("Error Failure", func(t *testing.T) {
		w := worker.FromFunc("failure", func(ctx context.Context) error {
			return errors.New("oops")
		})

		_ = w.Start(context.Background())
		<-w.Wait()

		state := w.State()
		if state.Status != worker.StatusFailed {
			t.Errorf("expected Failed, got %s", state.Status)
		}
	})
}

func TestProcessWorker_StateSemantics(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping process tests in CI environment")
	}

	getCmd := func() (string, []string) {
		if os.PathSeparator == '\\' {
			return "ping", []string{"127.0.0.1", "-n", "2"}
		}
		return "sleep", []string{"1"}
	}

	// 1. Natural Completion -> Finished
	t.Run("Natural Completion", func(t *testing.T) {
		cmd, args := getCmd()
		w := worker.NewProcessWorker("proc-natural", cmd, args...)

		_ = w.Start(context.Background())
		<-w.Wait()

		state := w.State()
		if state.Status != worker.StatusFinished {
			t.Errorf("expected Finished, got %s (exit code %d)", state.Status, state.ExitCode)
		}
	})

	// 2. Requested Stop -> Stopped
	t.Run("Requested Stop", func(t *testing.T) {
		// Long running process
		var cmd string
		var args []string
		if os.PathSeparator == '\\' {
			cmd, args = "ping", []string{"127.0.0.1", "-n", "20"}
		} else {
			cmd, args = "sleep", []string{"20"}
		}
		w := worker.NewProcessWorker("proc-stop", cmd, args...)

		_ = w.Start(context.Background())
		time.Sleep(100 * time.Millisecond) // Let it start
		_ = w.Stop(context.Background())
		<-w.Wait()

		state := w.State()
		if state.Status != worker.StatusStopped {
			t.Errorf("expected Stopped, got %s", state.Status)
		}
	})

	// 3. Forced Kill -> Killed (Mocking forced kill via short timeout on Stop)
	t.Run("Forced Kill", func(t *testing.T) {
		// Non-cooperative process (ping ignores SIGTERM on Windows? No, usually handles it or ignored.
		// On Windows, Signal(SIGTERM) is not fully supported, so it might just force kill anyway?
		// Actually ProcessWorker.Stop sends SIGTERM.
		// If we set a super short timeout for Stop, it should trigger Kill.

		var cmd string
		var args []string
		if os.PathSeparator == '\\' {
			cmd, args = "ping", []string{"127.0.0.1", "-n", "20"}
		} else {
			cmd, args = "sleep", []string{"20"}
		}
		w := worker.NewProcessWorker("proc-kill", cmd, args...)

		_ = w.Start(context.Background())
		time.Sleep(100 * time.Millisecond)

		// Short timeout to force kill
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		_ = w.Stop(ctx)
		<-w.Wait()

		state := w.State()
		// On Windows, Stop might return immediately?
		// We expect StatusKilled if the select hit ctx.Done()
		if state.Status != worker.StatusKilled {
			t.Errorf("expected Killed, got %s", state.Status)
		}
	})
}
