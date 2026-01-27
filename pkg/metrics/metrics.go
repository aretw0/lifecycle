package metrics

import (
	"sync"

	"github.com/aretw0/lifecycle/pkg/log"
)

// Provider defines the interface for collecting lifecycle metrics.
// Implementations can bridge these calls to Prometheus, OpenTelemetry, etc.
type Provider interface {
	IncSignalReceived(sig string)
	IncProcessStarted()
	IncProcessFailed()
	IncTerminalUpgrade(success bool)
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

func (n *NoOpProvider) IncSignalReceived(sig string)    {}
func (n *NoOpProvider) IncProcessStarted()              {}
func (n *NoOpProvider) IncProcessFailed()               {}
func (n *NoOpProvider) IncTerminalUpgrade(success bool) {}

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
