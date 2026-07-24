// Curated GOLDEN acceptance tests — hand-maintained, never regenerated.
//
// Tier B of the acceptance strategy: unlike the symbol-resolution gate (which
// proves a binding's symbol RESOLVES) and the layout gate (which proves a struct
// is the right SIZE), these call real framework functions and assert the result
// is CORRECT against a known-good value. They exercise the whole purego bridge
// end to end — argument marshalling, the call, and return unmarshalling — for a
// small, high-confidence set of deterministic APIs. Correctness here is what
// unit tests and shape tests structurally cannot prove.
//
//go:build darwin

package acceptance_test

import (
	"testing"
	"unsafe"

	cf "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/corefoundation"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
)

// CFStringGetLength returns the UTF-16 length; for ASCII that equals the byte
// count, giving a deterministic golden value.
func TestGolden_CFStringGetLength(t *testing.T) {
	runIsolated(t, "golden:CFStringGetLength", func(t *testing.T) {
		cases := []struct {
			in   string
			want int
		}{
			{"", 0},
			{"hello", 5},
			{"weave-acceptance", 16},
		}
		for _, c := range cases {
			s := cf.CFStringCreateWithCString(cf.CFAllocatorRef{}, c.in, int(cf.KCFStringEncodingUTF8))
			if s.IsNil() {
				t.Fatalf("CFStringCreateWithCString(%q) returned nil", c.in)
			}
			if got := cf.CFStringGetLength(s); got != c.want {
				t.Errorf("CFStringGetLength(%q) = %d, want %d", c.in, got, c.want)
			}
		}
	})
}

// A fresh mutable CFArray is empty; each append increments the count. Values are
// opaque here (nil callbacks), so any non-nil pointer is a valid element.
func TestGolden_CFArrayCountAppend(t *testing.T) {
	runIsolated(t, "golden:CFArray count/append", func(t *testing.T) {
		arr := cf.CFArrayCreateMutable(cf.CFAllocatorRef{}, 0, nil)
		if arr.IsNil() {
			t.Fatal("CFArrayCreateMutable returned nil")
		}
		if got := cf.CFArrayGetCount(arr.AsArray()); got != 0 {
			t.Errorf("empty CFArray count = %d, want 0", got)
		}
		var elem int
		cf.CFArrayAppendValue(arr, unsafe.Pointer(&elem))
		cf.CFArrayAppendValue(arr, unsafe.Pointer(&elem))
		if got := cf.CFArrayGetCount(arr.AsArray()); got != 2 {
			t.Errorf("CFArray count after 2 appends = %d, want 2", got)
		}
	})
}

// NSNumber must round-trip a non-negative int through the ObjC bridge unchanged.
func TestGolden_NSNumberIntRoundTrip(t *testing.T) {
	runIsolated(t, "golden:NSNumber int round-trip", func(t *testing.T) {
		for _, v := range []int{0, 1, 42, 1 << 20, 1<<31 - 1} {
			n := foundation.NumberWithInt(v)
			if n == nil {
				t.Fatalf("NumberWithInt(%d) returned nil", v)
			}
			if got := n.IntValue(); got != v {
				t.Errorf("NumberWithInt(%d).IntValue() = %d, want %d", v, got, v)
			}
		}
	})
}

// KNOWN BUG (tracked): a method whose C return is a sub-64-bit signed int (here
// -[NSNumber intValue], C int = 32-bit) is emitted as objc.Send[int] — reading 64
// bits of a 32-bit return, so a negative value comes back zero-extended, not
// sign-extended (NumberWithInt(-7).IntValue() == 4294967289, not -7). The fix is
// a return-width correction (objc.Send[int32] then widen to Go int), mirroring the
// struct-field StructFieldGoType correction. Unskip this test when that lands — it
// is the regression gate for the fix.
func TestKnownBug_NSNumberNegativeSignExtension(t *testing.T) {
	t.Skip("known bug: C-int method returns are not sign-extended (objc.Send[int] vs [int32]); see comment")
	runIsolated(t, "knownbug:NSNumber negative sign-extension", func(t *testing.T) {
		for _, v := range []int{-1, -7, -1000, -(1 << 20)} {
			if got := foundation.NumberWithInt(v).IntValue(); got != v {
				t.Errorf("NumberWithInt(%d).IntValue() = %d, want %d", v, got, v)
			}
		}
	})
}

// NSString round-trips a Go string through -stringWithUTF8String: and -String
// unchanged, and its length matches — exercising both directions of the bridge's
// string marshalling.
func TestGolden_NSStringRoundTrip(t *testing.T) {
	runIsolated(t, "golden:NSString round-trip", func(t *testing.T) {
		for _, in := range []string{"", "a", "weave", "the quick brown fox"} {
			s := foundation.StringWithUTF8String(in)
			if s == nil {
				t.Fatalf("StringWithUTF8String(%q) returned nil", in)
			}
			if got := s.String(); got != in {
				t.Errorf("StringWithUTF8String(%q).String() = %q, want %q", in, got, in)
			}
			if got := s.Length(); got != len(in) {
				t.Errorf("NSString(%q).Length() = %d, want %d", in, got, len(in))
			}
		}
	})
}
