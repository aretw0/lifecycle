package metrics

import (
	"sync"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/log"
)

// Provider defines the interface for collecting lifecycle metrics.
// Implementations can bridge these calls to Prometheus, OpenTelemetry, etc.
type Provider interface {
	IncSignalReceived(sig string)
	IncProcessStarted()
	IncProcessFailed()
	IncTerminalUpgrade(success bool)
	IncHookExecuted()
	IncHookPanicked()
	ObserveHookDuration(duration time.Duration)
	IncWorkerStarted(workerType string)
	IncWorkerStopped(workerType string)
	IncWorkerFailed(workerType string)
	ObserveWorkerDuration(workerType string, duration time.Duration)
	IncSupervisorRestart(supervisorName, strategy string)
	IncChildRestart(supervisorName, childName string)
	IncSupervisorAdd(supervisorName string)
	IncSupervisorRemove(supervisorName string)
	IncBackoffTriggered(childName string, delay time.Duration)
	ObserveShutdownDuration(workerType string, duration time.Duration)
	IncForceExitTriggered()
	IncCircuitBreakerTriggered(childName string)

	// Critical Sections
	IncCriticalSectionStarted()
	IncCriticalSectionFinished(success bool)
	ObserveCriticalSectionDuration(duration time.Duration)

	// Container metrics
	IncContainerStarted(image string)
	IncContainerStopped(image string)
	IncContainerFailed(image string)
	ObserveContainerDuration(image string, duration time.Duration)

	// Goroutine metrics (v2.0)
	IncGoroutineStarted()
	IncGoroutineFinished()
	IncGoroutinePanicked()

	ObserveGoroutineBlockDuration(duration time.Duration)

	// Waiting/Backpressure Gauge
	IncGoroutineWaiting()
	DecGoroutineWaiting()

	// Control Plane Metrics (v2.0)
	IncEventEmitted(source string)
	IncEventRouted(topic string)
	IncHandlerExecuted(topic string)
	IncHandlerError(topic string, err error)
	ObserveHandlerDuration(topic string, duration time.Duration)

	// Event Backpressure Metrics (v2.0)
	ObserveEventBlockDuration(source string, duration time.Duration)
	IncEventWaiting(source string)
	DecEventWaiting(source string)
}

var (
	providerMu sync.RWMutex
	provider   Provider = &NoOpProvider{}
)

// SetProvider sets the global metrics provider.
func SetProvider(p Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	if p == nil {
		provider = &NoOpProvider{}
		return
	}
	provider = p
}

// GetProvider returns the current global metrics provider.
func GetProvider() Provider {
	providerMu.RLock()
	defer providerMu.RUnlock()
	return provider
}

// NoOpProvider is a metrics provider that does nothing.
type NoOpProvider struct{}

func (n *NoOpProvider) IncSignalReceived(sig string)                       {}
func (n *NoOpProvider) IncProcessStarted()                                 {}
func (n *NoOpProvider) IncProcessFailed()                                  {}
func (n *NoOpProvider) IncTerminalUpgrade(success bool)                    {}
func (n *NoOpProvider) IncHookExecuted()                                   {}
func (n *NoOpProvider) IncHookPanicked()                                   {}
func (n *NoOpProvider) ObserveHookDuration(d time.Duration)                {}
func (n *NoOpProvider) IncWorkerStarted(wt string)                         {}
func (n *NoOpProvider) IncWorkerStopped(wt string)                         {}
func (n *NoOpProvider) IncWorkerFailed(wt string)                          {}
func (n *NoOpProvider) ObserveWorkerDuration(wt string, d time.Duration)   {}
func (n *NoOpProvider) IncSupervisorRestart(s, est string)                 {}
func (n *NoOpProvider) IncChildRestart(s, c string)                        {}
func (n *NoOpProvider) IncSupervisorAdd(s string)                          {}
func (n *NoOpProvider) IncSupervisorRemove(s string)                       {}
func (n *NoOpProvider) IncBackoffTriggered(c string, d time.Duration)      {}
func (n *NoOpProvider) ObserveShutdownDuration(wt string, d time.Duration) {}
func (n *NoOpProvider) IncForceExitTriggered()                             {}
func (n *NoOpProvider) IncCircuitBreakerTriggered(c string)                {}

