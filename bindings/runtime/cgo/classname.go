//go:build darwin

package cgo

// #include <objc/runtime.h>
//
// static inline const char* _runtime_class_name(void* ptr) {
//     if (!ptr) return "";
//     return class_getName(object_getClass((id)ptr));
// }
import "C"

import "unsafe"

// ClassNameOf returns the ObjC runtime class name of the object at ptr,
// or "" if ptr is nil. Wraps object_getClass() + class_getName().
func ClassNameOf(ptr unsafe.Pointer) string {
	return C.GoString(C._runtime_class_name(ptr))
}
