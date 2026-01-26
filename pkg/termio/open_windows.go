//go:build windows

package termio

import (
	"io"
	"os"

	"golang.org/x/term"
)

// Open returns a suitable reader for the terminal.
// On Windows, it attempts to use CONIN$ to support interruptible reads.
// This ensures that the handle doesn't close prematurely when receiving a Signal,
// allowing SignalContext to process the event before a fatal EOF occurs.
func Open() (io.ReadCloser, error) {
	// Check if Stdin is a terminal
	if term.IsTerminal(int(os.Stdin.Fd())) {
		conin, err := os.Open("CONIN$")
		if err == nil {
			return conin, nil
		}
	}
	return os.Stdin, nil
}
