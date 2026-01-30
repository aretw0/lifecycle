//go:build windows

package termio

import (
	"io"
	"os"

	"golang.org/x/term"

	"github.com/aretw0/lifecycle/pkg/log"
)

// Open returns a suitable reader for the terminal.
// On Windows, it attempts to use CONIN$ to support interruptible reads.
// This ensures that the handle doesn't close prematurely when receiving a Signal,
// allowing SignalContext to process the event before a fatal EOF occurs.
func Open() (io.ReadCloser, error) {
	// Check if Stdin is a terminal
	if term.IsTerminal(int(os.Stdin.Fd())) {
		conin, err := openConsole()
		if err == nil {
			return conin, nil
		}
	}
	return os.Stdin, nil
}

func openConsole() (io.ReadCloser, error) {
	f, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err == nil {
		log.Info("successfully opened Windows CONIN$ for interruptible reads")
	}
	return f, err
}
