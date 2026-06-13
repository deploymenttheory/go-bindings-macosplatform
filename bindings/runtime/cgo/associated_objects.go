//go:build darwin

package cgo

// #cgo CFLAGS: -fno-objc-arc -x objective-c
// #cgo LDFLAGS: -framework Foundation
// #include "bridge.h"
// #include <stdlib.h>
import "C"

import (
	"sync"
	"unsafe"
)

// Association policies for SetAssociatedObject, mirroring objc_AssociationPolicy.
const (
	AssocAssign          uintptr = 0
	AssocRetainNonatomic uintptr = 1
	AssocCopyNonatomic   uintptr = 3
	AssocRetain          uintptr = 0x301
	AssocCopy            uintptr = 0x303
)

// assocKeyCache stores stable *C.char pointers keyed by Go string so that the
// same key string always maps to the same C pointer (required by the ObjC
// associated-object API, which uses pointer identity for key lookup).
var assocKeyCache sync.Map // map[string]*C.char

func assocKey(name string) unsafe.Pointer {
	if v, ok := assocKeyCache.Load(name); ok {
		return unsafe.Pointer(v.(*C.char))
	}
	// CString is not freed — the pointer must be stable for the process lifetime.
	cs := C.CString(name)
	actual, loaded := assocKeyCache.LoadOrStore(name, cs)
	if loaded {
		// Another goroutine stored first; free ours and use theirs.
		C.free(unsafe.Pointer(cs))
		return unsafe.Pointer(actual.(*C.char))
	}
	return unsafe.Pointer(cs)
}

// SetAssociatedObject attaches value to obj under key using the given policy.
// key is a Go string; pointer identity is handled internally via a stable cache.
func SetAssociatedObject(obj unsafe.Pointer, key string, value unsafe.Pointer, policy uintptr) {
	C.runtime_set_associated_object(obj, assocKey(key), value, C.uintptr_t(policy))
}

// GetAssociatedObject retrieves the value previously attached to obj under key.
// Returns nil if no value has been set.
func GetAssociatedObject(obj unsafe.Pointer, key string) unsafe.Pointer {
	return C.runtime_get_associated_object(obj, assocKey(key))
}

// RemoveAssociatedObjects removes all associated objects from obj.
func RemoveAssociatedObjects(obj unsafe.Pointer) {
	C.runtime_remove_associated_objects(obj)
}
