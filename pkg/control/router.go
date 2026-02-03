package control

import (
	"context"
	"fmt"
	"path"
	"sync"

	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
	"github.com/aretw0/lifecycle/pkg/runtime"
)

// Middleware wraps a Handler.
type Middleware func(next Handler) Handler

// Router routes events from sources to reactions using a ServeMux-style pattern.
type Router struct {
	mu          sync.RWMutex
	routes      map[string]Handler // Pattern -> Handler
	middlewares []Middleware
	sources     []Source
	events      chan Event
	isRunning   bool
}

// RouterOption configures a Router.
type RouterOption func(*Router)

// WithEventBuffer sets the size of the event channel buffer.
// Default is 100.
func WithEventBuffer(size int) RouterOption {
	return func(r *Router) {
		if size < 0 {
			size = 0
		}
		r.events = make(chan Event, size)
	}
}

// NewRouter creates a new control router with optional configuration.
func NewRouter(opts ...RouterOption) *Router {
	r := &Router{
		routes: make(map[string]Handler),
		// Default buffer, can be overwritten by opts
		events: make(chan Event, 100),
	}

	for _, opt := range opts {
		opt(r)
	}
	return r
}

// DefaultRouter is the default instance for package-level helpers.
var DefaultRouter = NewRouter()

// Handle registers a handler on the DefaultRouter.
func Handle(pattern string, handler Handler) {
	DefaultRouter.Handle(pattern, handler)
}

// HandleFunc registers a handler function on the DefaultRouter.
func HandleFunc(pattern string, handler func(context.Context, Event) error) {
	DefaultRouter.HandleFunc(pattern, handler)
}

// Handle registers the handler for the given pattern.
// Patterns supports glob matching via path.Match.
func (r *Router) Handle(pattern string, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if pattern == "" {
		panic("control: invalid pattern")
	}
	if handler == nil {
		panic("control: nil handler")
	}

	// Check for collision? Standard ServeMux overwrites or panics.
	// We will overwrite for now to allow dynamic re-routing.
	r.routes[pattern] = handler
}

// HandleFunc registers the handler function for the given pattern.
func (r *Router) HandleFunc(pattern string, handler func(context.Context, Event) error) {
	r.Handle(pattern, HandlerFunc(handler))
}

// Use appends a middleware to the router stack.
func (r *Router) Use(mw Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middlewares = append(r.middlewares, mw)
}

// AddSource registers an event source.
func (r *Router) AddSource(s Source) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources = append(r.sources, s)
}

// Start runs the router loop.
func (r *Router) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.isRunning {
		r.mu.Unlock()
		return fmt.Errorf("router already running")
	}
	r.isRunning = true
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.isRunning = false
		r.mu.Unlock()
	}()

	var wg sync.WaitGroup

	// Start all sources
	for _, s := range r.sources {
		wg.Add(1)
		go func(src Source) {
			defer wg.Done()
			// Forward events from source to router
			// We iterate until src.Events() is closed or ctx is done
			for e := range runtime.Receive(ctx, src.Events()) {
				select {
				case r.events <- e:
				case <-ctx.Done():
					return // Context cancelled while sending
				}
			}
		}(s)

		// Start source lifecycle
		wg.Add(1)
		go func(src Source) {
			defer wg.Done()
			src.Start(ctx)
		}(s)
	}

	// Process events
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-r.events:
				r.Dispatch(ctx, e)
			}
		}
	}()

	wg.Wait()
	return nil
}

// Dispatch finds the handler for an event and executes it.
func (r *Router) Dispatch(ctx context.Context, e Event) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	topic := e.String()
	var matchedHandler Handler

	// Metrics
	metrics.GetProvider().IncEventRouted(topic)

	// 1. Exact match
	if h, ok := r.routes[topic]; ok {
		matchedHandler = h
	} else {
		// 2. Glob match
		// TODO: Optimize if many routes
		for pattern, h := range r.routes {
			if matched, _ := path.Match(pattern, topic); matched {
				matchedHandler = h
				break // First match wins? Or aggregation? Sticking to first match for Mux style.
			}
		}
	}

	if matchedHandler == nil {
		return
	}

	// Apply middleware
	finalHandler := matchedHandler
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		finalHandler = r.middlewares[i](finalHandler)
	}

	// Execute
	metrics.GetProvider().IncHandlerExecuted(topic)
	if err := finalHandler.HandleEvent(ctx, e); err != nil {
		metrics.GetProvider().IncHandlerError(topic, err)
		log.Error("handler error", "topic", topic, "error", err)
	}
}

// Routes returns a snapshot of the currently registered routes.
func (r *Router) Routes() []RouteInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes := make([]RouteInfo, 0, len(r.routes))
	for pattern, h := range r.routes {
		handlerName := "Handler"

		// Attempt to get function name via reflection
		// We first check if it's a HandlerFunc to get the underlying function
		if hf, ok := h.(HandlerFunc); ok {
			handlerName = runtime.GetFunctionName(hf)
		} else {
			// For generic interfaces, we just type name it
			handlerName = fmt.Sprintf("%T", h)
		}

		routes = append(routes, RouteInfo{
			Pattern: pattern,
			Handler: handlerName,
		})
	}
	return routes
}

// RouterState represents the serializable state of the router.
type RouterState struct {
	Routes      []RouteInfo `json:"routes"`
	Middlewares int         `json:"middlewares"`
	Sources     int         `json:"sources"`
	Running     bool        `json:"running"`
}

// State returns the current state of the router.
func (r *Router) State() any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return RouterState{
		Routes:      r.Routes(),
		Middlewares: len(r.middlewares),
		Sources:     len(r.sources),
		Running:     r.isRunning,
	}
}
