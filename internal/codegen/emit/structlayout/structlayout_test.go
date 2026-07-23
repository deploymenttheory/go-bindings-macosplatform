package structlayout

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
		sz, al, ok := ScalarGoSizeAlign(c.goType)
		if ok != c.ok || (ok && (sz != c.size || al != c.align)) {
			t.Errorf("ScalarGoSizeAlign(%q) = %d,%d,%v; want %d,%d,%v",
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
		if got := LayoutSafeFromGoTypes(c.packed, c.goTypes); got != c.want {
			t.Errorf("%s: LayoutSafeFromGoTypes(%v, %v) = %v; want %v",
				c.name, c.packed, c.goTypes, got, c.want)
		}
	}
}

func TestGoStructLayout(t *testing.T) {
	cases := []struct {
		name    string
		packed  bool
		goTypes []string
		offsets []int
		size    int
		ok      bool
	}{
		{"scalars natural", false, []string{"uint8", "uint16", "uint8"}, []int{0, 2, 4}, 6, true},
		{"int32 pair", false, []string{"int32", "int32"}, []int{0, 4}, 8, true},
		{"mixed align pads", false, []string{"uint8", "uint32"}, []int{0, 4}, 8, true},
		{"packed tight", true, []string{"uint8", "uint16", "uint8"}, []int{0, 1, 3}, 4, true},
		{"byte array field", false, []string{"uint8", "[3]uint8", "uint16"}, []int{0, 1, 4}, 6, true},
		{"unknown type", false, []string{"SomeStruct"}, nil, 0, false},
	}
	for _, c := range cases {
		offsets, size, ok := GoStructLayout(c.goTypes, c.packed)
		if ok != c.ok || (ok && (size != c.size || !intsEqualLayout(offsets, c.offsets))) {
			t.Errorf("%s: GoStructLayout = %v,%d,%v; want %v,%d,%v",
				c.name, offsets, size, ok, c.offsets, c.size, c.ok)
		}
	}
}

func TestStructFieldGoType(t *testing.T) {
	cases := []struct{ objc, mapped, want string }{
		{"int", "int", "int32"},
		{"unsigned int", "uint", "uint32"},
		{"int[4]", "[4]int", "[4]int32"},
		{"const int", "int", "int32"},
		{"long", "int", "int"},      // 8 bytes = Go int; unchanged
		{"NSInteger", "int", "int"}, // 8 bytes; unchanged
		{"short", "int16", "int16"},
		{"uint32_t", "uint32", "uint32"},
	}
	for _, c := range cases {
		if got := StructFieldGoType(c.objc, c.mapped); got != c.want {
			t.Errorf("StructFieldGoType(%q, %q) = %q; want %q", c.objc, c.mapped, got, c.want)
		}
	}
}

func TestLayoutMatchesAuthoritative(t *testing.T) {
	cases := []struct {
		name         string
		size         int
		packed       bool
		fieldOffsets []int
		goTypes      []string
		want         bool
	}{
		{"no authoritative layout", 0, false, nil, []string{"uint8", "uint32"}, true},
		{"matching natural layout", 8, false, []int{0, 4}, []string{"int32", "int32"}, true},
		{"mismatched offset", 8, false, []int{0, 2}, []string{"int32", "int32"}, false},
		{"mismatched size", 12, false, []int{0, 4}, []string{"int32", "int32"}, false},
		// A non-scalar field the layout computer cannot size — skip (fixpoint gates it).
		{"unsizable field skips check", 16, false, []int{0}, []string{"SomeStruct"}, true},
	}
	for _, c := range cases {
		if got := LayoutMatchesAuthoritative(c.size, c.packed, c.fieldOffsets, c.goTypes); got != c.want {
			t.Errorf("%s: LayoutMatchesAuthoritative = %v; want %v", c.name, got, c.want)
		}
	}
}

func intsEqualLayout(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
