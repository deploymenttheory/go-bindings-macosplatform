// Curated acceptance tests for the idiomatic layer's deterministic lifecycle —
// hand-maintained, never regenerated. Release must be idempotent, safe under
// concurrent callers, leave the wrapper in the documented nil-messaging state
// (methods become no-ops returning zero values), and must not lead to a second
// release when the garbage collector later runs the wrapper's finalizer.
//
//go:build darwin

package acceptance_test

import (
	"runtime"
	"sync"
	"testing"
	"time"

	idiofoundation "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/obj"

	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
)

func TestCurated_Idiomatic_ReleaseLifecycle(t *testing.T) {
	runIsolated(t, "curated:idiomatic Release lifecycle", func(t *testing.T) {
		array := idiofoundation.NewMutableArray()
		array.AddObject(obj.Wrap(purego.NSString("element")))
		if got := array.Count(); got != 1 {
			t.Fatalf("Count before Release = %d; want 1", got)
		}

		array.Release()
		array.Release() // idempotent: the second call must be a no-op

		// After Release the stored pointer is zero, so the send goes to nil and
		// Objective-C returns zero — the documented nil-messaging behavior.
		if got := array.Count(); got != 0 {
			t.Errorf("Count after Release = %d; want 0 (nil-messaging no-op)", got)
		}

		// The finalizer reads through the same atomic Release cleared, so
		// collecting the released wrapper must not release a second time (a
		// double release would crash the process; runIsolated would report it).
		runtime.GC()
		runtime.GC()
		time.Sleep(50 * time.Millisecond)
	})
}

func TestCurated_Idiomatic_ReleaseConcurrent(t *testing.T) {
	runIsolated(t, "curated:idiomatic Release concurrent", func(t *testing.T) {
		for range 100 {
			array := idiofoundation.NewMutableArray()
			var wg sync.WaitGroup
			for range 8 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					array.Release()
				}()
			}
			wg.Wait()
		}
	})
}

func TestCurated_Idiomatic_ReleaseThroughObjInterface(t *testing.T) {
	runIsolated(t, "curated:idiomatic Release via obj.Object", func(t *testing.T) {
		o := obj.Wrap(purego.NSString("interface release"))
		if o.Description() == "" {
			t.Fatal("Description before Release must be non-empty")
		}
		o.Release()
		o.Release()
		if got := o.Description(); got != "" {
			t.Errorf("Description after Release = %q; want \"\"", got)
		}
	})
}
