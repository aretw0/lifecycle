package termio

import (
	"io"
	"os"

	"golang.org/x/term"
)

// Upgrade checks if the provided reader is a file-based terminal.
// If it is, it upgrades it to a safe terminal reader (like CONIN$ on Windows) using Open().
// If it is not (e.g. pipe, file, buffer), it returns the original reader.
func Upgrade(r io.Reader) (io.Reader, error) {
	if f, ok := r.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		// Found a terminal file (e.g. os.Stdin).
		// Attempt to "re-open" it using the safe platform-specific Open() method.
		// On Windows, this swaps os.Stdin for CONIN$.
		// On POSIX, it just returns os.Stdin (or similar) effectively doing nothing but safe.
		return Open() // Open returns (io.ReadCloser, error)
	}
	return r, nil
}
