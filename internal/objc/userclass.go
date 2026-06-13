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

// MethodBinding describes one ObjC method implementation to install on a user
// class. Imp must be a C function pointer whose signature matches TypeEncoding,
// typically produced by imp_implementationWithBlock (runtime_imp_from_block).
type MethodBinding struct {
	// Selector is the ObjC selector string, e.g. "applicationDidFinishLaunching:".
	Selector string
	// TypeEncoding is the ObjC type encoding for the method, e.g. "v@:@".
	TypeEncoding string
	// Imp is the C function pointer for this method's implementation.
	// Obtain via MakeIMP or runtime_imp_from_block.
	Imp unsafe.Pointer
}

// UserClassHandle represents a Go-backed ObjC class registered with the runtime.
// Instances are created via NewInstance; the class lives for the process lifetime.
type UserClassHandle struct {
	ptr  unsafe.Pointer // Class *
	name string
}

// RegisterUserClass allocates and registers an ObjC class named name as a
// subclass of superName, conforms it to the named protocols, and installs
// each method binding. Panics if the class is already registered or if
// superName does not exist in the ObjC runtime.
//
// Classes are process-lifetime objects. Call this once at startup (or via
// sync.Once) rather than per-request.
func RegisterUserClass(name, superName string, protocols []string, methods []MethodBinding) *UserClassHandle {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	superCls := C.runtime_get_class(C.CString(superName))
	if superCls == nil {
		panic(fmt.Sprintf("objc.RegisterUserClass: superclass %q not found in ObjC runtime", superName))
	}

	cls := C.runtime_class_alloc_pair(superCls, cName)
	if cls == nil {
		panic(fmt.Sprintf("objc.RegisterUserClass: could not allocate class pair %q (already registered?)", name))
	}

	for _, p := range protocols {
		cp := C.CString(p)
		proto := C.runtime_get_protocol(cp)
		C.free(unsafe.Pointer(cp))
		if proto != nil {
			C.runtime_class_add_protocol(cls, proto)
		}
	}

	for _, m := range methods {
		sel := Sel(m.Selector)
		cTypes := C.CString(m.TypeEncoding)
		C.runtime_class_add_method(cls, sel, m.Imp, cTypes)
		C.free(unsafe.Pointer(cTypes))
	}

	C.runtime_class_register_pair(cls)
	return &UserClassHandle{ptr: cls, name: name}
}

// NewInstance allocates and initialises an instance of the user class.
// Returns a +1-retained pointer; the caller must call Track or Release when done.
func (h *UserClassHandle) NewInstance() unsafe.Pointer {
	raw := AllocObject(h.name)
	if raw == nil {
		return nil
	}
	return C.runtime_send_init(raw)
}

// Name returns the ObjC class name.
func (h *UserClassHandle) Name() string { return h.name }

// Dispose disposes the class pair. Must only be called after all instances have
// been released. Classes are normally process-lifetime — only dispose in tests.
func (h *UserClassHandle) Dispose() {
	if h.ptr != nil {
		C.runtime_class_dispose_pair(h.ptr)
		h.ptr = nil
	}
}

// MakeIMP converts an ObjC block (created via imp_implementationWithBlock) into
// a C IMP function pointer suitable for MethodBinding.Imp.
//
// block must be an ObjC block value (id). Ownership of the block is transferred
// to the IMP; do not release it after calling MakeIMP.
func MakeIMP(block unsafe.Pointer) unsafe.Pointer {
	return C.runtime_imp_from_block(block)
}
