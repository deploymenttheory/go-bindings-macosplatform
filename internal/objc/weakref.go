//go:build darwin

package objc

// #cgo CFLAGS: -fno-objc-arc -x objective-c
// #cgo LDFLAGS: -framework Foundation
// #include "bridge.h"
import "C"

import "unsafe"

// WeakRef holds a type-safe weak reference to an ObjC object.
// Unlike a strong reference, WeakRef does not increment the retain count.
// When the referenced object is deallocated the ObjC runtime automatically
// zeroes the stored pointer, causing GetPtr to return nil.
//
// WeakRef is the correct pattern for delegate and observer scenarios where
// the delegate holds a back-reference to its owner, avoiding retain cycles.
//
// Usage:
//
//	weak := objc.NewWeakRef[*appkit.NSView](myView)
//
//	// Later — retrieve and use (non-nil means the object is still alive):
//	if ptr := weak.GetPtr(); ptr != nil {
//	    view := appkit.NewNSView(ptr) // constructor takes ownership of the +1 retain
//	    view.SetNeedsDisplay(ctx, true)
//	}
type WeakRef[T Object] struct {
	// slot is a heap-allocated void* registered with the ObjC runtime via
	// runtime_weak_store (wraps objc_storeWeak). The runtime zeroes *slot when
	// the referenced object is deallocated.
	slot *unsafe.Pointer
}

// NewWeakRef creates a weak reference to obj.
// The obj argument must be non-nil and must carry a non-nil ObjC pointer;
// a zero WeakRef is returned otherwise.
func NewWeakRef[T Object](obj T) WeakRef[T] {
	ptr := obj.Ptr()
	if ptr == nil {
		return WeakRef[T]{}
	}
	slot := new(unsafe.Pointer)
	C.runtime_weak_store((*unsafe.Pointer)(slot), ptr)
	return WeakRef[T]{slot: slot}
}

// GetPtr returns the raw ObjC pointer of the referenced object with a +1
// retain (via objc_loadWeakRetained), or nil if the object was deallocated.
//
// The caller must transfer ownership of the +1 retain to an object wrapper
// that registers a Go finalizer. The idiomatic pattern is to pass the pointer
// directly to the framework constructor (which calls objc.Track):
//
//	if ptr := weak.GetPtr(); ptr != nil {
//	    view := appkit.NewNSView(ptr) // NewNSView calls objc.Track, taking the +1
//	    view.SetNeedsDisplay(ctx, true)
//	}
//
// If the pointer is not passed to a constructor, call objc.Release(ptr) to
// balance the retain.
func (w WeakRef[T]) GetPtr() unsafe.Pointer {
	if w.slot == nil {
		return nil
	}
	return C.runtime_weak_load((*unsafe.Pointer)(w.slot))
}

// IsAlive reports whether the referenced object is still alive.
// This is an advisory check — the object may be deallocated between an
// IsAlive call and the subsequent GetPtr call. Prefer a nil-check on
// GetPtr for correctness in concurrent code.
func (w WeakRef[T]) IsAlive() bool {
	if w.slot == nil {
		return false
	}
	ptr := C.runtime_weak_load((*unsafe.Pointer)(w.slot))
	if ptr == nil {
		return false
	}
	C.runtime_objc_release(ptr)
	return true
}

// Clear removes the weak reference registration, freeing the slot.
// After Clear, GetPtr always returns nil. Safe to call multiple times.
func (w *WeakRef[T]) Clear() {
	if w.slot == nil {
		return
	}
	C.runtime_weak_store((*unsafe.Pointer)(w.slot), nil)
	w.slot = nil
}
