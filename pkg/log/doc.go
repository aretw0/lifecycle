// Package log provides a lightweight, structured logging interface for the lifecycle library.
// It is built on top of the standard library's log/slog package, ensuring idiomatic
// and efficient logging without external dependencies.
//
// The package supports global logger configuration, allowing consumers of the library
// to inject their own slog.Logger instance to maintain consistency across their application.
//
// Usage:
//
//	import "github.com/aretw0/lifecycle/pkg/log"
//
//	// Set a custom logger (e.g., JSON output)
//	log.SetLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
//
//	// Standard logging calls
//	log.Info("process started", "pid", 1234)
package log
