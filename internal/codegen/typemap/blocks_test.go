package typemap

import "testing"

// --- IsNSError ---

func TestIsNSError(t *testing.T) {
	cases := []struct {
		qt   string
		want bool
	}{
		{"NSError *", true},
		{"NSError * _Nullable", true}, // normalises to "NSError *"
		{"NSError **", false},
		{"NSString *", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsNSError(c.qt); got != c.want {
			t.Errorf("IsNSError(%q) = %v, want %v", c.qt, got, c.want)
		}
	}
}

// --- GoBlockArgType ---

func TestGoBlockArgTypePrimitives(t *testing.T) {
	m := &Mapper{}
	cases := []struct{ qt, want string }{
		{"int64_t", "int64"},
		{"uint32_t", "uint32"},
		{"float", "float32"},
		{"double", "float64"},
		{"BOOL", "bool"},
		{"bool", "bool"},
		{"SEL", "string"},
		{"void", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := m.GoBlockArgType(c.qt); got != c.want {
			t.Errorf("GoBlockArgType(%q) = %q, want %q", c.qt, got, c.want)
		}
	}
}

func TestGoBlockArgTypeObjCObjects(t *testing.T) {
	m := &Mapper{}
	// All ObjC object types collapse to unsafe.Pointer in block trampoline signatures.
	for _, qt := range []string{"NSString *", "id", "Class", "void (^)()", "void *"} {
		if got := m.GoBlockArgType(qt); got != "unsafe.Pointer" {
			t.Errorf("GoBlockArgType(%q) = %q, want unsafe.Pointer", qt, got)
		}
	}
}

func TestGoBlockArgTypeCFTypedef(t *testing.T) {
	m := &Mapper{}
	// CF opaque typedefs (no pointer) collapse to unsafe.Pointer.
	if got := m.GoBlockArgType("CFStringRef"); got != "unsafe.Pointer" {
		t.Errorf("GoBlockArgType(CFStringRef) = %q, want unsafe.Pointer", got)
	}
}

func TestGoBlockArgTypeNamedEnum(t *testing.T) {
	m := &Mapper{EnumGoTypeIndex: map[string]string{"NSComparisonResult": "int64"}}
	if got := m.GoBlockArgType("NSComparisonResult"); got != "int64" {
		t.Errorf("GoBlockArgType(NSComparisonResult) = %q, want int64", got)
	}
}

func TestGoBlockArgTypeLowercaseEnum(t *testing.T) {
	m := &Mapper{EnumGoTypeIndex: map[string]string{"vmnet_return_t": "int64"}}
	if got := m.GoBlockArgType("vmnet_return_t"); got != "int64" {
		t.Errorf("GoBlockArgType(vmnet_return_t) = %q, want int64", got)
	}
}

func TestGoBlockArgTypeUnknownFallsBack(t *testing.T) {
	m := &Mapper{}
	if got := m.GoBlockArgType("SomeUnknownType"); got != "unsafe.Pointer" {
		t.Errorf("GoBlockArgType(SomeUnknownType) = %q, want unsafe.Pointer", got)
	}
}

// --- buildBlockGoType ---

func TestGoBlockTypeVoidNoArgs(t *testing.T) {
	m := &Mapper{}
	if got := m.buildBlockGoType("void (^)()", Context{}); got != "func()" {
		t.Errorf("buildBlockGoType(void (^)()) = %q, want func()", got)
	}
}

func TestGoBlockTypeWithPrimitiveArgs(t *testing.T) {
	m := &Mapper{}
	got := m.buildBlockGoType("void (^)(BOOL, int64_t)", Context{})
	if got != "func(bool, int64)" {
		t.Errorf("buildBlockGoType = %q, want func(bool, int64)", got)
	}
}

func TestGoBlockTypeObjectArgsBecomePtrs(t *testing.T) {
	m := &Mapper{}
	got := m.buildBlockGoType("void (^)(NSString *, NSError *)", Context{})
	if got != "func(unsafe.Pointer, unsafe.Pointer)" {
		t.Errorf("buildBlockGoType = %q, want func(unsafe.Pointer, unsafe.Pointer)", got)
	}
}

func TestGoBlockTypeBoolReturn(t *testing.T) {
	m := &Mapper{}
	got := m.buildBlockGoType("BOOL (^)(NSString *)", Context{})
	if got != "func(unsafe.Pointer) bool" {
		t.Errorf("buildBlockGoType = %q, want func(unsafe.Pointer) bool", got)
	}
}

func TestGoBlockTypeVoidArgs(t *testing.T) {
	m := &Mapper{}
	got := m.buildBlockGoType("id (^)(void)", Context{})
	if got != "func() unsafe.Pointer" {
		t.Errorf("buildBlockGoType = %q, want func() unsafe.Pointer", got)
	}
}

func TestGoBlockTypeMalformedReturnsEmpty(t *testing.T) {
	m := &Mapper{}
	if got := m.buildBlockGoType("not a block", Context{}); got != "" {
		t.Errorf("buildBlockGoType(malformed) = %q, want empty", got)
	}
}

// --- GoBlockUserFuncType ---

func TestGoBlockUserFuncTypeNSErrorArg(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	// NSError * should become Go error interface.
	got := m.GoBlockUserFuncType("void (^)(NSError *)", ctx, nil)
	if got != "func(error)" {
		t.Errorf("GoBlockUserFuncType(NSError * arg) = %q, want func(error)", got)
	}
}

func TestGoBlockUserFuncTypeKnownClass(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "AppKit")
	imports := make(ImportSet)
	// NSString * is a known class owned by Foundation → typed *foundation.NSString.
	got := m.GoBlockUserFuncType("void (^)(NSString *)", ctx, imports)
	if got != "func(*foundation.NSString)" {
		t.Errorf("GoBlockUserFuncType(NSString * arg) = %q, want func(*foundation.NSString)", got)
	}
	if imports["foundation"] == "" {
		t.Error("expected imports to contain foundation import")
	}
}

func TestGoBlockUserFuncTypeSameFrameworkClass(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	// NSString is owned by Foundation — same framework, no qualifier.
	got := m.GoBlockUserFuncType("void (^)(NSString *)", ctx, nil)
	if got != "func(*NSString)" {
		t.Errorf("GoBlockUserFuncType(NSString * same-fw) = %q, want func(*NSString)", got)
	}
}

func TestGoBlockUserFuncTypeUnknownClassFallsBack(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	// SomeUnknown is not in ClassNameIndex → falls back to GoBlockArgType → unsafe.Pointer.
	got := m.GoBlockUserFuncType("void (^)(SomeUnknown *)", ctx, nil)
	if got != "func(unsafe.Pointer)" {
		t.Errorf("GoBlockUserFuncType(unknown class) = %q, want func(unsafe.Pointer)", got)
	}
}

func TestGoBlockUserFuncTypePrimitiveArg(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	got := m.GoBlockUserFuncType("void (^)(BOOL, int64_t)", ctx, nil)
	if got != "func(bool, int64)" {
		t.Errorf("GoBlockUserFuncType(primitives) = %q, want func(bool, int64)", got)
	}
}

func TestGoBlockUserFuncTypeWithReturn(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	got := m.GoBlockUserFuncType("BOOL (^)(void)", ctx, nil)
	if got != "func() bool" {
		t.Errorf("GoBlockUserFuncType(BOOL return) = %q, want func() bool", got)
	}
}

func TestGoBlockUserFuncTypeMalformedReturnsEmpty(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	if got := m.GoBlockUserFuncType("not a block", ctx, nil); got != "" {
		t.Errorf("GoBlockUserFuncType(malformed) = %q, want empty", got)
	}
}

func TestGoBlockUserFuncTypeNoArgs(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	got := m.GoBlockUserFuncType("void (^)()", ctx, nil)
	if got != "func()" {
		t.Errorf("GoBlockUserFuncType(void (^)()) = %q, want func()", got)
	}
}
