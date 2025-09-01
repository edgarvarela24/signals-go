package signals

import "sync/atomic"

// Scope represents the lifetime of a reactive computation.
type Scope struct {
	isLive  atomic.Bool
	engine  *Engine
	cleanup []func()
}

func (s *Scope) Batch(fn func()) {
	// For now, just handles disposed state
	if !s.isLive.Load() {
		return
	}

	// Register batch with engine
	s.engine.isBatching.Store(true)

	// Ensure we always end the batch and flush the queue
	defer func() {
		s.engine.batchQueueMu.Lock()
		// Copy the queue to avoid holding the lock while notifying
		queue := make([]computation, 0, len(s.engine.batchQueue))
		for sub := range s.engine.batchQueue {
			queue = append(queue, sub)
		}
		s.engine.batchQueueMu.Unlock()

		// Notify subscribers
		for _, sub := range queue {
			sub.notify()
		}
	}()

	fn()
}

func (s *Scope) Dispose() {
	if !s.isLive.Swap(false) {
		return
	}

	// Run cleanup functions in reverse order
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		s.cleanup[i]()
	}
	s.cleanup = nil // Allow GC
}

func New[T any](s *Scope, initial T) Signal[T] {
	return &signal[T]{
		scope:       s,
		value:       initial,
		subscribers: make(map[computation]struct{}),
	}
}
