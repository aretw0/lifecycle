package worker_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/worker"
)

// HelperProcess is a magic value that allows the test binary to behave as a helper process.
const HelperProcess = "GO_HELPER_PROCESS"

func TestMain(m *testing.M) {
	// If the environment variable is set, run the helper logic instead of tests.
	if os.Getenv(HelperProcess) == "1" {
		runHelper()
		return
	}
	os.Exit(m.Run())
}

// runHelper is the entry point for the child process used in tests.
func runHelper() {
	mode := os.Args[1]
	switch mode {
	case "sleep":
		// Just sleep for a long time, waiting to be killed/stopped
		time.Sleep(1 * time.Hour)
	case "echo":
		// Echo arguments and exit
		fmt.Print(os.Args[2])
	case "fail":
		// Exit with error
		os.Exit(1)
	}
	os.Exit(0)
}

func TestProcess_StartStop(t *testing.T) {
	// Create a worker that sleeps
	w := worker.NewProcessWorker("test-sleep", os.Args[0], "sleep")
	// Inject the helper env.
	// The child process created by NewProcess will inherit the current process's environment (os.Environ()),
	// so setting the variable here is sufficient to pass it to the worker.

	// Inject the helper env using the safe instance method
	w.SetEnv(HelperProcess, "1")

	// Verify initial state
	initialState := w.State()
	if initialState.Status != worker.StatusCreated {
		t.Errorf("Expected Created status, got %s", initialState.Status)
	}

	ctx := context.Background()
	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify running state
	runningState := w.State()
	if runningState.Status != worker.StatusRunning {
		t.Errorf("Expected Running status, got %s", runningState.Status)
	}

	// Verify it's running (Wait channel shouldn't be closed immediately)
	select {
	case err := <-w.Wait():
		t.Fatalf("Worker exited prematurely with: %v", err)
	case <-time.After(100 * time.Millisecond):
		// OK
	}

	// Stop it
	stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := w.Stop(stopCtx); err != nil {
		if err != context.DeadlineExceeded {
			t.Fatalf("Stop failed with unexpected error: %v", err)
		}
		// DeadlineExceeded is acceptable for "sleep" on Windows (force exit)
	}

	// Verify Wait closes
	select {
	case err := <-w.Wait():
		if err != nil {
			// On Windows, killing a process might return an error code.
			// On Linux, it might be a signal error.
			// We accept any exit as long as it returned.
			t.Logf("Worker exited with: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Wait blocked after Stop")
	}

	// Verify stopped state
	finalState := w.State()
	if finalState.Status != worker.StatusStopped && finalState.Status != worker.StatusFailed {
		t.Errorf("Expected Stopped or Failed status, got %s", finalState.Status)
	}
}

func TestProcess_Wait(t *testing.T) {
	// Worker that exits immediately (echo)
	w := worker.NewProcessWorker("test-echo", os.Args[0], "echo", "hello")

	w.SetEnv(HelperProcess, "1")

	if err := w.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait should return (nil error usually for code 0)
	select {
	case err := <-w.Wait():
		if err != nil {
			t.Errorf("Wait returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Wait timeout")
	}
}
