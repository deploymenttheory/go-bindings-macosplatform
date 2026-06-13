// Curated acceptance tests — hand-maintained, never regenerated.
// These validate end-to-end purego bridge correctness for the most critical
// Foundation binding patterns: object creation, string conversion, numeric
// invariants, and round-trips. They serve as the stable regression anchor:
// if a bridge is broken in a fundamental way these tests catch it regardless
// of which random symbols the generator sampled.
//
//go:build darwin

package acceptance_test

import (
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
)

// --- NSProcessInfo -----------------------------------------------------------

func TestCurated_NSProcessInfo_NonNil(t *testing.T) {
	runIsolated(t, "curated:NSProcessInfo.processInfo non-nil", func(t *testing.T) {
		pi := foundation.NSProcessInfoProcessInfo()
		if pi == nil {
			t.Fatal("NSProcessInfo.processInfo returned nil")
		}
	})
}

func TestCurated_NSProcessInfo_ProcessIdentifierPositive(t *testing.T) {
	runIsolated(t, "curated:NSProcessInfo.processIdentifier > 0", func(t *testing.T) {
		pid := foundation.NSProcessInfoProcessInfo().ProcessIdentifier()
		if pid <= 0 {
			t.Errorf("processIdentifier = %d; want > 0", pid)
		}
	})
}

func TestCurated_NSProcessInfo_ProcessorCountPositive(t *testing.T) {
	runIsolated(t, "curated:NSProcessInfo.processorCount > 0", func(t *testing.T) {
		n := foundation.NSProcessInfoProcessInfo().ProcessorCount()
		if n == 0 {
			t.Error("processorCount returned 0")
		}
	})
}

// --- NSString round-trip -----------------------------------------------------

func TestCurated_NSString_RoundTrip(t *testing.T) {
	runIsolated(t, "curated:NSString Go→NSString→Go round-trip", func(t *testing.T) {
		const input = "weave acceptance test"
		nsStr := foundation.NSStringStringWithUTF8String(input)
		if nsStr == nil {
			t.Fatal("NSString.stringWithUTF8String returned nil")
		}
		got := purego.GoString(nsStr.Ptr())
		if got != input {
			t.Errorf("round-trip: got %q, want %q", got, input)
		}
	})
}

func TestCurated_NSString_LengthMatchesInput(t *testing.T) {
	runIsolated(t, "curated:NSString.length matches input", func(t *testing.T) {
		const input = "hello"
		nsStr := foundation.NSStringStringWithUTF8String(input)
		if nsStr == nil {
			t.Fatal("NSStringStringWithUTF8String returned nil")
		}
		if got := nsStr.Length(); got != uint(len(input)) {
			t.Errorf("NSString.length = %d, want %d", got, len(input))
		}
	})
}

func TestCurated_NSString_EmptyString(t *testing.T) {
	runIsolated(t, "curated:NSString.string returns empty NSString", func(t *testing.T) {
		empty := foundation.NSStringString()
		if empty == nil {
			t.Fatal("NSString.string returned nil")
		}
		if empty.Length() != 0 {
			t.Errorf("empty NSString.length = %d, want 0", empty.Length())
		}
	})
}

// --- NSArray -----------------------------------------------------------------

func TestCurated_NSArray_EmptyArray(t *testing.T) {
	runIsolated(t, "curated:NSArray.array returns empty array", func(t *testing.T) {
		arr := foundation.NSArrayArray()
		if arr == nil {
			t.Fatal("NSArray.array returned nil")
		}
		if arr.Count() != 0 {
			t.Errorf("empty NSArray.count = %d, want 0", arr.Count())
		}
	})
}

// --- NSURL -------------------------------------------------------------------

func TestCurated_NSURL_FileURL_NonNil(t *testing.T) {
	runIsolated(t, "curated:NSURL.fileURLWithPath:/tmp non-nil", func(t *testing.T) {
		path := foundation.NSStringStringWithUTF8String("/tmp")
		url := foundation.NSURLFileURLWithPath(path)
		if url == nil {
			t.Fatal("NSURL.fileURLWithPath: returned nil")
		}
	})
}

func TestCurated_NSURL_PathRoundTrip(t *testing.T) {
	runIsolated(t, "curated:NSURL.fileURLWithPath: path round-trip", func(t *testing.T) {
		const dir = "/tmp"
		path := foundation.NSStringStringWithUTF8String(dir)
		url := foundation.NSURLFileURLWithPath(path)
		if url == nil {
			t.Fatal("NSURL.fileURLWithPath: returned nil")
		}
		urlPath := url.Path()
		if urlPath == nil {
			t.Fatal("NSURL.path returned nil")
		}
		got := purego.GoString(urlPath.Ptr())
		if got != dir {
			t.Errorf("NSURL.path round-trip: got %q, want %q", got, dir)
		}
	})
}

// --- NSBundle ----------------------------------------------------------------

func TestCurated_NSBundle_MainBundle_NonNil(t *testing.T) {
	runIsolated(t, "curated:NSBundle.mainBundle non-nil", func(t *testing.T) {
		bundle := foundation.NSBundleMainBundle()
		if bundle == nil {
			t.Skip("NSBundle.mainBundle returned nil — no main bundle in test environment")
		}
	})
}

func TestCurated_NSBundle_MainBundle_BundlePathNonEmpty(t *testing.T) {
	runIsolated(t, "curated:NSBundle.mainBundle.bundlePath non-empty", func(t *testing.T) {
		bundle := foundation.NSBundleMainBundle()
		if bundle == nil {
			t.Skip("NSBundle.mainBundle returned nil — no main bundle in test environment")
		}
		pathNS := bundle.BundlePath()
		if pathNS == nil {
			t.Fatal("NSBundle.bundlePath returned nil")
		}
		path := purego.GoString(pathNS.Ptr())
		if path == "" {
			t.Error("NSBundle.bundlePath returned empty string")
		}
	})
}
