//go:build darwin

package idiofw

import "testing"

func TestScalarGoSizeAlign(t *testing.T) {
	cases := []struct {
		goType string
		size   int
		align  int
		ok     bool
	}{
		{"uint8", 1, 1, true},
		{"int8", 1, 1, true},
		{"bool", 1, 1, true},
		{"uint16", 2, 2, true},
		{"int16", 2, 2, true},
		{"uint32", 4, 4, true},
		{"float32", 4, 4, true},
		{"uint64", 8, 8, true},
		{"float64", 8, 8, true},
		{"int", 8, 8, true},
		{"uintptr", 8, 8, true},
		{"unsafe.Pointer", 0, 0, false},
		{"*Foo", 0, 0, false},
		{"SomeStruct", 0, 0, false},
	}
	for _, c := range cases {
		sz, al, ok := scalarGoSizeAlign(c.goType)
		if ok != c.ok || (ok && (sz != c.size || al != c.align)) {
			t.Errorf("scalarGoSizeAlign(%q) = %d,%d,%v; want %d,%d,%v",
				c.goType, sz, al, ok, c.size, c.align, c.ok)
		}
	}
}

func TestLayoutSafeFromGoTypes(t *testing.T) {
	cases := []struct {
		name    string
		packed  bool
		goTypes []string
		want    bool
	}{
		{"unpacked always safe", false, []string{"uint8", "uint32"}, true},
		{"unpacked non-scalar still safe", false, []string{"uint8", "SomeStruct"}, true},
		// IOUSBDeviceDescriptor shape: every uint16 lands on an even offset.
		{"packed contiguous device descriptor", true,
			[]string{"uint8", "uint8", "uint16", "uint8", "uint8", "uint8", "uint8",
				"uint16", "uint16", "uint16", "uint8", "uint8", "uint8", "uint8"}, true},
		// IOUSBConfigurationDescriptor shape: fixed 9-byte header, odd total size but
		// every field naturally aligned at its packed offset — safe for pointer access.
		{"packed odd-size but field-aligned", true,
			[]string{"uint8", "uint8", "uint16", "uint8", "uint8", "uint8", "uint8", "uint8"}, true},
		// A uint16 after a single uint8 would sit at packed offset 1 (misaligned).
		{"packed misaligned uint16", true, []string{"uint8", "uint16"}, false},
		{"packed non-scalar field", true, []string{"uint8", "SomeStruct"}, false},
	}
	for _, c := range cases {
		if got := layoutSafeFromGoTypes(c.packed, c.goTypes); got != c.want {
			t.Errorf("%s: layoutSafeFromGoTypes(%v, %v) = %v; want %v",
				c.name, c.packed, c.goTypes, got, c.want)
		}
	}
}
