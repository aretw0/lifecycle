package lifecycle

import (
	"context"
	"io"
	"iter"
	"os"
	"os/exec"
	"time"

	"log/slog"

	"github.com/aretw0/lifecycle/pkg/container"
	"github.com/aretw0/lifecycle/pkg/control"
	"github.com/aretw0/lifecycle/pkg/handlers"
	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
	"github.com/aretw0/lifecycle/pkg/proc"
	"github.com/aretw0/lifecycle/pkg/runtime"
	"github.com/aretw0/lifecycle/pkg/signal"
	"github.com/aretw0/lifecycle/pkg/sources"
	"github.com/aretw0/lifecycle/pkg/supervisor"
	"github.com/aretw0/lifecycle/pkg/termio"
	"github.com/aretw0/lifecycle/pkg/worker"
)

// ======================================================================================
// 1. Core Runtime
// ======================================================================================

// Runnable defines a long-running process.
// Alias for pkg/runtime.Runnable.
type Runnable = runtime.Runnable

// Job creates a Runnable from a function.
// Alias for pkg/runtime.Job.
func Job(fn func(context.Context) error) Runnable {
	return runtime.Job(fn)
}

// Run executes the application logic with a managed SignalContext.
// Alias for pkg/runtime.Run.
func Run(r Runnable, opts ...any) error {
	return runtime.Run(r, opts...)
}

// Go starts a tracked goroutine.
// Alias for pkg/runtime.Go.
func Go(ctx context.Context, fn func(context.Context) error) {
	runtime.Go(ctx, fn)
}

// Receive creates a push iterator that yields values from the channel until
// the context is cancelled or the channel is closed.
// Alias for pkg/runtime.Receive.
func Receive[V any](ctx context.Context, ch <-chan V) iter.Seq[V] {
	return runtime.Receive(ctx, ch)
}

// Do executes a function in a "Safe Executor" (Panic Recovery + Observability).
// Alias for pkg/runtime.Do.
func Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return runtime.Do(ctx, fn)
}

// DoDetached executes a function in a "Critical Section" (Detached Context).
// Alias for pkg/runtime.DoDetached.
func DoDetached(parent context.Context, fn func(ctx context.Context) error) error {
	return runtime.DoDetached(parent, fn)
}

// BlockWithTimeout blocks until the done channel is closed or the timeout expires.
// Alias for pkg/runtime.BlockWithTimeout.
func BlockWithTimeout(done <-chan struct{}, timeout time.Duration) error {
	return runtime.BlockWithTimeout(done, timeout)
}

// Sleep pauses the current goroutine for at least the duration d.
// Alias for pkg/runtime.Sleep.
func Sleep(ctx context.Context, d time.Duration) error {
	return runtime.Sleep(ctx, d)
}

// ======================================================================================
// 2. Signal Context
// ======================================================================================

// Context represents the signal context.
type Context = signal.Context

// Option is a functional option for signal configuration.
// Alias for pkg/signal.Option.
type SignalOption = signal.Option

// SignalState returns a snapshot of the current configuration.
// Alias for signal.State.
type SignalState = signal.State

// NewSignalContext creates a context that cancels on SIGTERM/SIGINT.
// On the first signal, context is cancelled. On the second, it force exits.
// Behavior can be customized via functional options.
// Alias for pkg/signal.NewContext.
func NewSignalContext(parent context.Context, opts ...signal.Option) *signal.Context {
	return signal.NewContext(parent, opts...)
}

// WithForceExit configures the threshold of signals required to trigger an immediate os.Exit(1).
// Threshold values:
// 1 (Default): SIGINT cancels context immediately.
// n >= 2: SIGINT is captured, os.Exit(1) at n-th signal (Escalation Mode).
// 0 (Unsafe): Automatic os.Exit(1) is disabled for SIGINT.
func WithForceExit(threshold int) signal.Option {
	return signal.WithForceExit(threshold)
}

