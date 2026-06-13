//go:build darwin

// Package delegates provides a handle-based registry that maps uint64 tokens to
// Go delegate implementations. The ObjC shim classes store these handles as ivar
// uint64_t values; on each protocol method callback they pass the handle back
// through CGo so the generated export functions can look up the Go impl.
//
// Lifetime: Register when the delegate is installed; the ObjC shim calls
// goDelegate_unregister from its -dealloc, which removes the entry.
package delegates

// #include <stdint.h>
import "C"

import (
	"sync"
	"sync/atomic"
)

var (
	registry sync.Map // uint64 → any
	seq      uint64
)

// Register stores impl in the registry and returns a unique handle.
func Register(impl any) uint64 {
	handle := atomic.AddUint64(&seq, 1)
	registry.Store(handle, impl)
	return handle
}

// Lookup returns the implementation stored under handle, or nil.
func Lookup(handle uint64) any {
	v, _ := registry.Load(handle)
	return v
}

// Unregister removes the entry for handle.
func Unregister(handle uint64) {
	registry.Delete(handle)
}

//export goDelegate_unregister
func goDelegate_unregister(handle C.uint64_t) {
	Unregister(uint64(handle))
}
