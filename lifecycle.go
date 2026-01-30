package lifecycle

import (
	"context"
	"io"
	"os/exec"
	"time"

	"log/slog"

	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
	"github.com/aretw0/lifecycle/pkg/proc"
	"github.com/aretw0/lifecycle/pkg/runtime"
	"github.com/aretw0/lifecycle/pkg/signal"
	"github.com/aretw0/lifecycle/pkg/termio"
	"github.com/aretw0/lifecycle/pkg/worker"
)

// NewSignalContext creates a context that cancels on SIGTERM/SIGINT.
// On the first signal, context is cancelled. On the second, it force exits.
// Behavior can be customized via functional options.
// Alias for pkg/signal.NewContext.
func NewSignalContext(parent context.Context, opts ...signal.Option) *signal.Context {
	return signal.NewContext(parent, opts...)
}

// WithInterrupt configures whether SIGINT (Ctrl+C) should cancel the context.
// Alias for pkg/signal.WithInterrupt.
func WithInterrupt(cancel bool) signal.Option {
	return signal.WithInterrupt(cancel)
}

// WithForceExit configures the threshold of signals required to trigger an immediate os.Exit(1).
// Set to 0 to disable forced exit. Alias for pkg/signal.WithForceExit.
func WithForceExit(threshold int) signal.Option {
	return signal.WithForceExit(threshold)
}

// WithHookTimeout configures the duration after which a running hook produces a warning log.
// Alias for pkg/signal.WithHookTimeout.
func WithHookTimeout(d time.Duration) signal.Option {
	return signal.WithHookTimeout(d)
}

// OpenTerminal checks for text input capability and returns a Reader.
// On Windows, it tries to open CONIN$. Alias for pkg/termio.Open.
func OpenTerminal() (io.ReadCloser, error) {
	return termio.Open()
}

// NewInterruptibleReader returns a reader that checks the cancel channel before/after blocking reads.
// Alias for pkg/termio.NewInterruptibleReader.
func NewInterruptibleReader(base io.Reader, cancel <-chan struct{}) *termio.InterruptibleReader {
	return termio.NewInterruptibleReader(base, cancel)
}

// IsInterrupted checks if an error indicates an interruption (Context Canceled, EOF, etc.).
// Alias for pkg/termio.IsInterrupted.
func IsInterrupted(err error) bool {
	return termio.IsInterrupted(err)
}

// UpgradeTerminal checks if the provided reader is a terminal and returns a safe reader (e.g. CONIN$ on Windows).
// If not a terminal, returns the original reader.
func UpgradeTerminal(r io.Reader) (io.Reader, error) {
	return termio.Upgrade(r)
}

// BlockWithTimeout blocks until the done channel is closed or the timeout expires.
// Alias for pkg/runtime.BlockWithTimeout.
func BlockWithTimeout(done <-chan struct{}, timeout time.Duration) error {
	return runtime.BlockWithTimeout(done, timeout)
}

// StartProcess starts the specified command with process hygiene (auto-kill on parent exit).
// Alias for pkg/proc.Start.
func StartProcess(cmd *exec.Cmd) error {
	return proc.Start(cmd)
}

// SetStrictMode sets whether to block on unsupported platforms for process hygiene.
// Alias for pkg/proc.StrictMode.
func SetStrictMode(strict bool) {
	proc.StrictMode = strict
}

// SetLogger overrides the global logger used by the library.
// Alias for pkg/log.SetLogger.
func SetLogger(l *slog.Logger) {
	log.SetLogger(l)
}

// SetMetricsProvider overrides the global metrics provider.
// This allowing bridging library metrics to Prometheus, OTEL, etc.
// Alias for pkg/metrics.SetProvider.
func SetMetricsProvider(p metrics.Provider) {
	metrics.SetProvider(p)
}

// NewLogMetricsProvider returns a metrics provider that logs to the current logger.
// Useful for development and local verification.
// Alias for pkg/metrics.LogProvider.
func NewLogMetricsProvider() metrics.Provider {
	return &metrics.LogProvider{}
}

// State represents the configuration state of the SignalContext.
// Alias for pkg/signal.State.
type State = signal.State

// Mermaid returns a Mermaid state diagram string representing the lifecycle configuration.
// Alias for pkg/signal.Mermaid.
func Mermaid(s State) string {
	return signal.Mermaid(s)
}

// Worker defines the interface for a managed unit of work.
// Alias for pkg/worker.Worker.
type Worker = worker.Worker

// NewProcessWorker creates a new Process worker for the given command.
// Alias for pkg/worker.NewProcess.
func NewProcessWorker(name string, nameCmd string, args ...string) Worker {
	return worker.NewProcess(name, nameCmd, args...)
}
