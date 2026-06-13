//go:build darwin

// Package callbacks manages the CGo trampolines that allow ObjC method calls
// on generated subclasses and protocol-conformance objects to invoke Go functions.
//
// ObjC method dispatch is class-based. When the code generator creates a subclass
// or a protocol-impl class at startup (via objc_allocateClassPair), it registers
// generated IMP trampolines for each overridable method. At runtime, when an
// instance is created via [New]NSFooSubclass or [New]NSFooProtocolImpl, the caller
// supplies Go callback functions. Each callback is registered here (assigned a
// uint64 handle) and the handle is stored as an ObjC associated object on the
// instance, keyed by the method's SEL.
//
// When ObjC later invokes the method, the IMP trampoline:
//  1. Calls orinCallbackLookup(self, _cmd) to retrieve the uint64 handle.
//  2. Calls goCallIMP_<sig>(handle, self, args...) — an //export'd Go function.
//  3. goCallIMP_<sig> looks up the handle in the registry and type-asserts the
//     stored func value to the concrete func type for that signature.
//  4. The Go callback is invoked.
//
// IMP type signatures are collected across all frameworks during generation and
// deduplicated; the resulting Go func types are emitted into callbacks_generated.go
// and the C IMP functions into method_trampolines_generated.m by the code generator.
package callbacks
