package mock

import (
	"sync"
	"time"

	"github.com/aretw0/lifecycle/pkg/metrics"
)

// Provider matches metrics.Provider interface for testing.
// It is safe for concurrent use.
type Provider struct {
	Mu            sync.Mutex
	Signals       []string
	Restarts      map[string]int // Supervisor -> count
	ChildRestarts map[string]int // Child -> count
	Backoffs      map[string]time.Duration

	// Additional counters for verification
	SupervisorAdds    map[string]int
	SupervisorRemoves map[string]int
	WorkerStarts      map[string]int
	WorkerStops       map[string]int
	WorkerFails       map[string]int

	// Goroutine metrics
	GoroutinesStarted  int
	GoroutinesFinished int
	GoroutinesPanicked int
}

// Ensure interface compliance
var _ metrics.Provider = (*Provider)(nil)

func New() *Provider {
	return &Provider{
		Restarts:          make(map[string]int),
		ChildRestarts:     make(map[string]int),
		Backoffs:          make(map[string]time.Duration),
		SupervisorAdds:    make(map[string]int),
		SupervisorRemoves: make(map[string]int),
		WorkerStarts:      make(map[string]int),
		WorkerStops:       make(map[string]int),
		WorkerFails:       make(map[string]int),
	}
}

func (m *Provider) IncSignalReceived(sig string) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.Signals = append(m.Signals, sig)
}

func (m *Provider) IncProcessStarted()              {}
func (m *Provider) IncProcessFailed()               {}
func (m *Provider) IncTerminalUpgrade(success bool) {}
func (m *Provider) IncHookExecuted() {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.Signals = append(m.Signals, "HookExecuted")
}
func (m *Provider) IncHookPanicked()                    {}
func (m *Provider) ObserveHookDuration(d time.Duration) {}
func (m *Provider) IncWorkerStarted(wt string) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.WorkerStarts[wt]++
}
func (m *Provider) IncWorkerStopped(wt string) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.WorkerStops[wt]++
}
func (m *Provider) IncWorkerFailed(wt string) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.WorkerFails[wt]++
}
func (m *Provider) ObserveWorkerDuration(wt string, d time.Duration) {}

func (m *Provider) IncSupervisorRestart(s, strategy string) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.Restarts[s]++
}

func (m *Provider) IncChildRestart(s, c string) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.ChildRestarts[c]++
}

func (m *Provider) ObserveShutdownDuration(wt string, d time.Duration) {}
func (m *Provider) IncForceExitTriggered()                             {}

func (m *Provider) IncContainerStarted(image string)                       {}
func (m *Provider) IncContainerStopped(image string)                       {}
func (m *Provider) IncContainerFailed(image string)                        {}
func (m *Provider) ObserveContainerDuration(image string, d time.Duration) {}

func (m *Provider) IncSupervisorAdd(s string) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.SupervisorAdds[s]++
}

func (m *Provider) IncSupervisorRemove(s string) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.SupervisorRemoves[s]++
}

func (m *Provider) IncBackoffTriggered(c string, d time.Duration) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.Backoffs[c] = d
}

func (m *Provider) IncCriticalSectionStarted()              {}
func (m *Provider) IncCriticalSectionFinished(success bool) {}

func (m *Provider) ObserveCriticalSectionDuration(d time.Duration) {}

func (m *Provider) IncGoroutineStarted() {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.GoroutinesStarted++
}

func (m *Provider) IncGoroutineFinished() {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.GoroutinesFinished++
}

func (m *Provider) IncGoroutinePanicked() {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.GoroutinesPanicked++
}

func (m *Provider) ObserveGoroutineBlockDuration(d time.Duration) {}

func (m *Provider) IncGoroutineWaiting() {
	// For testing, we might want to track this, but simpler to NoOp or just track calls if needed.
}
func (m *Provider) DecGoroutineWaiting() {}
