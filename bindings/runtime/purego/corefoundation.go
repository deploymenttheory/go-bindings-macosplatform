//go:build darwin

package purego

import (
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

// This file provides the CoreFoundation toll-free-bridging surface the idiomatic
// layer needs to use C APIs (Security keychain, etc.) without dropping to raw
// FFI: passing objects as CFTypeRef, dereferencing CF constant symbols, and
// normalising OSStatus return codes. CoreFoundation and Foundation objects are
// toll-free bridged, so a CFTypeRef and an objc.ID are the same pointer.

// CFRef passes an ObjC/CoreFoundation object id to a C function expecting a
// CFTypeRef-style argument (an opaque pointer). It is the inverse of receiving a
// CFTypeRef back as an objc.ID.
func CFRef(id objc.ID) unsafe.Pointer { return ptrFromAddr(uintptr(id)) }

// CFConstant dereferences a CoreFoundation constant's symbol address — as
// returned by the generated extern accessors for `const CF<Type>Ref` globals
// such as kSecClass — to the CFTypeRef value it points at, typed as an objc.ID.
// Returns 0 for a nil address.
func CFConstant(symbolAddr uintptr) objc.ID {
	if symbolAddr == 0 {
		return 0
	}
	return *(*objc.ID)(ptrFromAddr(symbolAddr))
}

// OSStatus is a Carbon/CoreServices result code (SInt32, e.g. errSecSuccess=0,
// errSecItemNotFound=-25300). C functions returning OSStatus are bridged as a Go
// int and the value arrives zero-extended, so a negative code appears as a large
// positive number (-25300 → 4294941996). Construct via [NewOSStatus] to restore
// the signed value before comparing.
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
