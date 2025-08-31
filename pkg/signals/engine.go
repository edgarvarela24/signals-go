package signals

import (
	"errors"
	"sync/atomic"
)

var ErrEngineClosed = errors.New("signals: engine is closed")

type Engine struct {
	root     *Scope
	isClosed atomic.Bool
}
type Option func(*Engine)

func Start(opts ...Option) *Engine {
	s := &Scope{}
	s.isLive.Store(true)
	eng := &Engine{
		root: s,
	}
	for _, opt := range opts {
		opt(eng)
	}
	return eng
}

func (e *Engine) Close() error {
	if !e.isClosed.CompareAndSwap(false, true) {
		return ErrEngineClosed
	}
	e.root.isLive.Store(false)
	return nil
}

func (e *Engine) Scope() *Scope {
	return e.root
}
