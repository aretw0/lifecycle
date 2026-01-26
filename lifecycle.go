package lifecycle

import (
	"context"
	"io"

	"github.com/aretw0/lifecycle/pkg/signal"
	"github.com/aretw0/lifecycle/pkg/termio"
)

// NewSignalContext creates a context that cancels on SIGTERM but captures SIGINT.
// Alias for pkg/signal.NewContext.
func NewSignalContext(parent context.Context) *signal.Context {
	return signal.NewContext(parent)
}

// OpenTerminal checks for text input capability and returns a Reader.
// On Windows, it tries to open CONIN$. Alias for pkg/termio.Open.
func OpenTerminal() (io.ReadCloser, error) {
	return termio.Open()
}

// NewInterruptibleReader returns a reader that checks the cancel channel before/after blocking reads.
// Alias for pkg/termio.NewInterruptibleReader.
func NewInterruptibleReader(base io.Reader, cancel <-chan struct{}) *termio.InterruptibleReader {
	return termio.NewInterruptibleReader(base, cancel)
}

// IsInterrupted checks if an error indicates an interruption (Context Canceled, EOF, etc.).
// Alias for pkg/termio.IsInterrupted.
func IsInterrupted(err error) bool {
	return termio.IsInterrupted(err)
}
