//go:build darwin

package idiofw

import "testing"

func TestImmutableConversion(t *testing.T) {
	handles := map[string]bool{
		"CFArrayRef":                   true,
		"CFMutableArrayRef":            true,
		"CFStringRef":                  true,
		"CFMutableStringRef":           true,
		"CFMutableWidgetRef":           true, // mutable with no owned immutable counterpart
		"CFAttributedStringRef":        true,
		"CFMutableAttributedStringRef": true,
	}
	cases := []struct {
		name       string
		wantType   string
		wantMethod string
	}{
		{"CFMutableArrayRef", "CFArrayRef", "AsArray"},
		{"CFMutableStringRef", "CFStringRef", "AsString"},
		{"CFMutableAttributedStringRef", "CFAttributedStringRef", "AsAttributedString"},
		{"CFMutableWidgetRef", "", ""}, // no CFWidgetRef in the set
		{"CFArrayRef", "", ""},         // not mutable
		{"CFTypeRef", "", ""},          // not mutable
		{"CFMutableRef", "", ""},       // empty core
		{"NotAHandle", "", ""},
	}
	for _, c := range cases {
		gotType, gotMethod := immutableConversion(c.name, handles)
		if gotType != c.wantType || gotMethod != c.wantMethod {
			t.Errorf("immutableConversion(%q) = (%q, %q), want (%q, %q)",
				c.name, gotType, gotMethod, c.wantType, c.wantMethod)
		}
	}
}
