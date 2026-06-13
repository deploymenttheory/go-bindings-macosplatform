package typemap

import "testing"

// --- EnumGoIntType ---

func TestEnumGoIntTypeSimpleName(t *testing.T) {
	m := &Mapper{EnumGoTypeIndex: map[string]string{"NSComparisonResult": "int64"}}
	if got := m.EnumGoIntType("NSComparisonResult"); got != "int64" {
		t.Errorf("EnumGoIntType(NSComparisonResult) = %q, want int64", got)
	}
}

func TestEnumGoIntTypeQualifiedName(t *testing.T) {
	m := &Mapper{EnumGoTypeIndex: map[string]string{"NSComparisonResult": "int64"}}
	if got := m.EnumGoIntType("foundation.NSComparisonResult"); got != "int64" {
		t.Errorf("EnumGoIntType(foundation.NSComparisonResult) = %q, want int64", got)
	}
}

func TestEnumGoIntTypeUnknown(t *testing.T) {
	m := &Mapper{EnumGoTypeIndex: map[string]string{}}
	if got := m.EnumGoIntType("UnknownEnum"); got != "" {
		t.Errorf("EnumGoIntType(UnknownEnum) = %q, want empty", got)
	}
}

// --- resolveEnumCType (all integer type branches) ---

func TestEnumCTypeAllIntegerBranches(t *testing.T) {
	cases := []struct{ goIntType, wantC string }{
		{"int8", "int8_t"},
		{"int16", "int16_t"},
		{"int32", "int32_t"},
		{"int64", "int64_t"},
		{"uint8", "uint8_t"},
		{"uint16", "uint16_t"},
		{"uint32", "uint32_t"},
		{"uint64", "uint64_t"},
	}
	for _, c := range cases {
		m := &Mapper{EnumGoTypeIndex: map[string]string{"MyEnum": c.goIntType}}
		if got := m.resolveEnumCType("MyEnum"); got != c.wantC {
			t.Errorf("resolveEnumCType for goIntType=%q: got %q, want %q", c.goIntType, got, c.wantC)
		}
	}
}

func TestEnumCTypeQualifiedName(t *testing.T) {
	m := &Mapper{EnumGoTypeIndex: map[string]string{"NSComparisonResult": "int64"}}
	if got := m.resolveEnumCType("foundation.NSComparisonResult"); got != "int64_t" {
		t.Errorf("resolveEnumCType(qualified) = %q, want int64_t", got)
	}
}

func TestEnumCTypeUnknown(t *testing.T) {
	m := &Mapper{EnumGoTypeIndex: map[string]string{}}
	if got := m.resolveEnumCType("UnknownEnum"); got != "" {
		t.Errorf("resolveEnumCType(unknown) = %q, want empty", got)
	}
}

// --- CType via enum path ---

func TestCTypeNamedEnum(t *testing.T) {
	m := &Mapper{
		EnumIndex:       map[string]string{"NSComparisonResult": "Foundation"},
		EnumGoTypeIndex: map[string]string{"NSComparisonResult": "int64"},
	}
	ctx := m.BaseContext("Foundation", map[string]bool{})
	if got := m.CType("NSComparisonResult", ctx, nil); got != "int64_t" {
		t.Errorf("CType(NSComparisonResult) = %q, want int64_t", got)
	}
}

// --- resolveGoType: typedef chain ---

func TestResolveGoTypeTypedefChain(t *testing.T) {
	m := newTestMapper()
	m.TypedefIndex = map[string]string{"NSMyBool": "BOOL"}
	// NSMyBool is a typedef for BOOL; BOOL resolves to bool.
	ctx := newTestCtx(m, "Foundation")
	got := m.GoType("NSMyBool", ctx, nil)
	if got != "bool" {
		t.Errorf("GoType(typedef NSMyBool→BOOL) = %q, want bool", got)
	}
}

