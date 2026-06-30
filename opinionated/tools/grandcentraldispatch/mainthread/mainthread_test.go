//go:build darwin

package mainthread_test

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/tools/grandcentraldispatch/mainthread"
)

// TestMain pins the main goroutine to the process main thread and keeps it
// servicing the main run loop (which drains the main GCD queue) while the
// tests run on their own goroutines — mirroring how a real app hands the
// main thread to AppKit's run loop.
func TestMain(m *testing.M) {
	runtime.LockOSThread()
	go func() {
		os.Exit(m.Run())
	}()
	for {
		foundation.NSRunLoopMainRunLoop().
			RunUntilDate(foundation.NSDateDateWithTimeIntervalSinceNow(0.05))
	}
}

func TestIsMainFromTestGoroutine(t *testing.T) {
	// Test functions run on their own goroutines, off the main thread.
	if mainthread.IsMain() {
		t.Fatal("IsMain() = true on a test goroutine")
	}
}

func TestDoRunsOnMainThread(t *testing.T) {
	onMain := make(chan bool, 1)
	done := make(chan struct{})
	go func() {
		mainthread.Do(func() {
			onMain <- mainthread.IsMain()
		})
		close(done)
	}()

	select {
	case got := <-onMain:
		if !got {
			t.Fatal("Do body did not run on the main thread")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("dispatched function never ran — main queue not drained")
	}
	select {
	case <-done:
		// Do returned only after the body completed (dispatch_sync semantics).
	case <-time.After(10 * time.Second):
		t.Fatal("Do did not return after the body completed")
	}
}

func TestDoNestedRunsInline(t *testing.T) {
	// A Do issued from code already on the main thread must run inline —
	// dispatching synchronously to the main queue from the main thread
	// would deadlock.
	ranNested := make(chan bool, 1)
	mainthread.Do(func() {
		mainthread.Do(func() {
			ranNested <- mainthread.IsMain()
		})
	})
	select {
	case got := <-ranNested:
		if !got {
			t.Fatal("nested Do body not on the main thread")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("nested Do deadlocked")
	}
}
