package signals

type memo[T any] struct {
	signal[T]
	fn      func() T
	isDirty bool
	sources map[subscribable]struct{}
}

// Memo creates a new computed signal.
// It's lazy, only re-computing its value when read and a dependency has changed.
func Memo[T any](s *Scope, fn func() T) Readonly[T] {
	m := &memo[T]{
		signal: signal[T]{
			scope:       s,
			subscribers: make(map[computation]struct{}),
		},
		fn:      fn,
		isDirty: true, // Start dirty to compute on first Get()
	}
	OnCleanup(s, m.cleanup)
	return m
}

func (m *memo[T]) Get() T {
	if listener := m.scope.engine.listener; listener != nil {
		m.mu.Lock()
		if m.subscribers == nil {
			m.subscribers = make(map[computation]struct{})
		}
		m.subscribers[listener] = struct{}{}
		m.mu.Unlock()
		listener.addSource(m)
	}

	if m.isDirty {
		m.runComputation()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.value
}

func (m *memo[T]) runComputation() {
	m.cleanup() // Clean up old dependencies before re-running
	m.scope.engine.pushListener(m)
	newValue := m.fn()
	m.scope.engine.popListener()

	m.mu.Lock()
	m.value = newValue
	m.isDirty = false
	m.mu.Unlock()
}

func (m *memo[T]) notify() {
	m.mu.Lock()
	if m.isDirty {
		m.mu.Unlock()
		return
	}
	m.isDirty = true
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()
	for sub := range m.subscribers {
		sub.notify()
	}
}

func (m *memo[T]) addSource(s subscribable) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sources == nil {
		m.sources = make(map[subscribable]struct{})
	}
	m.sources[s] = struct{}{}
}

func (m *memo[T]) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for s := range m.sources {
		s.unsubscribe(m)
	}
	m.sources = nil
}
