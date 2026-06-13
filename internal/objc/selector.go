//go:build darwin

package objc

// #cgo CFLAGS: -fno-objc-arc -x objective-c
// #cgo LDFLAGS: -framework Foundation
// #include "bridge.h"
// #include <stdlib.h>
import "C"

import "unsafe"

var selCache SyncCache[string, unsafe.Pointer]

// Sel returns the ObjC SEL for name, caching after the first lookup.
// Equivalent to @selector(name) in Objective-C.
func Sel(name string) unsafe.Pointer {
	return selCache.Load(name, func(n string) unsafe.Pointer {
		cs := C.CString(n)
		defer C.free(unsafe.Pointer(cs))
		return C.runtime_sel_register_name(cs)
	})
}
