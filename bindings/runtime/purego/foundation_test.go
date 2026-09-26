//go:build darwin

package purego

import "testing"

// This package's test binary loads no framework at launch, so these run in the
// state a consumer's process starts in: libobjc only, Foundation not loaded.

func TestNSStringRoundTripWithoutFoundationLoaded(t *testing.T) {
	const want = "Grüße aus /tmp/a b"
	if got := GoString(NSString(want)); got != want {
		t.Fatalf("GoString(NSString(%q)) = %q", want, got)
	}
}

func TestFoundationClassLoadsFoundation(t *testing.T) {
	for _, name := range []string{"NSURL", "NSThread", "NSXPCConnection"} {
		if FoundationClass(name) == 0 {
			t.Errorf("FoundationClass(%q) = nil", name)
		}
	}
}
