//go:build darwin

package objc

import "unsafe"

// Object is the constraint satisfied by every generated ObjC wrapper type.
// It guarantees the type exposes its underlying ObjC pointer.
type Object interface {
	Ptr() unsafe.Pointer
}

// WrapObject wraps an ObjC pointer in a minimal Object implementation.
// Used when a method's return type is a generic type parameter and the concrete
// wrapper type is not known at generation time (e.g. NSArray[T].firstObject).
// Returns nil when ptr is nil.
func WrapObject(ptr unsafe.Pointer) Object {
	if ptr == nil {
		return nil
	}
	return &opaqueObject{ptr: ptr}
}

// WrapTyped wraps an ObjC pointer in a typed Go wrapper using the provided
// constructor. The constructor is called only when ptr is non-nil. It is the
// centralised nil-guard used by generated method returns and block callbacks:
//
//	// generated method return
//	return objc.WrapTyped(_ptr, NewVZVirtualMachine)
//
//	// generated block callback
//	blocks.MakeBlock_void_ptr(func(_p0 unsafe.Pointer) {
//	    completionHandler(objc.WrapTyped(_p0, NewVZMacOSRestoreImage), objc.NSErrorToError(_p1))
//	})
//
// constructor must not be nil.
func WrapTyped[T any](ptr unsafe.Pointer, constructor func(unsafe.Pointer) T) (result T) {
	if ptr == nil {
		return
	}
	return constructor(ptr)
}

type opaqueObject struct{ ptr unsafe.Pointer }

func (o *opaqueObject) Ptr() unsafe.Pointer { return o.ptr }
