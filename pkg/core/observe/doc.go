// Package observe provides a lightweight observer facade for logs and key process events.
//
// The observer is optional. If no observer is set, the library uses its built-in
// logger and metrics providers. When an observer is set, log calls are delegated
// to it, allowing fully custom telemetry without coupling to slog.
package observe
