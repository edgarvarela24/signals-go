package signals

// Memo creates a new computed signal.
// It's lazy, only re-computing its value when read and a dependency has changed.
func Memo[T any](s *Scope, fn func() T) Readonly[T] {
	// TODO: M4
	return nil
}
