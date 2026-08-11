//go:build darwin

package puregolibs_test

import (
	"sync"
	"testing"

	dispatch "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/dispatch"
)

// TestDispatch_MainQueueResolves proves the hand-written Dispatch_get_main_queue
// reimplementation (dispatch_get_main_queue is a header-inline with no exported
// symbol; it returns the address of the exported _dispatch_main_q global). A
// non-nil, stable pointer means the Dlsym reimplementation found the global.
func TestDispatch_MainQueueResolves(t *testing.T) {
	q1 := dispatch.Dispatch_get_main_queue()
	if q1 == nil {
		t.Fatal("Dispatch_get_main_queue() = nil; _dispatch_main_q did not resolve")
	}
	if q2 := dispatch.Dispatch_get_main_queue(); q2 != q1 {
		t.Errorf("main queue not stable across calls: %p then %p", q1, q2)
	}
}

// TestDispatch_SyncBlockRuns runs a block synchronously on a global concurrent
// queue: the block must execute (on a real GCD worker thread) before
// dispatch_sync returns. This is the core block-crossing proof for dispatch —
// the objc.NewBlock adapter is invoked by libdispatch itself.
func TestDispatch_SyncBlockRuns(t *testing.T) {
	const dispatchQueuePriorityDefault = 0
	q := dispatch.Dispatch_get_global_queue(dispatchQueuePriorityDefault, 0)
	if q == nil {
		t.Fatal("dispatch_get_global_queue returned nil")
	}
	ran := false
	dispatch.Dispatch_sync(q, func() { ran = true })
	if !ran {
		t.Error("dispatch_sync returned without running the block")
	}
}

// TestDispatch_AsyncOnCustomQueue creates a serial queue, dispatches several
// blocks asynchronously, and waits for them — exercising queue creation, async
// block delivery on GCD threads, and lifetime (release). A WaitGroup provides
// the happens-before the test needs without touching the main run loop.
func TestDispatch_AsyncOnCustomQueue(t *testing.T) {
	q := dispatch.Dispatch_queue_create("com.weave.puregolibs.test", nil)
	if q == nil {
		t.Fatal("dispatch_queue_create returned nil")
	}
	defer dispatch.Dispatch_release(q)

	const n = 8
	var wg sync.WaitGroup
	wg.Add(n)
	var mu sync.Mutex
	count := 0
	for i := 0; i < n; i++ {
		dispatch.Dispatch_async(q, func() {
			mu.Lock()
			count++
			mu.Unlock()
			wg.Done()
		})
	}
	wg.Wait()
	if count != n {
		t.Errorf("dispatch_async ran %d of %d blocks", count, n)
	}
}
