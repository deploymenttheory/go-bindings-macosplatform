//go:build darwin

package purego

// This file re-exports the dynamic-loading surface of the top-level
// github.com/ebitengine/purego package (dlopen/dlsym + Go↔C function
// registration). Combined with the conversion helpers and the ObjC dispatch
// re-exports in this package, a consumer of the generated bindings can resolve
// and call arbitrary C symbols at runtime without ever importing
// github.com/ebitengine/purego directly — the rule stays: import only
// bindings/runtime/... .

import ebipurego "github.com/ebitengine/purego"

// dlopen mode flags (re-exported so callers need not import ebitengine/purego).
const (
	RTLD_DEFAULT = ebipurego.RTLD_DEFAULT // pseudo-handle: search every loaded image
	RTLD_LAZY    = ebipurego.RTLD_LAZY
	RTLD_NOW     = ebipurego.RTLD_NOW
	RTLD_LOCAL   = ebipurego.RTLD_LOCAL
	RTLD_GLOBAL  = ebipurego.RTLD_GLOBAL
)

// Dlopen loads the dynamic library at path with the given mode and returns its handle.
func Dlopen(path string, mode int) (uintptr, error) { return ebipurego.Dlopen(path, mode) }

// Dlsym resolves the address of symbol name within the library handle (or RTLD_DEFAULT).
func Dlsym(handle uintptr, name string) (uintptr, error) { return ebipurego.Dlsym(handle, name) }

// Dlclose decrements the reference count of the library handle.
func Dlclose(handle uintptr) error { return ebipurego.Dlclose(handle) }

// RegisterFunc binds the Go function pointer fptr to the C function at address cfn.
// fptr must be a pointer to a func variable with a C-compatible signature.
func RegisterFunc(fptr any, cfn uintptr) { ebipurego.RegisterFunc(fptr, cfn) }

// RegisterLibFunc resolves name in handle and binds it to the Go function pointer fptr.
func RegisterLibFunc(fptr any, handle uintptr, name string) {
	ebipurego.RegisterLibFunc(fptr, handle, name)
}

// NewCallback converts a Go function into a C function pointer usable as a callback.
func NewCallback(fn any) uintptr { return ebipurego.NewCallback(fn) }

// SyscallN calls the C function at fn with the given arguments.
func SyscallN(fn uintptr, args ...uintptr) (r1, r2, err uintptr) {
	return ebipurego.SyscallN(fn, args...)
}