// WithResetTimeout configures the duration after which the signal count resets.
// Alias for pkg/signal.WithResetTimeout.
func WithResetTimeout(d time.Duration) signal.Option {
	return signal.WithResetTimeout(d)
}

// WithHookTimeout configures the duration after which a running hook produces a warning log.
// Alias for pkg/signal.WithHookTimeout.
func WithHookTimeout(d time.Duration) signal.Option {
	return signal.WithHookTimeout(d)
}

// OnShutdown safely registers a shutdown hook on the context if it supports it.
// It abstracts the type assertion for *signal.Context.
func OnShutdown(ctx context.Context, fn func()) {
	if sc, ok := ctx.(*signal.Context); ok {
		sc.OnShutdown(fn)
	}
}

// IsUnsafe returns true if the context is configured to never force exit.
func IsUnsafe(ctx context.Context) bool {
	if sc, ok := ctx.(*signal.Context); ok {
		return sc.IsUnsafe()
	}
	return false
}

// GetForceExitThreshold returns the number of signals required to trigger os.Exit(1).
func GetForceExitThreshold(ctx context.Context) int {
	if sc, ok := ctx.(*signal.Context); ok {
		return sc.ForceExitThreshold()
	}
	return 0
}

// GetSignalState returns a snapshot of the context's signal configuration.
func GetSignalState(ctx context.Context) (SignalState, bool) {
	if sc, ok := ctx.(*signal.Context); ok {
		return sc.State(), true
	}
	return SignalState{}, false
}

// SignalStateDiagram returns a Mermaid state diagram string representing the signal context configuration.
// Alias for pkg/signal.MermaidState.
func SignalStateDiagram(s SignalState) string {
	return signal.MermaidState(s)
}

// ======================================================================================
// 3. Workers & Supervisor
// ======================================================================================

// Worker defines the interface for a managed unit of work.
// Alias for pkg/worker.Worker.
type Worker = worker.Worker

// Suspendable defines a worker that can pause its execution in-place without exiting.
// Alias for pkg/worker.Suspendable.
type Suspendable = worker.Suspendable

// WorkerState represents the snapshot of a worker's state.
type WorkerState = worker.State

// WorkerStatus represents the lifecycle state of a worker.
type WorkerStatus = worker.Status

const (
	WorkerStatusPending = worker.StatusPending
	WorkerStatusRunning = worker.StatusRunning
	WorkerStatusStopped = worker.StatusStopped
	WorkerStatusFailed  = worker.StatusFailed
)

// NewProcessWorker creates a new Process worker for the given command.
// Alias for pkg/worker.NewProcessWorker.
func NewProcessWorker(name string, nameCmd string, args ...string) Worker {
	return worker.NewProcessWorker(name, nameCmd, args...)
}

// NewWorkerFromFunc creates a Worker from a function.
// Alias for pkg/worker.FromFunc.
func NewWorkerFromFunc(name string, fn func(context.Context) error) Worker {
	return worker.FromFunc(name, fn)
}

// Container represents a generic container interface.
// Alias for container.Container.
type Container = container.Container

// ContainerStatus represents the lifecycle state of a container.
// Alias for container.Status.
type ContainerStatus = container.Status

// NewContainerWorker creates a new Worker from a Container interface.
// Alias for pkg/worker.NewContainerWorker.
func NewContainerWorker(name string, c Container) Worker {
	return worker.NewContainerWorker(name, c)
}

// NewMockContainer creates a new MockContainer for testing.
// Alias for pkg/container.NewMockContainer.
func NewMockContainer(id string) Container {
	return container.NewMockContainer(id)
}

// Handover Constants
const (
	// WorkerEnvResumeID is the unique session identifier for a worker.
	WorkerEnvResumeID = worker.EnvResumeID
	// WorkerEnvPrevExit is the exit code of the previous execution of this worker.
	WorkerEnvPrevExit = worker.EnvPrevExit
)