func TestResolveGoTypeTypedefToUnsafeDoesntShortCircuit(t *testing.T) {
	m := newTestMapper()
	m.TypedefIndex = map[string]string{"OpaqueHandle": "id"} // id → cgo.Object
	// A typedef that resolves to id propagates the cgo.Object result upward.
	ctx := newTestCtx(m, "Foundation")
	got := m.GoType("OpaqueHandle", ctx, nil)
	// id → cgo.Object, so the typedef resolves to cgo.Object.
	if got != "cgo.Object" {
		t.Errorf("GoType(typedef→id) = %q, want cgo.Object", got)
	}
}

// --- resolveGoType: C fixed-size array ---

func TestResolveGoTypeFixedSizeArray(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	// "unsigned char [6]" → "[6]uint8"
	got := m.GoType("unsigned char [6]", ctx, nil)
	if got != "[6]uint8" {
		t.Errorf("GoType(unsigned char [6]) = %q, want [6]uint8", got)
	}
}

func TestResolveGoTypeFixedSizeArrayInt32(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	got := m.GoType("int32_t [4]", ctx, nil)
	if got != "[4]int32" {
		t.Errorf("GoType(int32_t [4]) = %q, want [4]int32", got)
	}
}

// --- resolveGoType: C-style lowercase enum via EnumIndex direct lookup ---

func TestResolveGoTypeLowercaseEnum(t *testing.T) {
	m := newTestMapper()
	m.EnumIndex = map[string]string{"interface_event_t": "vmnet"}
	ctx := newTestCtx(m, "vmnet")
	got := m.GoType("interface_event_t", ctx, nil)
	// capitaliseFirst("interface_event_t") = "Interface_event_t"; same-fw → unqualified
	if got != "Interface_event_t" {
		t.Errorf("GoType(interface_event_t lowercase enum) = %q, want Interface_event_t", got)
	}
}

// --- resolveGoType: "struct XYZ" bare form ---

func TestResolveGoTypeStructBareForm(t *testing.T) {
	m := newTestMapper()
	m.StructIndex = map[string]string{"ether_addr": "NetworkExtension"}
	ctx := newTestCtx(m, "NetworkExtension")
	got := m.GoType("struct ether_addr", ctx, nil)
	if got != "Ether_addr" {
		t.Errorf("GoType(struct ether_addr) = %q, want Ether_addr", got)
	}
}

// --- resolvePointerType: InClassMethod path ---

func TestGoPointerTypeInClassMethod(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	ctx.GenericParams = []string{"ObjectType"}
	ctx.IsClassMethod = true
	// NSArray<ObjectType> * in a class method: ObjectType in scope but InClassMethod=true
	// → qualifiedType with "[cgo.Object]" instead of "[T]"
	got := m.GoType("NSArray<ObjectType> *", ctx, nil)
	if got != "*NSArray[cgo.Object]" {
		t.Errorf("GoType(NSArray<ObjectType> *, InClassMethod) = %q, want *NSArray[cgo.Object]", got)
	}
}

// --- resolvePointerType: struct pointer in return position ---

func TestGoPointerTypeStructPointerReturn(t *testing.T) {
	m := newTestMapper()
	m.StructIndex = map[string]string{"CGSize": "CoreFoundation"}
	ctx := newTestCtx(m, "Foundation")
ctx.IsReturn = true
	// CGSize * in return position → unsafe.Pointer (emitter can't dereference C struct)
	got := m.GoType("CGSize *", ctx, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("GoType(CGSize *, IsReturn) = %q, want unsafe.Pointer", got)
	}
}

// --- resolvePointerType: C-style lowercase struct pointer ---

func TestGoPointerTypeLowercaseStructPointer(t *testing.T) {
	m := newTestMapper()
	m.StructIndex = map[string]string{"vmnet_network": "vmnet"}
	ctx := newTestCtx(m, "Foundation")
// "struct vmnet_network *" — ClassName returns "" (lowercase), so falls to bare strip path
	got := m.GoType("struct vmnet_network *", ctx, nil)
	if got != "*vmnet.Vmnet_network" {
		t.Errorf("GoType(struct vmnet_network *) = %q, want *vmnet.Vmnet_network", got)
	}
}

