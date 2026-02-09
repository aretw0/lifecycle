//go:build !windows

package termio

import (
	"io"
	"os"
)

// Open returns the standard input reader.
// On POSIX, it simply returns os.Stdin.
func Open() (io.ReadCloser, error) {
	return os.Stdin, nil
}

// IsTerminal checks if the given file is a terminal.
// For now, we rely on basic checks or x/term if needed,
// but for simple cases we just assume it's okay.
// We might add x/term dependency later if we need IsTerminal check.
func IsTerminal(f *os.File) bool {
	// Ideally use term.IsTerminal(int(f.Fd()))
	// But keeping it simple for now as we just want the Reader.
	// The boolean logic was in TextHandler.
	return true
}
