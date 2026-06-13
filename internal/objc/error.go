//go:build darwin

package objc

// #cgo CFLAGS: -fno-objc-arc -x objective-c
// #cgo LDFLAGS: -framework Foundation
// #include "bridge.h"
// #include <stdlib.h>
import "C"

import (
	"fmt"
	"unsafe"
)

// NSError wraps an ObjC NSError pointer as a Go error.
type NSError struct {
	ptr unsafe.Pointer
}

func (e *NSError) Error() string {
	if e == nil || e.ptr == nil {
		return "<nil NSError>"
	}
	cstr := C.runtime_nserror_description(e.ptr)
	if cstr == nil {
		return "<NSError: no description>"
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr)
}

func (e *NSError) Ptr() unsafe.Pointer {
	if e == nil {
		return nil
	}
	return e.ptr
}

// NSErrorToError converts an ObjC NSError pointer to a Go error.
// Releases the pointer and returns nil if ptr is nil.
func NSErrorToError(ptr unsafe.Pointer) error {
	if ptr == nil {
		return nil
	}
	defer C.runtime_objc_release(ptr)
	cstr := C.runtime_nserror_description(ptr)
	if cstr == nil {
		return fmt.Errorf("NSError: <no description>")
	}
	defer C.free(unsafe.Pointer(cstr))
	return fmt.Errorf("%s", C.GoString(cstr))
}
