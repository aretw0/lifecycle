package termio

import (
	"io"
	"os"

	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"golang.org/x/term"
)

// Upgrade checks if the provided reader is a file-based terminal.
// If it is, it upgrades it to a safe terminal reader (like CONIN$ on Windows) using Open().
// If it is not (e.g. pipe, file, buffer), it returns the original reader.
func Upgrade(r io.Reader) (io.Reader, error) {
	if f, ok := r.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		log.Debug("upgrading terminal to raw/conin handle", "fd", f.Fd())
		newR, err := Open() // Open returns (io.ReadCloser, error)
		metrics.GetProvider().IncTerminalUpgrade(err == nil)
		return newR, err
	}
	return r, nil
}



