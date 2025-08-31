package signals

import "sync"

// Scope represents the lifetime of a reactive computation.
type Scope interface {
	Batch(fn func())
}

type scope struct {
	disposed bool
	mu       sync.Mutex
}

func (s *scope) Batch(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.disposed {
		return
	}

	fn()
}

// Root creates a new reactive scope and runs the given function within it.
// It returns a dispose function to tear down the scope.
func Root(fn func(s Scope)) (dispose func()) {
	// Create concrete scope instance.
	s := &scope{}

	// Run user's function with scope.
	fn(s)

	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.disposed = true
	}
}
