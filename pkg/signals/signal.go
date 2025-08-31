package signals

// Interfaces
type Readonly[T any] interface {
	Get() T
}

type Signal[T any] interface {
	Readonly[T] // Embeds Get()
	Set(T)
	Update(func(*T))
}

type signal[T any] struct {
	scope *Scope
	value T
}

func (s *signal[T]) Get() T {
	return s.value
}

func (s *signal[T]) Set(value T) {
	s.value = value
}

func (s *signal[T]) Update(fn func(*T)) {
	fn(&s.value)
}