func (n *NoOpProvider) IncCriticalSectionStarted()                     {}
func (n *NoOpProvider) IncCriticalSectionFinished(success bool)        {}
func (n *NoOpProvider) ObserveCriticalSectionDuration(d time.Duration) {}

func (n *NoOpProvider) IncContainerStarted(image string)                       {}
func (n *NoOpProvider) IncContainerStopped(image string)                       {}
func (n *NoOpProvider) IncContainerFailed(image string)                        {}
func (n *NoOpProvider) ObserveContainerDuration(image string, d time.Duration) {}

func (n *NoOpProvider) IncGoroutineStarted()  {}
func (n *NoOpProvider) IncGoroutineFinished() {}
func (n *NoOpProvider) IncGoroutinePanicked() {}

func (n *NoOpProvider) ObserveGoroutineBlockDuration(d time.Duration) {}

func (n *NoOpProvider) IncGoroutineWaiting() {}
func (n *NoOpProvider) DecGoroutineWaiting() {}

func (n *NoOpProvider) IncEventEmitted(source string)                        {}
func (n *NoOpProvider) IncEventRouted(topic string)                          {}
func (n *NoOpProvider) IncHandlerExecuted(topic string)                      {}
func (n *NoOpProvider) IncHandlerError(topic string, err error)              {}
func (n *NoOpProvider) ObserveHandlerDuration(topic string, d time.Duration) {}

func (n *NoOpProvider) ObserveEventBlockDuration(source string, d time.Duration) {}
func (n *NoOpProvider) IncEventWaiting(source string)                            {}
func (n *NoOpProvider) DecEventWaiting(source string)                            {}

// LogProvider is a metrics provider that logs increments at Debug level.
// This is useful for development and debugging without external dependencies.
type LogProvider struct{}

func (l *LogProvider) IncEventEmitted(source string) {
	log.Debug("metric incremented", "name", "lifecycle_events_emitted_total", "source", source)
}

func (l *LogProvider) IncEventRouted(topic string) {
	log.Debug("metric incremented", "name", "lifecycle_events_routed_total", "topic", topic)
}

func (l *LogProvider) IncHandlerExecuted(topic string) {
	log.Debug("metric incremented", "name", "lifecycle_handlers_executed_total", "topic", topic)
}

func (l *LogProvider) IncHandlerError(topic string, err error) {
	log.Debug("metric incremented", "name", "lifecycle_handler_errors_total", "topic", topic, "error", err)
}

func (l *LogProvider) ObserveHandlerDuration(topic string, d time.Duration) {
	log.Debug("metric observed", "name", "lifecycle_handler_duration_seconds", "topic", topic, "value", d.Seconds())
}

func (l *LogProvider) IncSignalReceived(sig string) {
	log.Debug("metric incremented", "name", "lifecycle_signals_total", "signal", sig)
}

func (l *LogProvider) IncProcessStarted() {
	log.Debug("metric incremented", "name", "lifecycle_processes_started_total")
}

func (l *LogProvider) IncProcessFailed() {
	log.Debug("metric incremented", "name", "lifecycle_processes_failed_total")
}

func (l *LogProvider) IncTerminalUpgrade(success bool) {
	log.Debug("metric incremented", "name", "lifecycle_terminal_upgrades_total", "success", success)
}

func (l *LogProvider) IncHookExecuted() {
	log.Debug("metric incremented", "name", "lifecycle_hooks_executed_total")
}

func (l *LogProvider) IncHookPanicked() {
	log.Debug("metric incremented", "name", "lifecycle_hooks_panicked_total")
}

func (l *LogProvider) ObserveHookDuration(d time.Duration) {
	log.Debug("metric observed", "name", "lifecycle_hook_duration_seconds", "value", d.Seconds())
}

func (l *LogProvider) IncWorkerStarted(wt string) {
	log.Debug("metric incremented", "name", "lifecycle_workers_started_total", "type", wt)
}

func (l *LogProvider) IncWorkerStopped(wt string) {
	log.Debug("metric incremented", "name", "lifecycle_workers_stopped_total", "type", wt)
}

func (l *LogProvider) IncWorkerFailed(wt string) {
	log.Debug("metric incremented", "name", "lifecycle_workers_failed_total", "type", wt)
}

func (l *LogProvider) ObserveWorkerDuration(wt string, d time.Duration) {
	log.Debug("metric observed", "name", "lifecycle_worker_duration_seconds", "type", wt, "value", d.Seconds())
}

