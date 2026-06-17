//go:build darwin

package purego

import (
	ebipurego "github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var (
	_dispatchSync func(queue uintptr, block objc.Block)
	_mainQueue    uintptr
	_mainReady    bool

	clsNSThread     = objc.GetClass("NSThread")
	selIsMainThread = objc.RegisterName("isMainThread")
)

func init() {
	// libdispatch ships inside libSystem; dispatch_get_main_queue() is the
	// address of the global _dispatch_main_q symbol.
	h, err := ebipurego.Dlopen("/usr/lib/libSystem.B.dylib", ebipurego.RTLD_GLOBAL|ebipurego.RTLD_LAZY)
	if err != nil {
		return
	}
	q, err := ebipurego.Dlsym(h, "_dispatch_main_q")
	if err != nil || q == 0 {
		return
	}
	_mainQueue = q
	ebipurego.RegisterLibFunc(&_dispatchSync, h, "dispatch_sync")
	_mainReady = _dispatchSync != nil
}

// OnMainThread reports whether the caller is currently running on the main thread.
func OnMainThread() bool {
	return objc.Send[bool](objc.ID(clsNSThread), selIsMainThread)
}

// Main runs fn on the main thread and blocks until it completes. AppKit/UIKit
// calls must originate on the main thread; wrap them with Main. When already on
// the main thread fn runs inline (a dispatch_sync to the main queue from the
// main thread would deadlock); if the dispatch runtime is unavailable, fn runs
// inline as a best effort.
func Main(fn func()) {
	if !_mainReady || OnMainThread() {
		fn()
		return
	}
	block := objc.NewBlock(func(_ objc.Block) { fn() })
	defer block.Release()
	_dispatchSync(_mainQueue, block)
}
