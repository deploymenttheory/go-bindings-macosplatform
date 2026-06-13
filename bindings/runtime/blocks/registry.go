//go:build darwin

package blocks

// #cgo CFLAGS: -fno-objc-arc -x objective-c
// #include "block_abi.h"
import "C"

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type blockEntry struct {
	fn       any
	refCount int64
}

var (
	blockRegistry sync.Map
	blockKeySeq   uint64
)

// BlockRegister stores fn in the block registry and returns a unique key.
// The reference count starts at 1. ObjC's Block_copy increments it via the
// copy helper; Block_release decrements it via the dispose helper. Call
// FreeBlock after the CGo call returns to release the initial reference.
func BlockRegister(fn any) uint64 {
	key := atomic.AddUint64(&blockKeySeq, 1)
	blockRegistry.Store(key, &blockEntry{fn: fn, refCount: 1})
	return key
}

// blockLookup retrieves the registered closure for key. Returns nil if not found.
func blockLookup(key uint64) any {
	if v, ok := blockRegistry.Load(key); ok {
		return v.(*blockEntry).fn
	}
	return nil
}

// FreeBlock decrements the reference count on block's goKey and frees the
// GoBlock heap allocation. Call this after the CGo call returns for every
// block created by a MakeBlock_* factory.
func FreeBlock(block unsafe.Pointer) {
	if block != nil {
		C.goFreeBlock(block)
	}
}

//export goBlockUnregister
func goBlockUnregister(key C.uint64_t) {
	k := uint64(key)
	if v, ok := blockRegistry.Load(k); ok {
		e := v.(*blockEntry)
		if atomic.AddInt64(&e.refCount, -1) <= 0 {
			blockRegistry.Delete(k)
		}
	}
}

//export goBlockRetain
func goBlockRetain(key C.uint64_t) {
	k := uint64(key)
	if v, ok := blockRegistry.Load(k); ok {
		e := v.(*blockEntry)
		atomic.AddInt64(&e.refCount, 1)
	}
}
