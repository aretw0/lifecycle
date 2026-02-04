package control

// BaseSource provides default implementation for Source.Events() method.
// Embed this in your source types to avoid repeating the events channel boilerplate.
//
// Example:
//
//	type MySource struct {
//	    control.BaseSource
//	    // custom fields...
//	}
//
//	func NewMySource() *MySource {
//	    return &MySource{
//	        BaseSource: control.NewBaseSource(10), // buffer size
//	    }
//	}
//
//	func (s *MySource) Start(ctx context.Context) error {
//	    go func() {
//	        for {
//	            event := // ... create event
//	            s.Emit(event) // Helper method
//	        }
//	    }()
//	    return nil
//	}
//
// The embedding provides Events() implementation automatically.
type BaseSource struct {
	events chan Event
}

// NewBaseSource creates a BaseSource with the specified buffer size.
// A buffer of 10-100 is recommended for most sources to prevent blocking.
func NewBaseSource(bufferSize int) BaseSource {
	return BaseSource{
		events: make(chan Event, bufferSize),
	}
}

// Events returns the read-only events channel.
// This method is automatically available via embedding.
func (b *BaseSource) Events() <-chan Event {
	return b.events
}

// Emit sends an event to the events channel.
// This is a helper method for source implementations.
// It blocks if the channel buffer is full.
func (b *BaseSource) Emit(e Event) {
	b.events <- e
}

// Close closes the events channel.
// Call this when the source is done emitting events.
func (b *BaseSource) Close() {
	close(b.events)
}
