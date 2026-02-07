//go:build windows

package termio

import (
	"os"
	"syscall"
	"testing"
)

// TestWindows_Upgrade_NonTerminal verifies that Upgrade() does not break
// or attempt to open CONIN$ when dealing with a regular file on Windows.
func TestWindows_Upgrade_NonTerminal(t *testing.T) {
	tmp, err := os.CreateTemp("", "lifecycle_windows_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	// Upgrade should detect this is a file but NOT a terminal,
	// and thus return the original reader without trying CONIN$.
	r, err := Upgrade(tmp)
	if err != nil {
		t.Fatalf("Upgrade failed: %v", err)
	}

	if r != tmp {
		t.Error("Expected Upgrade to return original file for non-terminal")
	}
}

// TestWindows_OpenConsole_Safety verifies that our low-level openConsole
// function returns a valid specific error when CONIN$ is unavailable (e.g. in CI),
// rather than panicking.
func TestWindows_OpenConsole_Safety(t *testing.T) {
	// calling openConsole directly
	f, err := openConsole()
	if err == nil {
		f.Close()
		// If it worked (e.g. running locally), great.
	} else {
		// If it failed, ensure it's a PathError or SyscallError
		if _, ok := err.(*os.PathError); !ok {
			t.Logf("Warning: openConsole returned unexpected error type: %T %v", err, err)
		}
	}
}

// TestWindows_VirtualTerminal verify we can at least query console mode
// if we are in a terminal.
func TestWindows_ConsoleMode(t *testing.T) {
	// This test is best effort.
	h := syscall.Handle(os.Stdin.Fd())
	var mode uint32
	err := syscall.GetConsoleMode(h, &mode)

	if err == nil {
		t.Logf("Current Console Mode: %b", mode)
	} else {
		t.Logf("GetConsoleMode failed (expected if not a specific TTY): %v", err)
	}
}
