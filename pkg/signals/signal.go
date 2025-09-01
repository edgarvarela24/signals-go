package signals

import "sync"

// Interfaces
type Readonly[T any] interface {
	Get() T
}

type Signal[T any] interface {
	Readonly[T] // Embeds Get()
	Set(T)
	Update(func(*T))
}

// A subscribable is a source that a computable can subscribe to
type subscribable interface {
	unsubscribe(c computation)
}

type signal[T any] struct {
	scope       *Scope
	value       T
	subscribers map[computation]struct{}
	mu          sync.RWMutex
}

func (s *signal[T]) unsubscribe(c computation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subscribers, c)
}

func (s *signal[T]) Get() T {
	// If listener, add to our subscribers
	if listener := s.scope.engine.listener; listener != nil {
		s.mu.Lock()
		if s.subscribers == nil {
			s.subscribers = make(map[computation]struct{})
		}
		s.subscribers[listener] = struct{}{}
		s.mu.Unlock()

		// And tell the listener that it is now subscribed to us.
		listener.addSource(s)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

func (s *signal[T]) Set(value T) {
	s.mu.Lock()
	s.value = value
	s.mu.Unlock()

	s.scope.engine.batchQueueMu.Lock()
	defer s.scope.engine.batchQueueMu.Unlock()
	if s.scope.engine.isBatching.Load() {
		for sub := range s.subscribers {
			s.scope.engine.batchQueue[sub] = struct{}{}
		}
	} else {
		// Notify subscribers
		for sub := range s.subscribers {
			sub.notify()
		}
	}
}

func (s *signal[T]) Update(fn func(*T)) {
	s.mu.Lock()
	fn(&s.value)
	s.mu.Unlock()
}
