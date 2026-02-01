// Package control implements the Event-Driven Control Plane for the lifecycle library.
//
// It defines the core interfaces (Event, Source, Handler) and the Router component
// that decouples event production from event consumption.
//
// The Control Plane allows applications to react dynamically to system stimuli
// (signals, webhooks, file changes) rather than just shutting down.
package control
