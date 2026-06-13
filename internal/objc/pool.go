//go:build darwin

package objc

// #cgo CFLAGS: -fno-objc-arc -x objective-c
// #cgo LDFLAGS: -framework Foundation
// #include "bridge.h"
import "C"

import "unsafe"

// WithAutoreleasePool runs task inside an NSAutoreleasePool drain cycle.
//
// Any ObjC work performed outside the main application run loop — including
// goroutines, test helpers, and background tasks — must be wrapped in this
// function to prevent autorelease objects from leaking.
func WithAutoreleasePool(task func()) {
	pool := C.runtime_autorelease_pool_push()
	defer C.runtime_autorelease_pool_pop(pool)
	task()
}

// PoolScope represents an explicit autorelease pool boundary.
// Use NewPoolScope for code paths that create many short-lived ObjC objects
// outside a normal run-loop cycle (e.g., tight loops over NSString conversions).
// Call Drain when the scope exits.
type PoolScope struct{ pool unsafe.Pointer }

// NewPoolScope pushes a new autorelease pool and returns the scope handle.
// The caller must call Drain() to pop the pool and release its contents.
// Typically used with defer:
//
//	scope := objc.NewPoolScope()
//	defer scope.Drain()
func NewPoolScope() PoolScope {
	return PoolScope{pool: C.runtime_autorelease_pool_push()}
}

// Drain pops the autorelease pool, releasing all autoreleased objects in the
// scope. Safe to call multiple times (subsequent calls are no-ops).
func (s *PoolScope) Drain() {
	if s.pool == nil {
		return
	}
	C.runtime_autorelease_pool_pop(s.pool)
	s.pool = nil
}
