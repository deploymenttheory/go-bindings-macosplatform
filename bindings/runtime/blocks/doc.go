//go:build darwin

// Package blocks manages the CGo trampolines that allow ObjC block callbacks
// to invoke Go functions.
//
// ObjC blocks are typed function-pointer closures. When generated code passes
// a Go func value to an ObjC API that expects a block, the trampoline layer
// bridges the two calling conventions:
//
//  1. The Go func is registered in a thread-safe registry and assigned a
//     numeric handle.
//  2. A C block is constructed that, when called by ObjC, looks up the handle
//     and dispatches to the stored Go func.
//  3. The registry uses atomic reference counting so the Go func is not
//     garbage-collected while ObjC holds a reference to the block.
//
// Block type signatures are collected across all frameworks during generation
// and deduplicated; the resulting Go func types are emitted into
// blocks_generated.go by the code generator.
package blocks
