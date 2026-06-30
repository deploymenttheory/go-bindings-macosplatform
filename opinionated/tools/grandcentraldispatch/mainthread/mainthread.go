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
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	loadOnce sync.Once

	mainQueue     uintptr
	dispatchSyncF func(queue, ctx uintptr, work uintptr)
	pthreadMainNp func() int32
	dispatchMain  func()
	trampoline    uintptr

	// CoreFoundation run-loop symbols, resolved in load(), used by
	// PumpMainRunLoop.
	cfRunLoopRunInMode    func(mode uintptr, seconds float64, returnAfterSourceHandled bool) int32
	kCFRunLoopDefaultMode uintptr

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
	purego.RegisterLibFunc(&dispatchMain, libSystem, "dispatch_main")
	trampoline = purego.NewCallback(runPending)

	// CoreFoundation run-loop, for PumpMainRunLoop. kCFRunLoopDefaultMode is a
	// `const CFStringRef` variable, so the dlsym address must be dereferenced
	// once to obtain the CFStringRef value (unlike _dispatch_main_q, whose
	// address is the queue itself).
	cf, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_GLOBAL|purego.RTLD_LAZY)
	if err != nil {
		panic("mainthread: dlopen CoreFoundation: " + err.Error())
	}
	modeSym, err := purego.Dlsym(cf, "kCFRunLoopDefaultMode")
	if err != nil {
		panic("mainthread: dlsym kCFRunLoopDefaultMode: " + err.Error())
	}
	// Dereference the symbol address to read the CFStringRef it holds. The
	// indirection through &p turns the C address into an unsafe.Pointer without
	// a direct uintptr→Pointer conversion, which `go vet` rejects.
	var p unsafe.Pointer
	*(*uintptr)(unsafe.Pointer(&p)) = modeSym
	kCFRunLoopDefaultMode = *(*uintptr)(p)
	purego.RegisterLibFunc(&cfRunLoopRunInMode, cf, "CFRunLoopRunInMode")
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

// PumpMainRunLoop services the main thread's Core Foundation run loop for up to
// seconds, draining any pending main-queue work (such as functions queued by Do
// from other goroutines) before returning. It must be called on the main
// thread.
//
// Use it when a program drives the main thread itself — e.g. running background
// work on goroutines while periodically pumping the run loop — instead of
// handing the thread to AppKit's [NSApplication run] or dispatch_main. A small
// interval (e.g. 0.05) polled in a loop keeps Do responsive while the caller
// checks for completion.
func PumpMainRunLoop(seconds float64) {
	loadOnce.Do(load)
	// returnAfterSourceHandled=false: run the loop for the full interval,
	// servicing as many queued items as arrive rather than returning after the
	// first.
	cfRunLoopRunInMode(kCFRunLoopDefaultMode, seconds, false)
}

// DispatchMain hands the calling thread to GCD to service the main dispatch
// queue, and never returns (it is dispatch_main). Call it on the process main
// thread when the program drives its own work on other goroutines and only needs
// the main queue kept alive — e.g. so the idiomatic layer's @MainActor
// auto-dispatch (and Do) keep working — without bringing up AppKit. Unlike
// PumpMainRunLoop, it does not spin: the thread blocks in libdispatch until a
// queued item arrives. The program must terminate some other way (e.g. os.Exit
// from a worker goroutine), since DispatchMain itself never returns.
func DispatchMain() {
	loadOnce.Do(load)
	dispatchMain()
}
