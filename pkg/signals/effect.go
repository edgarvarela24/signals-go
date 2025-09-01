package signals

import "sync"

// A computation is anything that can be subscribed to a signal.
type computation interface {
	// notify is called by a signal this computation is subscribed to.
	notify()
	addSource(s subscribable)
}

type effect struct {
	fn      func()
	scope   *Scope
	sources map[subscribable]struct{}
	mu      sync.Mutex
}

func (e *effect) addSource(s subscribable) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.sources == nil {
		e.sources = make(map[subscribable]struct{})
	}
	e.sources[s] = struct{}{}
}

func (e *effect) cleanup() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for s := range e.sources {
		s.unsubscribe(e)
	}
	e.sources = nil // Allow GC
}

func (e *effect) notify() {
	e.cleanup() // Clean up old dependencies before re-running
	e.scope.engine.pushListener(e)
	e.fn()
	e.scope.engine.popListener()
}

// Effect registers a function to be run when its dependencies change.
func Effect(s *Scope, fn func()) (stop func()) {
	e := &effect{fn: fn, scope: s}
	e.notify()
	return e.cleanup
}

// Untrack prevents a signal read from creating a dependency.
func Untrack(s *Scope, fn func()) {
	s.engine.pushListener(nil)
	defer s.engine.popListener()
	fn()
}

// OnCleanup registers a function to be run when the current scope is disposed.
func OnCleanup(s *Scope, fn func()) {
	s.cleanup = append(s.cleanup, fn)
}
