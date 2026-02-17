package runtime

// Task represents a handle to a background goroutine managed by the lifecycle runtime.
// It allows for synchronization (Wait) and retrieval of results or synchronization errors.
//
// A Task is created when using lifecycle.Go(). It provides a way to wait for a
// specific goroutine to finish even if the main application lifecycle is still running.
type Task interface {
	// Wait blocks until the task completes.
	// Returns nil if the task finished successfully, or an error if the task
	// returned an error or panicked.
	Wait() error
}

// taskHandle implements the Task interface.
type taskHandle struct {
	done chan struct{}
	err  error
}

// Wait implements Task.
func (t *taskHandle) Wait() error {
	<-t.done
	return t.err
}

// ErrorHandler is a function type used to capture and process errors from
// background tasks asynchronously. This is useful for logging, reporting,
// or triggering recovery logic without blocking the main flow.
type ErrorHandler func(error)

// GoOption provides functional configuration for background tasks.
type GoOption func(*goConfig)

type goConfig struct {
	errorHandler ErrorHandler
	stackCapture *bool
}

// WithErrorHandler registers a handler for task errors.
// This is useful for logging or metrics when you don't want to Wait() on the task.
func WithErrorHandler(h ErrorHandler) GoOption {
	return func(c *goConfig) {
		c.errorHandler = h
	}
}

// WithStackCapture forces stack capture behavior for background task panics.
// When enabled, stack traces are captured for observer hooks even if debug logging is off.
// When disabled, stack traces are skipped even if debug logging is on.
// If not set, behavior auto-detects based on debug logging level.
func WithStackCapture(enabled bool) GoOption {
	return func(c *goConfig) {
		c.stackCapture = &enabled
	}
}
