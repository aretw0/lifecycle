package proc_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/proc"
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

func runHelper() {
	mode := os.Args[1] // parent or child
	switch mode {
	case "child":
		// Child just runs for a while
		fmt.Println("Child: running")
		time.Sleep(1 * time.Hour)
	case "parent":
		// Parent spawns child using proc.Start, prints child PID, then exits
		args := []string{"child"}
		cmd := exec.Command(os.Args[0], args...)
		cmd.Env = append(os.Environ(), HelperProcess+"=1")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		fmt.Println("Parent: starting child")
		if err := proc.Start(cmd); err != nil {
			fmt.Fprintf(os.Stderr, "Parent: failed to start child: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("CHILD_PID:%d\n", cmd.Process.Pid)
		// Parent exits immediately
		fmt.Println("Parent: exiting")
		os.Exit(0)
	}
}

func TestStart(t *testing.T) {
	// This test verifies that proc.Start works (starts the process).
	// It does NOT strictly verify the "kill on exit" behavior because that requires
	// an external observer that outlives the test process, which is complex to orchestrate reliably in 'go test'.
	// However, we verify it doesn't crash and returns a valid process.

	cmd := exec.Command(os.Args[0], "child")
	cmd.Env = append(os.Environ(), HelperProcess+"=1")

	err := proc.Start(cmd)
	if err != nil {
		t.Fatalf("proc.Start failed: %v", err)
	}

	if cmd.Process == nil {
		t.Fatal("cmd.Process is nil after proc.Start")
	}

	// Clean up
	cmd.Process.Kill()
}
