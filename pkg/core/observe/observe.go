package observe

import "sync"

// Observer allows external packages to plug in observability (logs and process events)
// without coupling to specific implementations.
//
// Stable as of v1.6.0.
// Implementations should avoid calling the lifecycle log package directly to prevent
// recursive logging loops.
//
// OnGoroutinePanicked hook (v1.6.0): Invoked when a background task panics.
// Stack bytes are optional (depends on WithStackCapture configuration).
// Behavior details: See docs/LIMITATIONS.md (API Stability section) and docs/TECHNICAL.md §14.
type Observer interface {
	OnProcessStarted(pid int)
	OnProcessFailed(err error)
	OnGoroutinePanicked(recovered any, stack []byte)
	LogDebug(msg string, args ...any)
	LogInfo(msg string, args ...any)
	LogWarn(msg string, args ...any)
	LogError(msg string, args ...any)
}

var (
	observerMu sync.RWMutex
	observer   Observer
)

// SetObserver configures the global observer.
//
// Passing nil disables observer routing and falls back to the default logger.
func SetObserver(o Observer) {
	observerMu.Lock()
	defer observerMu.Unlock()
	observer = o
}

// GetObserver returns the current global observer, if any.
func GetObserver() Observer {
	observerMu.RLock()
	defer observerMu.RUnlock()
	return observer
}

// NoOpObserver provides a drop-in observer that does nothing.
type NoOpObserver struct{}

func (NoOpObserver) OnProcessStarted(int)    {}
func (NoOpObserver) OnProcessFailed(error)   {}
func (NoOpObserver) OnGoroutinePanicked(any, []byte) {}
func (NoOpObserver) LogDebug(string, ...any) {}
func (NoOpObserver) LogInfo(string, ...any)  {}
func (NoOpObserver) LogWarn(string, ...any)  {}
func (NoOpObserver) LogError(string, ...any) {}
