package typemap

import "testing"

func TestGoABIType(t *testing.T) {
	m := &Mapper{TypedefIndex: map[string]string{
		"AudioChannelLabel": "UInt32",
		"UInt32":            "unsigned int",
		"SInt32":            "int",
		"MyLong":            "long",
	}}
	cases := []struct{ qt, goType, want string }{
		{"unsigned int", "uint", "uint32"},      // direct keyword
		{"UInt32", "uint", "uint32"},            // one typedef hop (MacTypes)
		{"AudioChannelLabel", "uint", "uint32"}, // two hops
		{"SInt32", "int", "int32"},              // signed typedef → int
		{"MyLong", "int", "int"},                // resolves to long (64-bit) → keep
		{"NSInteger", "int", "int"},             // 64-bit primitive → keep
		{"UInt32[4]", "[4]uint", "[4]uint32"},   // array of integer typedef
		{"unsigned int", "int32", "int32"},      // already narrowed → unchanged
		{"id", "objc.ID", "objc.ID"},            // non-int → unchanged
	}
	for _, c := range cases {
		if got := m.GoABIType(c.qt, c.goType); got != c.want {
			t.Errorf("GoABIType(%q, %q) = %q; want %q", c.qt, c.goType, got, c.want)
		}
	}
}
