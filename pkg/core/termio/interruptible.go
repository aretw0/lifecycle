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
		// "Data First, Error Second" strategy:
		// If we actually read something, we return it NOW and nil error.
		// The NEXT call to Read() will find the cancellation and return ErrInterrupted.
		if n > 0 {
			return n, nil
		}
		return 0, ErrInterrupted
	default:
	}
	return n, err
}

// ReadInteractive reads from the underlying source but enforces a "Strict Cancel" policy.
// Unlike Read() (which prioritizes Data over Error to prevent data loss), ReadInteractive
// prioritizes the Cancellation Error over Data.
//
// If the context is cancelled while reading (or immediately after), any data read from
// the OS buffer is DISCARDED, and ErrInterrupted is returned.
//
// Use this for interactive prompts (e.g. "Do you want to continue? [y/N]") where a
// User Interrupt (Ctrl+C) should always take precedence over the input "y", preventing
// accidental execution of dangerous actions.
func (r *InterruptibleReader) ReadInteractive(p []byte) (n int, err error) {
	// Standard read first (which respects pre-cancellation)
	n, err = r.Read(p)

	// Post-Read Strict Check:
	// If the context is cancelled, we MUST discard the result to honor the
	// user's intent to "Cancel", even if they managed to type a character along with it.
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
