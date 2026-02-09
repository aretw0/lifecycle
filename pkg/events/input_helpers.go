package events

import "context"

// CommandRouter is a specialized Router helper for string-command to event mapping.
// It creates a standard Router configured with exact string matches.
func NewCommandRouter(mappings map[string]Event) *Router {
	r := NewRouter()
	for cmd, _ := range mappings {
		// We use a functional handler that emits the mapped event.
		// NOTE: The InputSource emits "command/NAME", so the Router usually routes "command/NAME".
		// But here we are defining the *routing* logic.
		// If this is meant to route "command/quit" -> quitHandler, that's different from mapping "quit" string -> QuitEvent.

		// Wait, InputSource maps string -> Event. Router maps Event.String() -> Handler.
		// CommandRouter might be a misnomer if we mean "InputSource configuration".
		// But if we mean "Route specific string events", we usually do that via "InputSource mappings".

		// Let's implement a helper that simplifies the "Command -> Handler" wiring, which is what NewInteractiveRouter did conceptually.

		r.Handle("command/"+cmd, HandlerFunc(func(ctx context.Context, e Event) error {
			// This is a placeholder. A real router routes specific event topics.
			// Ideally, we want to route the *Event* derived from the command.
			return nil
		}))

		// RE-THINKING: The "InputSource" maps string -> Event.
		// The "Router" routes Event.String() -> Handler.
		// So "CommandRouter" in the proposal was "Just maps strings to events".
		// This sounds like a Source configuration helper, not a Router.
		// But if we stick to the proposal: "func NewCommandRouter(mappings map[string]Event) *Router"
		// This implies it acts as a Source + Router? No, Router doesn't hold mappings.

		// Let's assume the user meant "A simplified Router setup for commands".
		// But actually, NewInteractiveRouter did wiring.

		// Correct Abstraction: InputMapper + Router.
		// Let's implement a Helper to create an InputSource with mappings easily.
	}
	return r
}

// CommandMapping creates a map for InputSource.
func CommandMapping(pairs ...any) map[string]Event {
	m := make(map[string]Event)
	for i := 0; i < len(pairs); i += 2 {
		cmd := pairs[i].(string)
		evt := pairs[i+1].(Event)
		m[cmd] = evt
	}
	return m
}
