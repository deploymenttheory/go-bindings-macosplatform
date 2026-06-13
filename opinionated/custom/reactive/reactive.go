// Package reactive provides a generic observable value type used to decouple
// state producers (VM lifecycle) from consumers (UI layer).
package reactive

import "sync"

// Observable[T] holds a value of type T and notifies subscribers whenever it changes.
type Observable[T any] struct {
	mu          sync.RWMutex
	value       T
	subscribers map[uint64]func(T)
	nextID      uint64
}

// New creates an Observable initialised with the given value.
func New[T any](initial T) *Observable[T] {
	return &Observable[T]{
		value:       initial,
		subscribers: make(map[uint64]func(T)),
	}
}

// Get returns the current value.
func (o *Observable[T]) Get() T {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.value
}

// Set updates the value and synchronously notifies all subscribers.
func (o *Observable[T]) Set(v T) {
	o.mu.Lock()
	o.value = v
	subs := make([]func(T), 0, len(o.subscribers))
	for _, fn := range o.subscribers {
		subs = append(subs, fn)
	}
	o.mu.Unlock()
	for _, fn := range subs {
		fn(v)
	}
}

// Subscribe registers fn to be called whenever the value changes.
// Returns an unsubscribe function; call it to stop receiving notifications.
func (o *Observable[T]) Subscribe(fn func(T)) func() {
	o.mu.Lock()
	id := o.nextID
	o.nextID++
	o.subscribers[id] = fn
	o.mu.Unlock()
	return func() {
		o.mu.Lock()
		delete(o.subscribers, id)
		o.mu.Unlock()
	}
}
