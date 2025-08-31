package signals

// Scope represents the lifetime of a reactive computation.
type Scope interface{}

// Root creates a new reactive scope and runs the given function within it.
// It returns a dispose function to tear down the scope.
func Root(fn func(s Scope)) (dispose func()) {
	// For M0, we just call the function and return a no-op disposer.
	fn(nil)
	return func() {}
}
