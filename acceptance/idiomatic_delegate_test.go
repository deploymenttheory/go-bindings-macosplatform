// Curated acceptance test for the generated delegate support — hand-maintained,
// never regenerated. A plain Go value is installed as an NSCache delegate via
// the generated WithDelegate setter; forcing an eviction must drive the
// framework's cache:willEvictObject: callback through the runtime-registered
// shim class back into the Go method. This exercises the whole chain end to
// end: interface + upgrade-interface detection, objc.RegisterClass, the
// respondsToSelector: override, the associated-object lifetime, and argument
// conversion inside the IMP.
//
//go:build darwin

package acceptance_test

import (
	"testing"
	"time"

	idiofoundation "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/foundation"
	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/obj"

	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
)

// evictionRecorder implements the optional cache:willEvictObject: upgrade
// interface on top of the (empty) required CacheDelegate interface.
type evictionRecorder struct {
	evicted chan string
}

func (r *evictionRecorder) CacheWillEvictObject(_ obj.Object, object obj.Object) {
	select {
	case r.evicted <- object.Description():
	default:
	}
}

func TestCurated_Idiomatic_DelegateCallback(t *testing.T) {
	runIsolated(t, "curated:idiomatic NSCache delegate eviction callback", func(t *testing.T) {
		recorder := &evictionRecorder{evicted: make(chan string, 4)}

		cache := idiofoundation.NewCache().
			WithDelegate(recorder).
			WithCountLimit(1)

		first := obj.Wrap(purego.NSString("first-object"))
		second := obj.Wrap(purego.NSString("second-object"))
		keyA := obj.Wrap(purego.NSString("a"))
		keyB := obj.Wrap(purego.NSString("b"))

		// With a count limit of 1, inserting a second object forces the cache
		// to evict — which must arrive as a Go method call.
		cache.SetObjectForKey(first, keyA)
		cache.SetObjectForKey(second, keyB)

		select {
		case evicted := <-recorder.evicted:
			if evicted != "first-object" && evicted != "second-object" {
				t.Errorf("evicted object description = %q; want one of the inserted strings", evicted)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("cache:willEvictObject: never reached the Go delegate")
		}
	})
}