// Supervisor defines the interface for a supervisor.
// Alias for pkg/supervisor.Supervisor.
type Supervisor = supervisor.Supervisor

// SupervisorStrategy defines how the supervisor handles child failures.
// Alias for pkg/supervisor.Strategy.
type SupervisorStrategy = supervisor.Strategy

const (
	// SupervisorStrategyOneForOne: If a child process terminates, only that process is restarted.
	SupervisorStrategyOneForOne = supervisor.StrategyOneForOne
	// SupervisorStrategyOneForAll: If a child process terminates, all other child processes are terminated.
	SupervisorStrategyOneForAll = supervisor.StrategyOneForAll
)

// SupervisorSpec defines the configuration for a supervised child worker.
// Alias for pkg/supervisor.Spec.
type SupervisorSpec = supervisor.Spec

// SupervisorBackoff defines the retry policy for failed children.
// Alias for pkg/supervisor.Backoff.
type SupervisorBackoff = supervisor.Backoff

// SupervisorRestartPolicy defines when a child worker should be restarted.
// Alias for pkg/supervisor.RestartPolicy.
type SupervisorRestartPolicy = supervisor.RestartPolicy

const (
	RestartAlways    = supervisor.RestartAlways
	RestartOnFailure = supervisor.RestartOnFailure
	RestartNever     = supervisor.RestartNever
)

// SupervisorFactory is a function that creates a new worker instance.
// Alias for pkg/supervisor.Factory.
type SupervisorFactory = supervisor.Factory

// NewSupervisor creates a new Supervisor for the given workers.
// Alias for pkg/supervisor.New.
func NewSupervisor(name string, strategy SupervisorStrategy, specs ...SupervisorSpec) Supervisor {
	return supervisor.New(name, strategy, specs...)
}

// WorkerTreeDiagram returns a Mermaid diagram string representing the worker structure (Tree).
// Alias for pkg/worker.MermaidTree.
func WorkerTreeDiagram(s WorkerState) string {
	return worker.MermaidTree(s)
}

// WorkerStateDiagram returns a Mermaid state diagram string representing the worker state transitions.
// Alias for pkg/worker.MermaidState.
func WorkerStateDiagram(s WorkerState) string {
	return worker.MermaidState(s)
}

// ======================================================================================
// 4. Control Plane (v2)
// ======================================================================================

// Event is a stimulus that triggers a reaction.
// Alias for pkg/control.Event.
type Event = control.Event

// Handler responds to an event.
// Alias for pkg/control.Handler.
type Handler = control.Handler

// HandlerFunc matches the signature of a Handler.
// Alias for pkg/control.HandlerFunc.
type HandlerFunc = control.HandlerFunc

// Source is a producer of events.
// Alias for pkg/control.Source.
type Source = control.Source

// Router maps events to reactions.
// Alias for pkg/control.Router.
type Router = control.Router

// RouterOption is an option for configuring a Router.
// Alias for pkg/control.RouterOption.
type RouterOption = control.RouterOption

// NewRouter creates a new Control Router.
// Alias for pkg/control.NewRouter.
func NewRouter(opts ...RouterOption) *Router {
	return control.NewRouter(opts...)
}

// DefaultRouter is the default instance for package-level helpers.
// Alias for pkg/control.DefaultRouter.
var DefaultRouter = control.DefaultRouter

// Handle registers a handler on the DefaultRouter.
// Alias for pkg/control.Handle.
func Handle(pattern string, handler Handler) {
	control.Handle(pattern, handler)
}

// HandleFunc registers a handler function on the DefaultRouter.
// Alias for pkg/control.HandleFunc.
func HandleFunc(pattern string, handler func(context.Context, Event) error) {
	control.HandleFunc(pattern, handler)
}

// ======================================================================================
// 5. Sources (Event Producers)
// ======================================================================================

