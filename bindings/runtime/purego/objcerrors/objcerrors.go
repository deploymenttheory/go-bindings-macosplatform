//go:build darwin

// Package objcerrors provides [ObjCError], a structured Go error that preserves
// the full diagnostic payload of an Objective-C NSError: domain, code, localized
// strings, and any chain of underlying errors.
//
// Generated bindings surface errors through [pureobjc.NSErrorToError], which
// returns an *ObjCError wrapped as the error interface. Callers that need
// structured access use [errors.As]:
//
//	var e *objcerrors.ObjCError
//	if errors.As(err, &e) {
//	    log.Printf("domain=%s code=%d reason=%s", e.Domain, e.Code, e.FailureReason)
//	}
package objcerrors

import (
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

var (
	selUTF8String                  = objc.RegisterName("UTF8String")
	selDomain                      = objc.RegisterName("domain")
	selCode                        = objc.RegisterName("code")
	selLocalizedDescription        = objc.RegisterName("localizedDescription")
	selLocalizedFailureReason      = objc.RegisterName("localizedFailureReason")
	selLocalizedRecoverySuggestion = objc.RegisterName("localizedRecoverySuggestion")
	selUnderlyingErrors            = objc.RegisterName("underlyingErrors")
	selRespondsToSelector          = objc.RegisterName("respondsToSelector:")
	selCount                       = objc.RegisterName("count")
	selObjectAtIndex               = objc.RegisterName("objectAtIndex:")
)

// ObjCError is a structured Go error wrapping an Objective-C NSError.
//
// Domain and Code together are the machine-readable identity of the error and
// are stable across OS versions. The string fields are for human display only
// and may be localised. Use [errors.As] to access these fields programmatically.
type ObjCError struct {
	// Domain is the NSError error domain string, e.g. "VZErrorDomain" or
	// "NSPOSIXErrorDomain". Together with Code it uniquely identifies the error kind.
	Domain string

	// Code is the numeric error code within the domain.
	Code int64

	// Description is the localizedDescription — the primary human-readable message.
	Description string

	// FailureReason is the localizedFailureReason — a more specific explanation of
	// why the operation failed. Empty if the NSError did not supply one.
	FailureReason string

	// RecoverySuggestion is the localizedRecoverySuggestion — what the user or
	// caller could do to resolve the error. Empty if the NSError did not supply one.
	RecoverySuggestion string

	// Underlying holds errors from the NSError underlyingErrors array (macOS 11.3+).
	// Nil when the error has no underlying causes.
	Underlying []*ObjCError
}

// Error implements the error interface. The string includes the domain, code,
// and all non-empty localized strings so it is immediately useful in log output
// without requiring a separate errors.As call.
func (e *ObjCError) Error() string {
	s := fmt.Sprintf("[%s:%d] %s", e.Domain, e.Code, e.Description)
	if e.FailureReason != "" {
		s += " — " + e.FailureReason
	}
	if e.RecoverySuggestion != "" {
		s += " (" + e.RecoverySuggestion + ")"
	}
	return s
}

// Unwrap returns the slice of underlying errors so that [errors.Is] and
// [errors.As] traverse the full error tree. Returns nil when there are none.
func (e *ObjCError) Unwrap() []error {
	if len(e.Underlying) == 0 {
		return nil
	}
	out := make([]error, len(e.Underlying))
	for i, u := range e.Underlying {
		out[i] = u
	}
	return out
}

// CFErrorToError converts a CFErrorRef out-parameter to a Go error.
//
// CFError is toll-free bridged with NSError at the ObjC runtime level.
// Casting the CFErrorRef pointer to objc.ID and calling NSError methods via
// objc.Send works without any CGo or C shim — the ObjC runtime dispatches
// to the correct implementation through the toll-free bridge.
//
// Returns nil if ptr is nil (no error was set by the callee).
func CFErrorToError(ptr unsafe.Pointer) error {
	if ptr == nil {
		return nil
	}
	return New(objc.ID(uintptr(ptr)))
}

// New converts an NSError ObjC object to an *ObjCError, extracting all
// available diagnostic fields. Returns nil if id is zero (no error).
func New(id objc.ID) *ObjCError {
	if id == 0 {
		return nil
	}
	return build(id)
}

// build recursively constructs an ObjCError from a non-zero NSError ID.
func build(id objc.ID) *ObjCError {
	e := &ObjCError{
		Domain:             nsstr(objc.Send[objc.ID](id, selDomain)),
		Code:               objc.Send[int64](id, selCode),
		Description:        nsstr(objc.Send[objc.ID](id, selLocalizedDescription)),
		FailureReason:      nsstr(objc.Send[objc.ID](id, selLocalizedFailureReason)),
		RecoverySuggestion: nsstr(objc.Send[objc.ID](id, selLocalizedRecoverySuggestion)),
	}

	// underlyingErrors was added in macOS 11.3. Guard with respondsToSelector: so
	// this is safe to call on any NSError subclass regardless of OS version.
	if objc.Send[bool](id, selRespondsToSelector, selUnderlyingErrors) {
		arr := objc.Send[objc.ID](id, selUnderlyingErrors)
		if arr != 0 {
			n := objc.Send[uint](arr, selCount)
			for i := uint(0); i < n; i++ {
				child := objc.Send[objc.ID](arr, selObjectAtIndex, i)
				if child != 0 {
					e.Underlying = append(e.Underlying, build(child))
				}
			}
		}
	}

	return e
}

// nsstr converts an NSString ObjC ID to a Go string.
// Kept private to this package to avoid a circular dependency with pureobjc.
func nsstr(id objc.ID) string {
	if id == 0 {
		return ""
	}
	return cstrToGo(objc.Send[*byte](id, selUTF8String))
}

// cstrToGo builds a Go string from a null-terminated C string pointer.
func cstrToGo(p *byte) string {
	if p == nil {
		return ""
	}
	n := 0
	base := unsafe.Pointer(p)
	for *(*byte)(unsafe.Pointer(uintptr(base) + uintptr(n))) != 0 {
		n++
	}
	if n == 0 {
		return ""
	}
	return string(unsafe.Slice(p, n))
}
