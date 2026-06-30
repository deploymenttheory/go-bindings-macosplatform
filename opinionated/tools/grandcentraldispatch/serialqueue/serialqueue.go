//go:build darwin

// Package serialqueue runs Go functions on a dedicated GCD serial dispatch
// queue, off the main thread, without CGo (purego over libdispatch).
//
// It is the queue-confined sibling of opinionated/tools/grandcentraldispatch/mainthread: where that
// package serialises work onto the process main queue (for AppKit's @MainActor
// isolation), this one creates an independent serial queue for frameworks that
// are queue-confined rather than main-actor — notably Virtualization, whose
// VZVirtualMachine must be used on the dispatch queue it was created on
// (init(configuration:queue:)). Pass Queue.Handle() as that queue, then drive
// every VZVirtualMachine call through Queue.Do so they all run on it.
//
// Do runs inline when the caller is already on the queue (so a re-entrant call
// from a delegate callback or a nested Do does not deadlock), otherwise it
// dispatch_sync's and blocks until the work completes. Unlike the main queue,
// a serial queue is serviced by its own GCD-managed thread, so no run loop needs
// to be pumped for Do to make progress.
package serialqueue

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	loadOnce sync.Once

	dispatchQueueCreate      func(label string, attr uintptr) uintptr
	dispatchSyncF            func(queue, ctx, work uintptr)
	dispatchQueueSetSpecific func(queue, key, ctx, destructor uintptr)
	dispatchGetSpecific      func(key uintptr) uintptr
	trampoline               uintptr

	pendingMu sync.Mutex
	pending   = make(map[uintptr]func())
	nextKey   uintptr
)

// load resolves the libdispatch symbols from libSystem (always present on macOS).
func load() {
	libSystem, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_GLOBAL|purego.RTLD_LAZY)
	if err != nil {
		panic("serialqueue: dlopen libSystem: " + err.Error())
	}
	purego.RegisterLibFunc(&dispatchQueueCreate, libSystem, "dispatch_queue_create")
	purego.RegisterLibFunc(&dispatchSyncF, libSystem, "dispatch_sync_f")
	purego.RegisterLibFunc(&dispatchQueueSetSpecific, libSystem, "dispatch_queue_set_specific")
	purego.RegisterLibFunc(&dispatchGetSpecific, libSystem, "dispatch_get_specific")
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

// Queue is a serial dispatch queue. The zero value is not usable; create one
// with New. A Queue's underlying dispatch_queue_t lives for the process
// lifetime (it is never released), matching the lifetime of the object bound to
// it (e.g. a VZVirtualMachine).
type Queue struct {
	q   uintptr // dispatch_queue_t (serial)
	key uintptr // dispatch specific key/value identifying this queue
}

// New creates a serial dispatch queue with the given label.
func New(label string) *Queue {
	loadOnce.Do(load)
	q := &Queue{q: dispatchQueueCreate(label, 0)} // attr == NULL → serial
	// Tag the queue with a unique specific so Do can detect re-entrancy. The
	// key and value are both the Queue's own address: get_specific returns the
	// value (non-zero) exactly when the caller is running on this queue.
	q.key = uintptr(unsafe.Pointer(q))
	dispatchQueueSetSpecific(q.q, q.key, q.key, 0)
	return q
}

// Do runs fn on the queue and blocks until it returns. When the caller is
// already on this queue, fn runs inline (a dispatch_sync onto the current serial
// queue would deadlock).
func (q *Queue) Do(fn func()) {
	loadOnce.Do(load)
	if dispatchGetSpecific(q.key) != 0 {
		fn()
		return
	}

	pendingMu.Lock()
	nextKey++
	key := nextKey
	pending[key] = fn
	pendingMu.Unlock()

	// dispatch_sync_f blocks the calling thread until the work item has run on
	// the queue, so fn's effects are visible when Do returns.
	dispatchSyncF(q.q, key, trampoline)
}

// Handle returns the underlying dispatch_queue_t as a uintptr, for APIs that
// take a dispatch queue (e.g. VZVirtualMachine init(configuration:queue:)).
func (q *Queue) Handle() uintptr { return q.q }
