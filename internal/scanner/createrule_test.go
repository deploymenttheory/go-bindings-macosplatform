package scanner

import "testing"

func TestFollowsCreateRule(t *testing.T) {
	yes := []string{"CFArrayCreateMutable", "CFStringCreateWithCString", "CFCopyDescription",
		"SomethingCreate", "SecKeyCopyPublicKey"}
	no := []string{"CFRunLoopGetCurrent", "CFArrayGetCount", "GetCopyright", "MyCreator",
		"CFBundleGetMainBundle", "NoOwnershipHere"}
	for _, n := range yes {
		if !followsCreateRule(n) {
			t.Errorf("followsCreateRule(%q) = false; want true", n)
		}
	}
	for _, n := range no {
		if followsCreateRule(n) {
			t.Errorf("followsCreateRule(%q) = true; want false", n)
		}
	}
}
