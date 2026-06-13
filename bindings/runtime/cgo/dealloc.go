//go:build darwin

package cgo

// #cgo CFLAGS: -fno-objc-arc -x objective-c
// #cgo LDFLAGS: -framework Foundation
// #include "bridge.h"
import "C"

import (
	"runtime/cgo"
	"unsafe"
)

//export goObjcDeallocFired
func goObjcDeallocFired(handle C.uintptr_t) {
	h := cgo.Handle(handle)
	if fn, ok := h.Value().(func()); ok {
		fn()
	}
	h.Delete()
}

// SetDeallocListener fires listener exactly once when obj is deallocated by the
// ObjC runtime. This is the correct mechanism to notify Go when an ObjC object
// it holds a reference to is being torn down — for example, to clean up Go
// closures stored via associated objects that would otherwise be orphaned.
//
// Internally this attaches a GoObjcDeallocListener as a retained associated
// object on obj. When obj is released, ObjC clears its associated objects,
// releasing the listener, whose -dealloc fires the callback.
func SetDeallocListener(obj unsafe.Pointer, listener func()) {
	h := cgo.NewHandle(listener)
	listenerPtr := C.runtime_new_dealloc_listener(C.uintptr_t(h))
	// Retain policy: ObjC retains the listener; when obj is freed its
	// associated objects are released, triggering the listener's -dealloc.
	SetAssociatedObject(obj, "_go_dealloc_listener", listenerPtr, AssocRetain)
	// Release our +1 from runtime_new_dealloc_listener; ObjC holds the only ref.
	Release(listenerPtr)
}