// NewOSSignalSource creates a source that listens for OS signals.
// Alias for pkg/sources.NewOSSignalSource.
func NewOSSignalSource(signals ...os.Signal) Source {
	return sources.NewOSSignalSource(signals...)
}

// InputSource reads from Stdin and emits events.
// Alias for pkg/sources.InputSource.
type InputSource = sources.InputSource

// InputOption configures the InputSource.
// Alias for pkg/sources.InputOption.
type InputOption = sources.InputOption

// NewInputSource creates a new source that listens for standard CLI commands.
// Alias for pkg/sources.NewInputSource.
func NewInputSource(opts ...InputOption) *InputSource {
	return sources.NewInputSource(opts...)
}

// WithUnknownHandler configures a custom handler for unknown commands.
// Alias for pkg/sources.WithUnknownHandler.
func WithUnknownHandler(fn func(cmd string, known []string)) InputOption {
	return sources.WithUnknownHandler(fn)
}

// WithInputMapping adds a custom command mapping.
// Alias for pkg/sources.WithInputMapping.
func WithInputMapping(key string, event control.Event) InputOption {
	return sources.WithInputMapping(key, event)
}

// WithInputBackoff configures the duration to wait before retrying interruptions or errors.
// Alias for pkg/sources.WithInputBackoff.
func WithInputBackoff(d time.Duration) InputOption {
	return sources.WithInputBackoff(d)
}

// InputEvent represents a generic text command.
// Alias for pkg/sources.InputEvent.
type InputEvent = sources.InputEvent

// TickEvent represents a periodic time tick.
// Alias for pkg/sources.TickEvent.
type TickEvent = sources.TickEvent

// NewTickerSource creates a source that emits periodic events.
// Alias for pkg/sources.NewTickerSource.
func NewTickerSource(interval time.Duration) Source {
	return sources.NewTickerSource(interval)
}

// HealthCheckSource runs a periodic health check.
// Alias for pkg/sources.HealthCheckSource.
type HealthCheckSource = sources.HealthCheckSource

// HealthOption configures a HealthCheckSource.
// Alias for pkg/sources.HealthOption.
type HealthOption = sources.HealthOption

// NewHealthCheckSource creates a new health monitor.
// Alias for pkg/sources.NewHealthCheckSource.
func NewHealthCheckSource(name string, check sources.CheckFunc, opts ...HealthOption) *HealthCheckSource {
	return sources.NewHealthCheckSource(name, check, opts...)
}

// WithHealthInterval sets the check interval.
// Alias for pkg/sources.WithHealthInterval.
func WithHealthInterval(d time.Duration) HealthOption {
	return sources.WithHealthInterval(d)
}

// WithHealthStrategy sets the triggering strategy (Edge vs Level).
// Alias for pkg/sources.WithHealthStrategy.
func WithHealthStrategy(strategy sources.TriggerStrategy) HealthOption {
	return sources.WithHealthStrategy(strategy)
}

// FileWatchSource watches for file changes via polling.
// Alias for pkg/sources.FileWatchSource.
type FileWatchSource = sources.FileWatchSource

// NewFileWatchSource creates a new source that polls the given file path.
// Alias for pkg/sources.NewFileWatchSource.
func NewFileWatchSource(path string, interval time.Duration) *FileWatchSource {
	return sources.NewFileWatchSource(path, interval)
}

// NewWebhookSource creates a source that listens for Webhooks.
// Alias for pkg/sources.NewWebhookSource.
func NewWebhookSource(addr string) *sources.WebhookSource {
	return sources.NewWebhookSource(addr)
}

// ChannelSource adapts a generic Go channel to the Source interface.
// Alias for pkg/sources.ChannelSource.
type ChannelSource = sources.ChannelSource

// NewChannelSource creates a new source that reads from the given channel.
// Alias for pkg/sources.NewChannelSource.
func NewChannelSource(ch <-chan Event) *ChannelSource {
	return sources.NewChannelSource(ch)
}

