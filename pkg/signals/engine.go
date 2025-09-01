package signals

import (
	"errors"
	"sync"
	"sync/atomic"
)

var ErrEngineClosed = errors.New("signals: engine is closed")

type Engine struct {
	root          *Scope
	isClosed      atomic.Bool
	listener      computation
	listenerStack []computation
	listenerMu    sync.Mutex
	isBatching    atomic.Bool
	batchQueue    map[computation]struct{}
	batchQueueMu  sync.Mutex
}
type Option func(*Engine)

func Start(opts ...Option) *Engine {
	e := &Engine{
		batchQueue: make(map[computation]struct{}),
	}
	e.root = &Scope{
		isLive: atomic.Bool{},
		engine: e,
	}
	e.root.isLive.Store(true)
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func (e *Engine) Close() error {
	if e.isClosed.Swap(true) {
		return ErrEngineClosed
	}
	e.root.Dispose()
	return nil
}

func (e *Engine) Scope() *Scope {
	return e.root
}

func (e *Engine) pushListener(c computation) {
	e.listenerMu.Lock()
	defer e.listenerMu.Unlock()
	e.listenerStack = append(e.listenerStack, e.listener)
	e.listener = c
}

func (e *Engine) popListener() {
	e.listenerMu.Lock()
	defer e.listenerMu.Unlock()
	if len(e.listenerStack) > 0 {
		e.listenerStack = e.listenerStack[:len(e.listenerStack)-1]
	}
	if len(e.listenerStack) > 0 {
		e.listener = e.listenerStack[len(e.listenerStack)-1]
	} else {
		e.listener = nil
	}
}
