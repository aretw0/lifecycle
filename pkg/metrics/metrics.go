package metrics

import (
	"sync"
	"time"

	"github.com/aretw0/lifecycle/pkg/log"
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
	IncDataLost(bytes int)
	ObserveShutdownDuration(workerType string, duration time.Duration)
	IncForceExitTriggered()
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
func (n *NoOpProvider) IncDataLost(bytes int)                              {}
func (n *NoOpProvider) ObserveShutdownDuration(wt string, d time.Duration) {}
func (n *NoOpProvider) IncForceExitTriggered()                             {}

// LogProvider is a metrics provider that logs increments at Debug level.
// This is useful for development and debugging without external dependencies.
type LogProvider struct{}

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

func (l *LogProvider) IncDataLost(bytes int) {
	log.Debug("metric incremented", "name", "lifecycle_data_lost_bytes_total", "bytes", bytes)
}

func (l *LogProvider) ObserveShutdownDuration(wt string, d time.Duration) {
	log.Debug("metric observed", "name", "lifecycle_worker_shutdown_duration_seconds", "type", wt, "value", d.Seconds())
}

func (l *LogProvider) IncForceExitTriggered() {
	log.Debug("metric incremented", "name", "lifecycle_force_exit_triggered_total")
}
