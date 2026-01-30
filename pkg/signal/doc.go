// Package signal provides a stateful signal context with introspection and LIFO hooks.
//
// It solves two main problems for CLI applications:
//  1. Distinguishing between SIGINT (soft interrupt) and SIGTERM (hard termination).
//  2. Managing cleanup operations (Hooks) in a reliable, observable way.
//
// # Dual Signal Handling
//
// Unlike standard signal.NotifyContext, this package allows configuring SIGINT to
// initiate a graceful shutdown logic rather than immediate cancellation. Repeated signals
// can trigger a "Force Exit" (os.Exit(1)) to prevent hung processes.
//
// # Lifecycle Hooks
//
// Users can register cleanup functions via [Context.OnShutdown]. These hooks execution
// is guaranteed to run in LIFO (Last-In-First-Out) order, simulating `defer`.
//
// # Introspection & Visualization
//
// The package implements an "Introspection Pattern". The [Context.State] method returns
// a complete snapshot of the configuration. The [Mermaid] function can consume this
// state to generate a visual representation of the lifecycle policy for documentation.
// The [MermaidState] function can consume this state.
package signal
