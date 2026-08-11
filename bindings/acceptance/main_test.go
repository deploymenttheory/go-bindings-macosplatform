//go:build darwin

package acceptance_test

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/tools/grandcentraldispatch/mainthread"
)

func init() {
	// Pin the main goroutine to the OS main thread for the lifetime of this
	// test binary. TestMain pumps dispatch_get_main_queue() — which must run
	// on the OS main thread — so we lock early to prevent the Go scheduler
	// from ever migrating this goroutine to a different OS thread.
	runtime.LockOSThread()
}

// TestMain pumps the main run loop on the OS main thread while all tests run in
// a background goroutine. This is required because RunOnMainThread uses
// dispatch_sync_f(dispatch_get_main_queue()), which requires the main OS thread
// to be servicing its dispatch queue. Without this, any test that calls
// RunOnMainThread would deadlock: the test goroutine waits for the main queue to
// drain, while the main goroutine waits for the test goroutine to finish.
func TestMain(m *testing.M) {
	done := make(chan int, 1)
	go func() {
		done <- m.Run()
	}()
	// Deadline prevents the pump loop from spinning forever if a test deadlocks
	// after m.Run() returns (e.g. a goroutine leaked from a CGo callback).
	deadline := time.After(10 * time.Minute)
	for {
		select {
		case code := <-done:
			os.Exit(code)
		case <-deadline:
			os.Exit(2)
		default:
			// Drain the main dispatch queue (what mainthread.Do posts to) on the
			// OS main thread for up to 50ms per iteration.
			mainthread.PumpMainRunLoop(0.05)
		}
	}
}
