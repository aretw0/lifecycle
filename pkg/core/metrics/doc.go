// Package metrics provides a decoupled interface for collecting library metrics.
//
// By using the Provider interface, the lifecycle library remains free of external
// monitoring dependencies (like Prometheus or OpenTelemetry) while still providing
// hook points for observability.
//
// Consumers can implement their own Provider or use the provided LogProvider
// for debugging.
package metrics



