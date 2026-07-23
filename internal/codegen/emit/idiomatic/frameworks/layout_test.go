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
		// Fixed-size arrays: N × elem size, elem alignment.
		{"[16]uint8", 16, 1, true},
		{"[8]uint16", 16, 2, true},
		{"[4]uint32", 16, 4, true},
		{"[2][3]uint8", 6, 1, true}, // multi-dimensional
		{"[]uint8", 0, 0, false},    // slice, not a fixed array
		{"[3]*Foo", 0, 0, false},    // array of non-scalar
		{"[0]uint8", 0, 1, true},    // zero-length fixed array is 0 bytes
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
		// A fixed byte array keeps 1-byte alignment, so following fields stay aligned.
		{"packed array field aligned", true, []string{"uint8", "[3]uint8", "uint16"}, true},
		// A uint16 array forces 2-byte alignment; after a lone uint8 it is misaligned.
		{"packed uint16 array misaligned", true, []string{"uint8", "[2]uint16"}, false},
	}
	for _, c := range cases {
		if got := layoutSafeFromGoTypes(c.packed, c.goTypes); got != c.want {
			t.Errorf("%s: layoutSafeFromGoTypes(%v, %v) = %v; want %v",
				c.name, c.packed, c.goTypes, got, c.want)
		}
	}
}