func (l *LogProvider) IncSupervisorRestart(s, strategy string) {
	log.Debug("metric incremented", "name", "lifecycle_supervisor_restarts_total", "supervisor", s, "strategy", strategy)
}

func (l *LogProvider) IncChildRestart(s, c string) {
	log.Debug("metric incremented", "name", "lifecycle_worker_restarts_total", "supervisor", s, "child", c)
}

func (l *LogProvider) IncSupervisorAdd(s string) {
	log.Debug("metric incremented", "name", "lifecycle_supervisor_add_total", "supervisor", s)
}

func (l *LogProvider) IncSupervisorRemove(s string) {
	log.Debug("metric incremented", "name", "lifecycle_supervisor_remove_total", "supervisor", s)
}

func (l *LogProvider) IncBackoffTriggered(c string, d time.Duration) {
	log.Debug("metric observed", "name", "lifecycle_backoff_triggered_seconds", "child", c, "delay", d.Seconds())
}

func (l *LogProvider) ObserveShutdownDuration(wt string, d time.Duration) {
	log.Debug("metric observed", "name", "lifecycle_worker_shutdown_duration_seconds", "type", wt, "value", d.Seconds())
}

func (l *LogProvider) IncForceExitTriggered() {
	log.Debug("metric incremented", "name", "lifecycle_force_exit_triggered_total")
}

func (l *LogProvider) IncCircuitBreakerTriggered(c string) {
	log.Debug("metric incremented", "name", "lifecycle_supervisor_circuit_breaker_triggered_total", "child", c)
}

func (l *LogProvider) IncCriticalSectionStarted() {
	log.Debug("metric incremented", "name", "lifecycle_critical_section_started_total")
}

func (l *LogProvider) IncCriticalSectionFinished(success bool) {
	log.Debug("metric incremented", "name", "lifecycle_critical_section_finished_total", "success", success)
}

func (l *LogProvider) ObserveCriticalSectionDuration(d time.Duration) {
	log.Debug("metric observed", "name", "lifecycle_critical_section_duration_seconds", "value", d.Seconds())
}

func (l *LogProvider) IncContainerStarted(image string) {
	log.Debug("metric incremented", "name", "lifecycle_container_started_total", "image", image)
}

func (l *LogProvider) IncContainerStopped(image string) {
	log.Debug("metric incremented", "name", "lifecycle_container_stopped_total", "image", image)
}

func (l *LogProvider) IncContainerFailed(image string) {
	log.Debug("metric incremented", "name", "lifecycle_container_failed_total", "image", image)
}

func (l *LogProvider) ObserveContainerDuration(image string, d time.Duration) {
	log.Debug("metric observed", "name", "lifecycle_container_duration_seconds", "image", image, "value", d.Seconds())
}

func (l *LogProvider) IncGoroutineStarted() {
	log.Debug("metric incremented", "name", "lifecycle_goroutines_started_total")
}

func (l *LogProvider) IncGoroutineFinished() {
	log.Debug("metric incremented", "name", "lifecycle_goroutines_finished_total")
}

func (l *LogProvider) IncGoroutinePanicked() {
	log.Debug("metric incremented", "name", "lifecycle_goroutines_panicked_total")
}

func (l *LogProvider) ObserveGoroutineBlockDuration(d time.Duration) {
	log.Debug("metric observed", "name", "lifecycle_goroutines_block_duration_seconds", "value", d.Seconds())
}

func (l *LogProvider) IncGoroutineWaiting() {
	log.Debug("metric incremented", "name", "lifecycle_goroutines_waiting_current")
}

func (l *LogProvider) DecGoroutineWaiting() {
	log.Debug("metric decremented", "name", "lifecycle_goroutines_waiting_current")
}

func (l *LogProvider) ObserveEventBlockDuration(source string, d time.Duration) {
	log.Debug("metric observed", "name", "lifecycle_events_block_duration_seconds", "source", source, "value", d.Seconds())
}

func (l *LogProvider) IncEventWaiting(source string) {
	log.Debug("metric incremented", "name", "lifecycle_events_waiting_current", "source", source)
}

func (l *LogProvider) DecEventWaiting(source string) {
	log.Debug("metric decremented", "name", "lifecycle_events_waiting_current", "source", source)
}



