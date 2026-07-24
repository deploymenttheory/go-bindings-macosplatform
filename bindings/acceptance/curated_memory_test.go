// Curated MEMORY / lifetime acceptance tests — hand-maintained, never regenerated.
//
// Tier C of the acceptance strategy: assert the reference-counting contract the
// idiomatic layer promises (the CF Create Rule: a Create/Copy result is +1 and
// must be ADOPTED, not re-retained) and that churning objects through the
// finalizer never crashes. A retain-count regression here means a leak (double
// retain) or a use-after-free (missing retain) — exactly the class of bug the
// obj.Wrap/Adopt/WrapUnmanaged distinction exists to prevent.
//
//go:build darwin

package acceptance_test

import (
	"runtime"
	"testing"

	cf "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/corefoundation"
)

// A CF Create result must have retain count exactly 1: the binding ADOPTS the +1
// the Create Rule guarantees. If it re-retained (obj.Wrap) the count would be 2 —
// a leak. This is the runtime proof of the Phase-4 Adopt fix.
func TestMem_CFCreateReturnsPlusOne(t *testing.T) {
	runIsolated(t, "mem:CF Create returns +1 (adopt, no double-retain)", func(t *testing.T) {
		arr := cf.CFArrayCreateMutable(cf.CFAllocatorRef{}, 0, nil)
		if arr.IsNil() {
			t.Fatal("CFArrayCreateMutable returned nil")
		}
		if got := cf.CFGetRetainCount(arr); got != 1 {
			t.Errorf("CFArrayCreateMutable retain count = %d, want 1 (a +1 Create result must be adopted, not re-retained)", got)
		}
	})
}

// Churning many adopted objects through the garbage collector's finalizer must
// not crash or double-release. Each iteration drops its only reference; the
// finalizer sends the matching -release. A double retain (leak) or missing retain
// (over-release → crash) would surface here under GC pressure.
func TestMem_FinalizerChurnNoCrash(t *testing.T) {
	runIsolated(t, "mem:finalizer churn under GC", func(t *testing.T) {
		for i := 0; i < 5000; i++ {
			arr := cf.CFArrayCreateMutable(cf.CFAllocatorRef{}, 0, nil)
			if arr.IsNil() {
				t.Fatalf("CFArrayCreateMutable returned nil at i=%d", i)
			}
			_ = cf.CFGetRetainCount(arr) // touch it, then drop the reference
		}
		runtime.GC()
		runtime.GC()
		// Reaching here without a crash is the assertion.
	})
}
