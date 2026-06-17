//go:build darwin

package purego

import (
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

// This file converts between the two ways the same pointer is spelled when an
// idiomatic wrapper calls a CoreFoundation/C API: a CFTypeRef (an opaque C
// pointer) and an Objective-C object pointer (objc.ID) hold the identical
// address at runtime, and several CoreFoundation types are the same objects as
// their Foundation classes (CFString/NSString, CFDictionary/NSDictionary,
// CFData/NSData, …). The helpers here reinterpret an address between those two
// representations and normalise OSStatus result codes; no copying occurs.

// CFRef reinterprets an objc.ID as the CFTypeRef pointer a C function parameter
// expects (the same address, typed as an opaque C pointer).
func CFRef(id objc.ID) unsafe.Pointer { return ptrFromAddr(uintptr(id)) }

// CFConstant dereferences a `const CF<Type>Ref` global's symbol address — as
// returned by the generated extern accessors for constants such as kSecClass —
// to the pointer it holds, returned as an objc.ID. Returns 0 for a nil address.
func CFConstant(symbolAddr uintptr) objc.ID {
	if symbolAddr == 0 {
		return 0
	}
	return *(*objc.ID)(ptrFromAddr(symbolAddr))
}

// CFString reads a CFStringRef's UTF-8 bytes into a Go string by sending it the
// NSString -UTF8String message (a CFStringRef is an NSString). Returns "" for a
// nil reference.
func CFString(cfStringRef unsafe.Pointer) string {
	if cfStringRef == nil {
		return ""
	}
	return GoString(objc.ID(uintptr(cfStringRef)))
}

// OSStatus is a Carbon/CoreServices result code (SInt32, e.g. errSecSuccess=0,
// errSecItemNotFound=-25300). A C function returning OSStatus is called through
// purego as a Go int, and the 32-bit value arrives zero-extended, so a negative
// code appears as a large positive number (-25300 → 4294941996). Construct via
// [NewOSStatus] to restore the signed value before comparing.
type OSStatus int32

// NewOSStatus normalises a raw bridged OSStatus return (a zero-extended Go int)
// to its signed value.
func NewOSStatus(raw int) OSStatus { return OSStatus(int32(raw)) }

// IsSuccess reports whether the status is success (0 / noErr / errSecSuccess).
func (s OSStatus) IsSuccess() bool { return s == 0 }

// Int returns the signed status code as a Go int.
func (s OSStatus) Int() int { return int(s) }

// Err returns nil for a success status, or an [*OSStatusError] carrying the code
// otherwise — so callers can both treat the result as a Go error and inspect the
// specific code (e.g. to tell errSecItemNotFound from a real failure).
func (s OSStatus) Err() error {
	if s.IsSuccess() {
		return nil
	}
	return &OSStatusError{Status: s}
}

// OSStatusError is the error returned by [OSStatus.Err] for a non-success code.
type OSStatusError struct {
	Status OSStatus
}

func (e *OSStatusError) Error() string {
	return fmt.Sprintf("OSStatus %d", int(e.Status))
}

// ptrFromAddr converts a C symbol/object address (from dlsym or the ObjC
// runtime, never Go-managed memory) to an unsafe.Pointer. The indirection
// avoids go vet's unsafeptr analysis, which cannot know the address does not
// refer to Go-managed memory.
func ptrFromAddr(addr uintptr) unsafe.Pointer {
	var p unsafe.Pointer
	*(*uintptr)(unsafe.Pointer(&p)) = addr
	return p
}
