// Package signal provides a stateful signal context for interactive CLI applications.
//
// Unlike the standard signal.NotifyContext which cancels on the first signal,
// this package distinguishes between SIGINT (soft interrupt) and SIGTERM (hard stop).
//
// This allows applications to implement patterns like "Press Ctrl+C again to exit"
// or to prompt for confirmation before shutting down.
package signal
