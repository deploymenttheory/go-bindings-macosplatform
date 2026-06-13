//go:build darwin

// Package pureobjc provides the runtime support layer for purego-based macOS
// framework bindings. It exposes retain/release/track helpers and string
// conversion utilities that generated packages import. Zero CGo dependency.
package purego

import (
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego/objcerrors"
)

var (
	selRetain     = objc.RegisterName("retain")
	selRelease    = objc.RegisterName("release")
	selUTF8String = objc.RegisterName("UTF8String")

	clsNSString = objc.GetClass("NSString")

	selStringWithUTF8String = objc.RegisterName("stringWithUTF8String:")
)

// Object is the constraint for generated ObjC wrapper pointer types.
// Every generated struct (e.g. *NSString, *NSArray) satisfies this interface.
type Object interface {
	Ptr() objc.ID
}

// AnyObject is the constraint for ObjC generic type parameters (NSArray[T],
// NSDictionary[K,V], etc). Using any allows both raw objc.ID handles and
// fully typed wrapper structs as type arguments without unsafe.Pointer fallback.
type AnyObject = any

// Track registers a Go finalizer on o that releases the underlying ObjC object
// when o is garbage collected. o must be a pointer type.
func Track[T Object](o T) {
	runtime.SetFinalizer(o, func(o T) {
		ptr := o.Ptr()
		if ptr != 0 {
			ptr.Send(selRelease)
		}
	})
}

// Retain sends -retain to the given ObjC object and returns it unchanged.
func Retain(id objc.ID) objc.ID {
	if id != 0 {
		id.Send(selRetain)
	}
	return id
}

// Release sends -release to the given ObjC object.
func Release(id objc.ID) {
	if id != 0 {
		id.Send(selRelease)
	}
}

// GoString converts an NSString ObjC object to a Go string.
func GoString(nsstr objc.ID) string {
	if nsstr == 0 {
		return ""
	}
	cstr := objc.Send[*byte](nsstr, selUTF8String)
	if cstr == nil {
		return ""
	}
	return cstrToGoString(cstr)
}

// NSString converts a Go string to an autoreleased NSString object.
// The returned ID is autoreleased; retain it if you need it to outlive the
// current autorelease pool drain.
func NSString(s string) objc.ID {
	return objc.Send[objc.ID](objc.ID(clsNSString), selStringWithUTF8String, s)
}

// NSErrorToError converts an NSError ObjC object to a Go error.
// The returned error is *[objcerrors.ObjCError]; use [errors.As] to access
// structured fields such as Domain, Code, and FailureReason.
func NSErrorToError(id objc.ID) error {
	if id == 0 {
		return nil
	}
	return objcerrors.New(id)
}

// GoCString builds a Go string from the address of a null-terminated C
// string. Generated block adapters use it to convert char* block arguments
// (purego block callbacks cannot take string parameters directly).
func GoCString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	p := (*byte)(
		unsafe.Pointer(ptr),
	) //nolint:govet // C-owned memory, valid for the callback's duration
	return cstrToGoString(p)
}

// cstrToGoString builds a Go string from a null-terminated C string pointer.
func cstrToGoString(p *byte) string {
	if p == nil {
		return ""
	}
	n := 0
	ptr := unsafe.Pointer(p)
	for {
		b := *(*byte)(unsafe.Pointer(uintptr(ptr) + uintptr(n)))
		if b == 0 {
			break
		}
		n++
	}
	if n == 0 {
		return ""
	}
	return string(unsafe.Slice(p, n))
}

// BoolToObjC converts a Go bool to the ObjC BOOL representation (1 or 0).
func BoolToObjC(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}
