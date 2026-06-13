//go:build darwin

package cgo

// #cgo CFLAGS: -fno-objc-arc -x objective-c
// #cgo LDFLAGS: -framework Foundation
// #include "bridge.h"
// #include <stdlib.h>
import "C"

import "unsafe"

// ExceptionInfo holds structured fields from a caught ObjC NSException.
type ExceptionInfo struct {
	// Name is the exception class name (e.g. "NSInvalidArgumentException").
	Name string
	// Reason is the human-readable failure description.
	Reason string
}

// ExceptionInfoFromPtr extracts structured fields from a caught ObjC NSException,
// releases the exception object, and returns the populated ExceptionInfo.
// Returns zero value if ptr is nil.
func ExceptionInfoFromPtr(ptr unsafe.Pointer) ExceptionInfo {
	if ptr == nil {
		return ExceptionInfo{}
	}
	defer C.runtime_objc_release(ptr)

	var info ExceptionInfo

	nameC := C.runtime_nsexception_name(ptr)
	if nameC != nil {
		info.Name = C.GoString(nameC)
		C.free(unsafe.Pointer(nameC))
	}

	reasonC := C.runtime_nsexception_reason(ptr)
	if reasonC != nil {
		info.Reason = C.GoString(reasonC)
		C.free(unsafe.Pointer(reasonC))
	} else {
		info.Reason = "<no reason>"
	}

	return info
}

// ExceptionReason extracts the reason string from a caught ObjC NSException.
// Deprecated: prefer ExceptionInfoFromPtr for structured access.
func ExceptionReason(ptr unsafe.Pointer) string {
	return ExceptionInfoFromPtr(ptr).Reason
}