func TestGoPointerTypeLowercaseStructPointerReturn(t *testing.T) {
	m := newTestMapper()
	m.StructIndex = map[string]string{"vmnet_network": "vmnet"}
	ctx := newTestCtx(m, "Foundation")
ctx.IsReturn = true
	got := m.GoType("struct vmnet_network *", ctx, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("GoType(struct vmnet_network *, IsReturn) = %q, want unsafe.Pointer", got)
	}
}

// --- IsDoublePointer: generic with inner asterisk is not a double pointer ---

func TestIsDoublePointerGenericNotDouble(t *testing.T) {
	// NSArray<NSString *> * has * inside <> but only one * outside → not double pointer
	if IsDoublePointer("NSArray<NSString *> *") {
		t.Error("IsDoublePointer(NSArray<NSString *> *) = true, want false")
	}
}

func TestIsDoublePointerActualDouble(t *testing.T) {
	if !IsDoublePointer("SomeClass **") {
		t.Error("IsDoublePointer(SomeClass **) = false, want true")
	}
}

// --- NormaliseBlock ---

func TestNormaliseBlockStripsNullability(t *testing.T) {
	input := "void (^ _Nonnull)(NSString * _Nullable)"
	got := NormaliseBlock(input)
	// _Nonnull and _Nullable are replaced with a space, then whitespace is collapsed.
	// The space between ^ and ) remains as a single space (block syntax is not parsed).
	if got != "void (^ )(NSString * )" {
		t.Errorf("NormaliseBlock = %q, want \"void (^ )(NSString * )\"", got)
	}
}

func TestNormaliseBlockPreservesConst(t *testing.T) {
	// NormaliseBlock keeps const (unlike Normalise which strips it)
	input := "void (^)(const char *)"
	got := NormaliseBlock(input)
	if got != "void (^)(const char *)" {
		t.Errorf("NormaliseBlock(const) = %q, want \"void (^)(const char *)\"", got)
	}
}

// --- qualifiedProtocolType: blocked cycle drops constraint ---

func TestQualifiedProtocolTypeBlockedCycleDropsConstraint(t *testing.T) {
	m := newTestMapper()
	m.ProtocolIndex = map[string]string{"NSCopying": "Foundation"}
	ctx := newBlockedCtx(m, "AppKit", "Foundation")
	// NSCopying is blocked (Foundation→AppKit cycle) → protocolGoName returns "" → constraint dropped
	got := m.qualifiedProtocolType([]string{"NSCopying"}, ctx, nil)
	if got != "" {
		t.Errorf("qualifiedProtocolType(blocked) = %q, want empty", got)
	}
}

// --- resolveGoType: id<Protocol> in return position ---

func TestResolveGoTypeIDProtocolReturnPosition(t *testing.T) {
	m := newTestMapper()
	m.ProtocolIndex = map[string]string{"NSCopying": "Foundation"}
	ctx := newTestCtx(m, "Foundation")
	ctx.IsReturn = true
	// In return position with no proxy registered, id<Protocol> falls back to cgo.Object.
	got := m.GoType("id<NSCopying>", ctx, nil)
	if got != "cgo.Object" {
		t.Errorf("GoType(id<NSCopying>, IsReturn) = %q, want cgo.Object", got)
	}
}

// --- resolveGoType: lowercase struct in StructIndex direct lookup ---

func TestResolveGoTypeLowercaseStruct(t *testing.T) {
	m := newTestMapper()
	m.StructIndex = map[string]string{"vmnet_network": "vmnet"}
	ctx := newTestCtx(m, "vmnet")
	got := m.GoType("vmnet_network", ctx, nil)
	// qualifiedStructType with same-fw owner → bare name via naming.GoTypeName
	if got != "Vmnet_network" {
		t.Errorf("GoType(vmnet_network struct) = %q, want Vmnet_network", got)
	}
}
