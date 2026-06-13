package typemap

import "testing"

func TestIsCoreFoundationOpaqueRefKnown(t *testing.T) {
	known := []string{
		"CFStringRef", "CFArrayRef", "CFDictionaryRef", "CFURLRef",
		"CFMutableArrayRef", "CFMutableDictionaryRef", "CFMutableStringRef",
		"CFRunLoopRef", "CFAllocatorRef", "CFBundleRef", "CFDataRef",
		"CFErrorRef", "CFNumberRef", "CFBooleanRef", "CFUUIDRef",
	}
	for _, name := range known {
		if !IsCoreFoundationOpaqueRef(name) {
			t.Errorf("IsCoreFoundationOpaqueRef(%q) = false, want true", name)
		}
	}
}

func TestIsCoreFoundationOpaqueRefUnknown(t *testing.T) {
	for _, name := range []string{"NSString", "id", "CFIndex", "CFStringEncoding", "", "CFString"} {
		if IsCoreFoundationOpaqueRef(name) {
			t.Errorf("IsCoreFoundationOpaqueRef(%q) = true, want false", name)
		}
	}
}
