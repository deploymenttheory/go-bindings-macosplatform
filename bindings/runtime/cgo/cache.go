//go:build darwin

package cgo

import "sync"

// SyncCache is a thread-safe compute-once cache. The first call to Load for a
// given key invokes compute; subsequent calls return the cached value.
type SyncCache[K comparable, V any] struct {
	m  sync.Map
	mu sync.Mutex
}

// Load returns the cached value for key, computing it via compute on first access.
func (c *SyncCache[K, V]) Load(key K, compute func(K) V) V {
	if v, ok := c.m.Load(key); ok {
		return v.(V)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// Double-check after acquiring the lock.
	if v, ok := c.m.Load(key); ok {
		return v.(V)
	}
	v := compute(key)
	c.m.Store(key, v)
	return v
}
