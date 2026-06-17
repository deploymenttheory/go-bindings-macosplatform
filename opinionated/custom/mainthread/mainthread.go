//go:build darwin

// Package mainthread dispatches Go functions onto the macOS main thread.
//
// AppKit (and most UI frameworks) require their calls to run on the process
// main thread; the generated bindings do not dispatch automatically. This
// package provides that dispatch without CGo, using GCD's main queue via
// purego.
//
// The main queue is only drained while the main thread is servicing it — by
// AppKit's run loop ([NSApplication run]), a Core Foundation run loop, or
// dispatch_main. A typical UI app locks the main goroutine and hands the
// thread to AppKit:
//
//	func init() { runtime.LockOSThread() }
//
//	func main() {
//	    go worker() // calls mainthread.Do(...) for UI work
//	    appkit.NSApplicationSharedApplication().Run()
//	}
//
// Calling Do from a goroutine while nothing runs the main run loop blocks
// until the queue is next drained.
package mainthread

import (
	"sync"

	"github.com/ebitengine/purego"
)

var (
	loadOnce sync.Once

	mainQueue     uintptr
	dispatchSyncF func(queue, ctx uintptr, work uintptr)
	pthreadMainNp func() int32
	trampoline    uintptr

	pendingMu sync.Mutex
	pending   = make(map[uintptr]func())
	nextKey   uintptr
)

// load resolves the GCD symbols from libSystem (always present on macOS).
func load() {
	libSystem, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_GLOBAL|purego.RTLD_LAZY)
	if err != nil {
		panic("mainthread: dlopen libSystem: " + err.Error())
	}
	// dispatch_get_main_queue() is a macro for &_dispatch_main_q — the
	// symbol's address is the dispatch_queue_t itself.
	mainQueue, err = purego.Dlsym(libSystem, "_dispatch_main_q")
	if err != nil {
		panic("mainthread: dlsym _dispatch_main_q: " + err.Error())
	}
	purego.RegisterLibFunc(&dispatchSyncF, libSystem, "dispatch_sync_f")
	purego.RegisterLibFunc(&pthreadMainNp, libSystem, "pthread_main_np")
	trampoline = purego.NewCallback(runPending)
}

// runPending is the dispatch_function_t target: it looks up and runs the Go
// function registered under key.
func runPending(key uintptr) uintptr {
	pendingMu.Lock()
	fn := pending[key]
	delete(pending, key)
	pendingMu.Unlock()
	if fn != nil {
		fn()
	}
	return 0
}

// IsMain reports whether the calling goroutine is currently executing on the
// process main thread.
func IsMain() bool {
	loadOnce.Do(load)
	return pthreadMainNp() != 0
}

// Do runs fn on the macOS main thread and blocks until it returns. When
// already on the main thread, fn runs inline (dispatching synchronously to
// the main queue from the main thread would deadlock).
//
// fn only starts once the main thread services its queue — see the package
// documentation for the required run-loop setup.
func Do(fn func()) {
	loadOnce.Do(load)
	if pthreadMainNp() != 0 {
		fn()
		return
	}

	pendingMu.Lock()
	nextKey++
	key := nextKey
	pending[key] = fn
	pendingMu.Unlock()

	// dispatch_sync_f blocks the calling thread until the work item has run
	// on the main queue, so fn's effects are visible when Do returns.
	dispatchSyncF(mainQueue, key, trampoline)
}
