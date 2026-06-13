//go:build darwin

package callbacks

// #cgo CFLAGS: -fno-objc-arc -x objective-c
// #cgo LDFLAGS: -framework Foundation
// #include "callbacks_abi.h"
import "C"

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type callbackEntry struct {
	fn       any
	refCount int64
}

var (
	callbackRegistry sync.Map
	callbackKeySeq   uint64
)

// Register stores fn in the callback registry and returns a unique handle.
// The caller must call Free(handle) when the callback is no longer needed
// (e.g. when the ObjC object that holds it is deallocated).
func Register(fn any) uint64 {
	key := atomic.AddUint64(&callbackKeySeq, 1)
	callbackRegistry.Store(key, &callbackEntry{fn: fn, refCount: 1})
	return key
}

// callbackLookup returns the Go func stored under key, or nil if not found.
// Used by generated goCallIMP_* functions (unexported by convention, same as blocks).
func callbackLookup(key uint64) any {
	if v, ok := callbackRegistry.Load(key); ok {
		return v.(*callbackEntry).fn
	}
	return nil
}

// Free decrements the reference count for key and removes the entry when it
// reaches zero. Callers should invoke Free after the ObjC object that holds
// the binding is released.
func Free(key uint64) {
	if v, ok := callbackRegistry.Load(key); ok {
		e := v.(*callbackEntry)
		if atomic.AddInt64(&e.refCount, -1) <= 0 {
			callbackRegistry.Delete(key)
		}
	}
}

// ClassRegistrationMu serializes ObjC dynamic class registration across all
// generated subclass and protocol-impl factory functions.
// Apple's objc_allocateClassPair and objc_registerClassPair are not safe to
// call concurrently; Go's scheduler can context-switch between goroutines at
// any CGo call boundary, so the full alloc→add_method→register_and_init
// sequence must be held under this lock.
var ClassRegistrationMu sync.Mutex

// BindMethod registers fn in the callback registry and stores the resulting
// handle as an associated object on obj for the given ObjC selector name.
// The IMP trampoline retrieves this handle at call time via goBridge_Callback_Lookup.
// obj must point to a valid ObjC instance.
func BindMethod(obj unsafe.Pointer, selName string, fn any) uint64 {
	key := Register(fn)
	csel := C.CString(selName)
	C.goBridge_Callback_Bind(obj, csel, C.uint64_t(key))
	C.free(unsafe.Pointer(csel))
	return key
}
