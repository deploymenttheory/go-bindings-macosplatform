//go:build darwin

package objc

// #cgo CFLAGS: -fno-objc-arc -x objective-c
// #cgo LDFLAGS: -framework Foundation
// #include "bridge.h"
import "C"

import (
	"runtime"
	"unsafe"
)

// Retain calls ObjC retain on ptr and returns ptr.
func Retain(ptr unsafe.Pointer) unsafe.Pointer {
	if ptr != nil {
		C.runtime_objc_retain(ptr)
	}
	return ptr
}

// Release calls ObjC release on ptr immediately.
func Release(ptr unsafe.Pointer) {
	if ptr != nil {
		C.runtime_objc_release(ptr)
	}
}

// Track registers a finalizer on wrapper that calls Release on its ObjC pointer
// when the GC collects the wrapper. ptrFn must return the wrapper's ptr field.
//
// AppKit objects (NSView and its subclasses) perform CoreAnimation layer cleanup
// in -dealloc that must run on the main thread. Go's finalizer goroutine is never
// the main thread, so we always dispatch the release to the main queue rather than
// calling CFRelease inline. This prevents kCAValueWeakPointer / NSViewBackingLayer
// corruption that causes CA::AttrList::get crashes on the next AppKit call.
func Track(wrapper any, ptrFn func() unsafe.Pointer) {
	runtime.SetFinalizer(wrapper, func(_ any) {
		ptr := ptrFn()
		if ptr == nil {
			return
		}
		if C.runtime_is_main_thread() != 0 {
			C.runtime_objc_release(ptr)
		} else {
			DispatchAsyncMain(func() {
				C.runtime_objc_release(ptr)
			})
		}
	})
}

// KeepAlive marks v as live until the call to KeepAlive, preventing the GC
// from finalizing ObjC wrapper objects before a CGo call using the raw pointer
// has completed.
func KeepAlive(v any) { runtime.KeepAlive(v) }

// AllocObject calls +alloc on the named ObjC class and returns the uninitialized
// +1-retained pointer. The caller must immediately pass it to an init method.
// Returns nil if the class is not registered at runtime.
func AllocObject(className string) unsafe.Pointer {
	cs := C.CString(className)
	defer C.free(unsafe.Pointer(cs))
	return C.runtime_objc_alloc_class(cs)
}

// FreePtr releases a heap-allocated C pointer returned by the bridge layer.
// Used by generated code to free the malloc'd struct buffer after copying it
// into a Go value (value-type struct returns).
func FreePtr(ptr unsafe.Pointer) {
	if ptr != nil {
		C.free(ptr)
	}
}
