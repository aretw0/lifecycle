package termio

import (
	"context"
	"errors"
	"io"
)

var ErrInterrupted = errors.New("interrupted")

// InterruptibleReader wraps an io.Reader and checks for cancellation before and after reads.
// Note: The underlying Read() call may still block! This wrapper primarily ensures that
// if the context is cancelled *before* we read, we return immediately, and if cancelled
// *during* the read (and the read returns), we prioritize the cancellation error.
type InterruptibleReader struct {
	base   io.Reader
	cancel <-chan struct{}
}

// NewInterruptibleReader returns a reader that checks the cancel channel.
func NewInterruptibleReader(base io.Reader, cancel <-chan struct{}) *InterruptibleReader {
	return &InterruptibleReader{
		base:   base,
		cancel: cancel,
	}
}

func (r *InterruptibleReader) Read(p []byte) (n int, err error) {
	// Check before blocking
	select {
	case <-r.cancel:
		return 0, ErrInterrupted
	default:
	}

	// Read (This blocks!)
	n, err = r.base.Read(p)

	// Check after returning
	select {
	case <-r.cancel:
		return 0, ErrInterrupted
	default:
	}
	return n, err
}

// IsInterrupted checks if the error is related to an interruption (Context Canceled, ErrInterrupted, or EOF).
func IsInterrupted(err error) bool {
	if err == nil {
		return false
	}
	// errors.Is already unwraps the error chain
	if errors.Is(err, ErrInterrupted) || errors.Is(err, context.Canceled) {
		return true
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	// Fallback for string-based errors (only shallow check)
	if err.Error() == "interrupted" {
		return true
	}
	return false
}
