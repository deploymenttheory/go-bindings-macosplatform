//go:build darwin

package objc

// #cgo CFLAGS: -fno-objc-arc -x objective-c
// #cgo LDFLAGS: -framework Foundation
// #include "bridge.h"
// #include <stdlib.h>
import "C"

import "unsafe"

// NSStringToGoString converts an ObjC NSString pointer to a Go string.
// Returns an empty string if ptr is nil.
func NSStringToGoString(ptr unsafe.Pointer) string {
	if ptr == nil {
		return ""
	}
	cstr := C.runtime_nsstring_to_cstring(ptr)
	if cstr == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr)
}

// GoStringToNSString converts a Go string to an ObjC NSString (+1 retain).
// The caller (or finalizer) is responsible for releasing the returned pointer.
func GoStringToNSString(s string) unsafe.Pointer {
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	return C.runtime_nsstring_from_cstring(cs)
}
