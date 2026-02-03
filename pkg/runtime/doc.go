// Package runtime provides utilities for deterministic process management and boilerplate reduction.
//
// It focuses on:
//  1. Preventing hangs during shutdown (timeouts).
//  2. Standardizing application entry points (Run).
//  3. Providing context-aware primitives (Sleep).
//  4. Enabling Critical Sections via DoDetached (for durable operations).
//
// # Configuration
//
// The Run function accepts generic options to configure the runtime environment:
//
//   - WithLogger(l): Sets the global logger.
//   - WithMetrics(p): Sets the global metrics provider.
//   - signal.Option: Configures the SignalContext (e.g. WithForceExit).
package runtime