// ======================================================================================
// 6. Handlers (Event Reactions)
// ======================================================================================

// ShutdownEvent is triggered when the application should shut down gracefully.
// Alias for pkg/control.ShutdownEvent.
type ShutdownEvent = control.ShutdownEvent

// NewShutdownHandler returns a handler that cancels context.
// Alias for pkg/handlers.NewShutdown.
func NewShutdownHandler(cancel context.CancelFunc) Handler {
	return handlers.NewShutdown(cancel)
}

// NewReloadHandler returns a handler that reloads configuration.
// Alias for pkg/handlers.NewReload.
func NewReloadHandler(onReload func(context.Context) error) Handler {
	return handlers.NewReload(onReload)
}

// SuspendHandler manages Suspend and Resume events.
// Alias for pkg/handlers.SuspendHandler.
type SuspendHandler = handlers.SuspendHandler

// SuspendEvent is the event that triggers a suspension.
// Alias for pkg/control.SuspendEvent.
type SuspendEvent = control.SuspendEvent

// ResumeEvent is the event that triggers a resumption.
// Alias for pkg/control.ResumeEvent.
type ResumeEvent = control.ResumeEvent

// NewSuspendHandler creates a new handler for suspend/resume events.
// Alias for pkg/handlers.NewSuspendHandler.
func NewSuspendHandler() *SuspendHandler {
	return handlers.NewSuspendHandler()
}

// SmartSignalHandler implements "Double-Tap" signal logic (Suspend -> Quit).
// Alias for pkg/handlers.SmartSignalHandler.
type SmartSignalHandler = handlers.SmartSignalHandler

// NewSmartSignalHandler creates a new smart signal handler.
// Alias for pkg/handlers.NewSmartSignalHandler.
func NewSmartSignalHandler(s *SuspendHandler, q Handler) *SmartSignalHandler {
	return handlers.NewSmartSignalHandler(s, q)
}

// ClearLineEvent is triggered when an interactive input is interrupted.
// Alias for pkg/control.ClearLineEvent.
type ClearLineEvent = control.ClearLineEvent

// ======================================================================================
// 6a. Interactive Router Helpers
// ======================================================================================

// See interactive.go for NewInteractiveRouter and related options.

// ======================================================================================
// 7. Process Management (Low Level)
// ======================================================================================

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

// ======================================================================================
// 8. Terminal & I/O
// ======================================================================================

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

// ======================================================================================
// 9. Observability & Metrics
// ======================================================================================

// SetLogger overrides the global logger used by the library.
// Alias for pkg/log.SetLogger.
func SetLogger(l *slog.Logger) {
	log.SetLogger(l)
}

// WithLogger returns a RunOption to configure the global logger.
// Alias for pkg/runtime.WithLogger.
func WithLogger(l *slog.Logger) any {
	return runtime.WithLogger(l)
}

// SetMetricsProvider overrides the global metrics provider.
// This allowing bridging library metrics to Prometheus, OTEL, etc.
// Alias for pkg/metrics.SetProvider.
func SetMetricsProvider(p metrics.Provider) {
	metrics.SetProvider(p)
}

// WithMetrics returns a RunOption to configure the global metrics provider.
// Alias for pkg/runtime.WithMetrics.
func WithMetrics(p metrics.Provider) any {
	return runtime.WithMetrics(p)
}

// WithShutdownTimeout returns a RunOption to configure the diagnostic timeout during shutdown.
// Alias for pkg/runtime.WithShutdownTimeout.
func WithShutdownTimeout(d time.Duration) any {
	return runtime.WithShutdownTimeout(d)
}

// NewLogMetricsProvider returns a metrics provider that logs to the current logger.
// Useful for development and local verification.
// Alias for pkg/metrics.LogProvider.
func NewLogMetricsProvider() metrics.Provider {
	return &metrics.LogProvider{}
}
