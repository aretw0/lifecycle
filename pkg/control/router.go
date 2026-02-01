package control

import (
	"context"
	"fmt"
	"sync"
)

// Router routes events from sources to reactions.
type Router struct {
	mu        sync.RWMutex
	routes    map[string][]Reaction
	sources   []Source
	events    chan Event
	isRunning bool
}

// NewRouter creates a new control router.
func NewRouter() *Router {
	return &Router{
		routes: make(map[string][]Reaction),
		events: make(chan Event, 100), // Buffer events to prevent blocking sources
	}
}

// AddSource registers an event source.
func (r *Router) AddSource(s Source) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources = append(r.sources, s)
}

// On binds an event pattern (via string matching) to a reaction.
// For now, it uses exact string matching of Event.String().
// TODO: Implement regex or type-based matching.
func (r *Router) On(pattern string, reaction Reaction) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[pattern] = append(r.routes[pattern], reaction)
}

// Start runs the router loop. It starts all sources and listens for events.
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
			for e := range src.Events() {
				select {
				case r.events <- e:
				case <-ctx.Done():
					return
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
	go func() {
		defer wg.Done()
		wg.Add(1)
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-r.events:
				r.dispatch(ctx, e)
			}
		}
	}()

	wg.Wait()
	return nil
}

func (r *Router) dispatch(ctx context.Context, e Event) {
	r.mu.RLock()
	reactions := r.routes[e.String()]
	// Also check for wildcard or partial matches if we implement them later
	r.mu.RUnlock()

	if len(reactions) == 0 {
		return
	}

	// Execute reactions
	// TODO: Managed Concurrency for reactions? Or Router blocks?
	// For now, execute sequentially to ensure order.
	for _, rn := range reactions {
		if err := rn(ctx); err != nil {
			// Log error but don't stop router
			// TODO: Use pkg/log
			fmt.Printf("Error executing reaction for %s: %v\n", e, err)
		}
	}
}
