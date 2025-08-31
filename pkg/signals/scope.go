package signals

import "sync/atomic"

// Scope represents the lifetime of a reactive computation.
type Scope struct {
	isLive atomic.Bool
}

// Batch method to resolve the compilation error in the test file.
func (s *Scope) Batch(fn func()) {
	// For now, just handles disposed state
	if !s.isLive.Load() {
		return
	}
	fn()
}
