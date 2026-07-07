package mempool

import (
	"sync"
)

// Erasable is an object whose data can be completely nullified in order to reuse it.
type Erasable interface {
	// Erase resets all the fields of the object to defaults (not deeply).
	Erase()
}

type PoolOption func(*poolConfig)

type poolConfig struct {
	capacity   int
	threadSafe bool
}

func WithCapacity(capacity int) PoolOption {
	return func(cfg *poolConfig) {
		cfg.capacity = capacity
	}
}

func WithThreadSafety(threadSafe bool) PoolOption {
	return func(cfg *poolConfig) {
		cfg.threadSafe = threadSafe
	}
}

type Pool[T Erasable] struct {
	stack       []T
	top         int32
	constructor func() T
	mutex       *sync.Mutex
	threadSafe  bool
}

func NewPool[T Erasable](constructor func() T, options ...PoolOption) *Pool[T] {
	cfg := &poolConfig{
		capacity:   1000, // Default capacity
		threadSafe: false,
	}
	for _, option := range options {
		option(cfg)
	}
	p := &Pool[T]{
		stack:       make([]T, cfg.capacity),
		top:         -1,
		constructor: constructor,
		threadSafe:  cfg.threadSafe,
	}
	if cfg.threadSafe {
		p.mutex = &sync.Mutex{}
	}
	return p
}

func (p *Pool[T]) Get() T {
	if p.threadSafe {
		return p.getThreadSafe()
	}
	return p.getUnsafe()
}

func (p *Pool[T]) getThreadSafe() T {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.top >= 0 {
		obj := p.stack[p.top]
		p.stack[p.top] = *new(T) // Clear the reference
		p.top--
		return obj
	}
	return p.constructor()
}

func (p *Pool[T]) getUnsafe() T {
	if p.top >= 0 {
		obj := p.stack[p.top]
		p.stack[p.top] = *new(T) // Clear the reference
		p.top--
		return obj
	}
	return p.constructor()
}

func (p *Pool[T]) Put(obj T) {
	if p.threadSafe {
		p.putThreadSafe(obj)
	} else {
		p.putUnsafe(obj)
	}
}

func (p *Pool[T]) putThreadSafe(obj T) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.top < int32(len(p.stack))-1 {
		obj.Erase()
		p.top++
		p.stack[p.top] = obj
	}
	// If stack is full, the object is discarded
}

func (p *Pool[T]) putUnsafe(obj T) {
	if p.top < int32(len(p.stack))-1 {
		obj.Erase()
		p.top++
		p.stack[p.top] = obj
	}
	// If stack is full, the object is discarded
}

func (p *Pool[T]) Size() int {
	if p.threadSafe {
		p.mutex.Lock()
		defer p.mutex.Unlock()
	}
	return int(p.top + 1)
}

func (p *Pool[T]) Capacity() int {
	return len(p.stack)
}
