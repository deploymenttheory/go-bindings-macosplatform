//go:build darwin

package cgo

// #cgo CFLAGS: -fno-objc-arc -x objective-c
// #cgo LDFLAGS: -framework Foundation
// #include <dispatch/dispatch.h>
// #include <Foundation/Foundation.h>
// #include <stdlib.h>
//
// extern void goMainCallback(void *ctx);
//
// static void dispatchMainSync(void *ctx) {
//     if ([NSThread isMainThread]) {
//         goMainCallback(ctx);
//     } else {
//         dispatch_sync_f(dispatch_get_main_queue(), ctx, goMainCallback);
//     }
// }
//
// static void dispatchMainAsync(void *ctx) {
//     dispatch_async_f(dispatch_get_main_queue(), ctx, goMainCallback);
// }
//
// static void pumpMainRunLoop(double seconds) {
//     [[NSRunLoop mainRunLoop] runUntilDate:[NSDate dateWithTimeIntervalSinceNow:seconds]];
// }
import "C"

import (
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"unsafe"
)

// OnMainThreadPanic is called when a panic is recovered inside a
// RunOnMainThread or DispatchAsyncMain closure. Wire this to your structured
// logger at application startup. Called on the main OS thread; must not block.
var OnMainThreadPanic func(r interface{}, stack []byte)

// mainPanic captures a recovered panic value from goMainCallback so that
// RunOnMainThread can re-panic on the calling goroutine.
// We use a sync.Map keyed by the same id as mainPending.
var mainPanics sync.Map

type mainEntry struct {
	fn   func()
	done chan struct{}
}

var (
	mainPending sync.Map
	mainIDSeq   int64
)

// RunOnMainThread executes f on the macOS main OS thread and blocks until
// f returns. Required for all AppKit/UIKit operations.
//
// If the calling goroutine is already running on the main thread, f is called
// inline without dispatching.
func RunOnMainThread(f func()) {
	id := atomic.AddInt64(&mainIDSeq, 1)
	done := make(chan struct{})
	mainPending.Store(id, mainEntry{f, done})

	idVal := new(int64)
	*idVal = id
	C.dispatchMainSync(unsafe.Pointer(idVal))
	<-done

	// Re-panic on the calling goroutine if the closure panicked on the main thread.
	if pv, ok := mainPanics.LoadAndDelete(id); ok {
		panic(pv)
	}
}

// DispatchAsyncMain schedules f to run on the macOS main OS thread without
// blocking. The function will be called during the next available main run
// loop iteration. Use this to schedule UI setup after [NSApp run] has started.
func DispatchAsyncMain(f func()) {
	id := atomic.AddInt64(&mainIDSeq, 1)
	mainPending.Store(id, mainEntry{f, nil}) // nil done = fire-and-forget
	idVal := new(int64)
	*idVal = id
	C.dispatchMainAsync(unsafe.Pointer(idVal))
}

// PumpMainRunLoop runs the main NSRunLoop for up to seconds seconds, then
// returns. Drains pending main-queue dispatch items; useful for test binaries
// that need RunOnMainThread to work without a full NSApplicationMain run loop.
func PumpMainRunLoop(seconds float64) {
	C.pumpMainRunLoop(C.double(seconds))
}

//export goMainCallback
func goMainCallback(ctx unsafe.Pointer) {
	id := *(*int64)(ctx)
	val, ok := mainPending.LoadAndDelete(id)
	if !ok {
		return
	}
	entry := val.(mainEntry)

	var panicVal interface{}
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicVal = r
				stack := debug.Stack()
				if h := OnMainThreadPanic; h != nil {
					h(r, stack)
				} else {
					fmt.Fprintf(os.Stderr,
						"\n=== panic inside RunOnMainThread closure ===\n%v\n%s============================================\n",
						r, stack)
				}
			}
		}()
		entry.fn()
	}()

	if panicVal != nil && entry.done != nil {
		mainPanics.Store(id, panicVal)
	}
	if entry.done != nil {
		close(entry.done)
	}
}
