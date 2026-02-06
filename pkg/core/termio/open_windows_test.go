//go:build windows

package termio

import (
	"io"
	"os"
	"testing"
)

// TestOpenConsoleAvailable attempts to open CONIN$ via openConsole.
// If CONIN$ is not available in the environment, the test will be skipped.
func TestOpenConsoleAvailable(t *testing.T) {
	rc, err := openConsole()
	if err != nil {
		t.Skipf("CONIN$ not available: %v", err)
	}
	defer rc.Close()
	// Ensure we got an io.ReadCloser
	var _ io.ReadCloser = rc
}

// TestOpenReturnsReader verifies Open() returns a non-nil reader and does not panic.
func TestOpenReturnsReader(t *testing.T) {
	// preserve original Stdin
	old := os.Stdin
	defer func() { os.Stdin = old }()

	// Prefer to set Stdin to CONIN$ if possible so Open() may choose console path.
	if f, err := os.OpenFile("CONIN$", os.O_RDWR, 0); err == nil {
		os.Stdin = f
		defer f.Close()
	}

	rc, err := Open()
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}
	if rc == nil {
		t.Fatalf("Open() returned nil reader")
	}
	rc.Close()
}
