package typemap

import (
	"testing"
)

func newTestMapper() *Mapper {
	return &Mapper{
		GenericClasses: map[string]bool{
			"NSArray":      true,
			"NSDictionary": true,
		},
		OwnerIndex: map[string]string{
			"NSString":     "Foundation",
			"NSArray":      "Foundation",
			"NSDictionary": "Foundation",
			"NSObject":     "Foundation",
			"NSWindow":     "AppKit",
		},
		ModulePrefix: "github.com/deploymenttheory/go-bindings-macosplatform/frameworks",
	}
}

func newTestCtx(m *Mapper, framework string) Context {
	return m.BaseContext(framework, map[string]bool{
		"NSString":     true,
		"NSArray":      true,
		"NSDictionary": true,
		"NSObject":     true,
		"NSWindow":     true,
	})
}

// TestGoTypePrimitives verifies that all well-known C/ObjC primitives map to
// the expected Go type strings.
func TestGoTypePrimitives(t *testing.T) {
	m := newTestMapper()

	cases := []struct {
		qt, want string
	}{
		// Fixed-width integer types.
		{"int64_t", "int64"},
		{"uint64_t", "uint64"},
		// Floating-point types.
		{"float", "float32"},
		{"double", "float64"},
		// Apple platform integer/float aliases.
		{"NSInteger", "int64"},
		{"NSUInteger", "uint64"},
		{"CGFloat", "float64"},
		// Boolean types.
		{"BOOL", "bool"},
		{"bool", "bool"},
		{"_Bool", "bool"},
		// char * and const char * map to string.
		{"char *", "string"},
		{"const char *", "string"},
		// void * is unsafe.Pointer.
		{"void *", "unsafe.Pointer"},
		// SEL is represented as a string (bridge converts).
		{"SEL", "string"},
		// Class meta-object → unsafe.Pointer.
		{"Class", "unsafe.Pointer"},
		// id → objc.Object (the common ObjC object interface).
		{"id", "objc.Object"},
	}

	for _, c := range cases {
		ctx := newTestCtx(m, "Foundation")
		got := m.GoType(c.qt, ctx, nil)
		if got != c.want {
			t.Errorf("GoType(%q) = %q, want %q", c.qt, got, c.want)
		}
	}
}

// TestGoTypePrimitivesUnknownClass verifies that an ObjC pointer to an unknown
// class (not in ClassNameIndex) falls back to unsafe.Pointer.
func TestGoTypePrimitivesUnknownClass(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")

	// "NSUnknown" is not registered in ClassNameIndex.
	got := m.GoType("NSUnknown *", ctx, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("GoType(%q) = %q, want unsafe.Pointer", "NSUnknown *", got)
	}
}

// TestGoTypeSameFramework verifies that a class pointer from the same framework
// is returned without a package qualifier.
func TestGoTypeSameFramework(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")

	got := m.GoType("NSString *", ctx, nil)
	if got != "*NSString" {
		t.Errorf("GoType(NSString *, Foundation) = %q, want *NSString", got)
	}
}

// TestGoTypeCrossFramework verifies that a class pointer from a different
// framework is qualified with a package alias and that imports is updated.
func TestGoTypeCrossFramework(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "AppKit")
	imports := make(ImportSet)

	got := m.GoType("NSString *", ctx, imports)
	want := "*foundation.NSString"
	if got != want {
		t.Errorf("GoType(NSString *, AppKit) = %q, want %q", got, want)
	}
	if imports["foundation"] == "" {
		t.Error("expected imports to contain a foundation import path")
	}
}

// TestGoTypeCrossFrameworkGeneric verifies that a generic class from a different
// framework is qualified and receives the runtime.Object type parameter when the
// type argument is not an in-scope generic param.
func TestGoTypeCrossFrameworkGeneric(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "AppKit")

	got := m.GoType("NSArray *", ctx, nil)
	want := "*foundation.NSArray[objc.Object]"
	if got != want {
		t.Errorf("GoType(NSArray *, AppKit) = %q, want %q", got, want)
	}
}

// TestGoTypeBlockedImport verifies that when a cross-framework reference would
// create a blocked import, the type falls back to unsafe.Pointer.
func TestGoTypeBlockedImport(t *testing.T) {
	m := newTestMapper()
	m.BlockedImports = map[string]map[string]bool{
		"AppKit": {"Foundation": true},
	}
	ctx := newTestCtx(m, "AppKit")

	// NSString lives in Foundation; from AppKit that import is blocked.
	got := m.GoType("NSString *", ctx, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("GoType(NSString *, AppKit [blocked]) = %q, want unsafe.Pointer", got)
	}
}

// TestCTypeMapping verifies that CType produces correct C type strings for
// various Go-level types.
func TestCTypeMapping(t *testing.T) {
	m := newTestMapper()

	cases := []struct {
		qt, want string
	}{
		// void / empty → "void".
		{"void", "void"},
		{"", "void"},
		// Boolean.
		{"bool", "bool"},
		// Fixed-width integer types.
		{"int8_t", "int8_t"},
		{"uint64_t", "uint64_t"},
		// Floating-point.
		{"float", "float"},
		{"double", "double"},
		// string types.
		{"char *", "const char *"},
		{"const char *", "const char *"},
		// ObjC object pointer → id; genuinely opaque → void *.
		{"NSString *", "id"},
		{"void *", "void *"},
	}

	for _, c := range cases {
		ctx := newTestCtx(m, "Foundation")
		got := m.CType(c.qt, ctx, nil)
		if got != c.want {
			t.Errorf("CType(%q) = %q, want %q", c.qt, got, c.want)
		}
	}
}

// TestGoTypeInstancetype verifies that "instancetype" resolves to a pointer to
// the current class when ctx.ClassName is set, and to unsafe.Pointer otherwise.
func TestGoTypeInstancetype(t *testing.T) {
	m := newTestMapper()

	withClass := newTestCtx(m, "Foundation")
	withClass.ClassName = "NSArray"
	got := m.GoType("instancetype", withClass, nil)
	if got != "*NSArray" {
		t.Errorf("GoType(instancetype, ClassName=NSArray) = %q, want *NSArray", got)
	}

	withoutClass := newTestCtx(m, "Foundation")
	got = m.GoType("instancetype", withoutClass, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("GoType(instancetype, ClassName='') = %q, want unsafe.Pointer", got)
	}
}

// TestGoTypeVoidEmpty verifies that "void" and the empty string both produce an
// empty Go type string (no return value).
func TestGoTypeVoidEmpty(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")

	if got := m.GoType("void", ctx, nil); got != "" {
		t.Errorf("GoType(%q) = %q, want empty string", "void", got)
	}
	if got := m.GoType("", ctx, nil); got != "" {
		t.Errorf("GoType(%q) = %q, want empty string", "", got)
	}
}

// TestGoReturnTypeVoid verifies that GoReturnType returns an empty string for void.
func TestGoReturnTypeVoid(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")

	if got := m.GoReturnType("void", ctx, nil); got != "" {
		t.Errorf("GoReturnType(%q) = %q, want empty string", "void", got)
	}
}

// TestGoTypeNSErrorDoublePointer verifies that NSError ** is mapped to
// unsafe.Pointer (it is handled at the bridge layer, not as a real Go type).
func TestGoTypeNSErrorDoublePointer(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")

	got := m.GoType("NSError **", ctx, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("GoType(%q) = %q, want unsafe.Pointer", "NSError **", got)
	}
}

// TestNew verifies that New() returns a non-nil Mapper and that it can be used
// for basic type resolution without panicking.
func TestNew(t *testing.T) {
	m := New()
	if m == nil {
		t.Fatal("New() returned nil")
	}
	ctx := Context{Framework: "Foundation", ClassNameIndex: map[string]bool{}}
	// A primitive type should resolve without any mapper state.
	got := m.GoType("int64_t", ctx, nil)
	if got != "int64" {
		t.Errorf("New mapper GoType(int64_t) = %q, want int64", got)
	}
}

// TestCTypeMissingBranches exercises the CType switch cases not yet covered by
// the existing TestCTypeMapping test — specifically int16, int32, uint8, uint16,
// uint32, float32, and the void* fall-through for ObjC objects.
func TestCTypeMissingBranches(t *testing.T) {
	m := newTestMapper()

	cases := []struct {
		qt, want string
	}{
		// int / uint fixed-width types mapped through primitiveGoType then CType
		{"int8_t", "int8_t"},
		{"int16_t", "int16_t"},
		{"int32_t", "int32_t"},
		{"uint8_t", "uint8_t"},
		{"uint16_t", "uint16_t"},
		{"uint32_t", "uint32_t"},
		// floating-point aliases
		{"CGFloat", "double"},
		{"NSTimeInterval", "double"},
		// BOOL → bool in GoType, bool in CType
		{"BOOL", "bool"},
	}

	for _, c := range cases {
		ctx := newTestCtx(m, "Foundation")
		got := m.CType(c.qt, ctx, nil)
		if got != c.want {
			t.Errorf("CType(%q) = %q, want %q", c.qt, got, c.want)
		}
	}
}

// TestPrimitiveGoTypeAllBranches exhaustively verifies every case in the
// primitiveGoType switch to close the coverage gap for that function.
func TestPrimitiveGoTypeAllBranches(t *testing.T) {
	m := newTestMapper()

	cases := []struct {
		qt, want string
	}{
		// Exact-match cases not covered by existing primitive tests
		{"signed char", "int8"},
		{"unsigned char", "uint8"},
		{"short", "int16"},
		{"signed short", "int16"},
		{"short int", "int16"},
		{"unsigned short", "uint16"},
		{"unsigned short int", "uint16"},
		{"int", "int32"},
		{"signed", "int32"},
		{"signed int", "int32"},
		{"unsigned", "uint32"},
		{"unsigned int", "uint32"},
		{"long", "int64"},
		{"signed long", "int64"},
		{"long int", "int64"},
		{"unsigned long", "uint64"},
		{"unsigned long int", "uint64"},
		{"long long", "int64"},
		{"signed long long", "int64"},
		{"long long int", "int64"},
		{"unsigned long long", "uint64"},
		{"unsigned long long int", "uint64"},
		{"long double", "float64"},
		{"int8_t", "int8"},
		{"int16_t", "int16"},
		{"int32_t", "int32"},
		{"int64_t", "int64"},
		{"uint8_t", "uint8"},
		{"uint16_t", "uint16"},
		{"uint32_t", "uint32"},
		{"uint64_t", "uint64"},
		{"intptr_t", "int64"},
		{"ptrdiff_t", "int64"},
		{"uintptr_t", "uint64"},
		{"size_t", "uint64"},
		{"NSInteger", "int64"},
		{"NSUInteger", "uint64"},
		{"CGFloat", "float64"},
		{"NSTimeInterval", "float64"},
		{"unichar", "uint16"},
		{"wchar_t", "uint32"},
		// non-primitive (should return unsafe.Pointer via GoType)
	}

	for _, c := range cases {
		ctx := newTestCtx(m, "Foundation")
		got := m.GoType(c.qt, ctx, nil)
		if got != c.want {
			t.Errorf("GoType(%q) = %q, want %q", c.qt, got, c.want)
		}
	}
}

// TestGoBlockType exercises the buildBlockGoType helper indirectly by invoking GoType
// on block strings that are routed through resolveGoType → buildBlockGoType.
// Note: resolveGoType currently returns unsafe.Pointer for blocks (IsBlock fast-
// path), so buildBlockGoType is only reachable via the exported buildBlockGoType method
// which is unexported. We therefore test by confirming the block fast-path and
// TestGoTypeBlock verifies that GoType resolves block types to primitive Go func
// types matching the generated MakeBlock_* factory signatures.
func TestGoTypeBlockFastPath(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")

	cases := []struct {
		block string
		want  string
	}{
		// No args, void return → func()
		{"void (^)()", "func()"},
		// Object args (NSString*, NSError*) → unsafe.Pointer; void return → func(unsafe.Pointer, unsafe.Pointer)
		{"void (^)(NSString *, NSError *)", "func(unsafe.Pointer, unsafe.Pointer)"},
		// BOOL return + object arg → func(unsafe.Pointer) bool
		{"BOOL (^)(NSString *)", "func(unsafe.Pointer) bool"},
		// void args, id return → func() unsafe.Pointer
		{"id (^)(void)", "func() unsafe.Pointer"},
	}
	for _, c := range cases {
		got := m.GoType(c.block, ctx, nil)
		if got != c.want {
			t.Errorf("GoType(%q) = %q, want %q", c.block, got, c.want)
		}
	}
}

// TestGoBlockTypeProducesCorrectFuncStrings verifies that buildBlockGoType returns
// primitive Go func types (matching MakeBlock_* factory signatures) rather than
// ObjC-resolved types. All ObjC objects become unsafe.Pointer; primitives stay as-is.
func TestGoBlockTypeProducesCorrectFuncStrings(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")

	cases := []struct {
		block string
		want  string
	}{
		// void (^)() → func()
		{"void (^)()", "func()"},
		// void (^)(NSString *) → all objects become unsafe.Pointer
		{"void (^)(NSString *)", "func(unsafe.Pointer)"},
		// BOOL (^)(NSString *) → func(unsafe.Pointer) bool
		{"BOOL (^)(NSString *)", "func(unsafe.Pointer) bool"},
		// void (^)(void) → func()
		{"void (^)(void)", "func()"},
		// NSString * (^)(void) → object return becomes unsafe.Pointer
		{"NSString * (^)(void)", "func() unsafe.Pointer"},
		// malformed → empty string (not unsafe.Pointer)
		{"not a block", ""},
	}

	for _, c := range cases {
		got := m.buildBlockGoType(c.block, ctx)
		if got != c.want {
			t.Errorf("buildBlockGoType(%q) = %q, want %q", c.block, got, c.want)
		}
	}
}

// TestResolveGoTypeStructSameFramework verifies that a struct registered in
// StructIndex and owned by the current framework resolves to the bare name.
func TestResolveGoTypeStructSameFramework(t *testing.T) {
	m := newTestMapper()
	m.StructIndex = map[string]string{"NSRange": "Foundation"}
	ctx := newTestCtx(m, "Foundation")

	got := m.GoType("NSRange", ctx, nil)
	if got != "NSRange" {
		t.Errorf("GoType(NSRange same-fw) = %q, want %q", got, "NSRange")
	}
}

// TestResolveGoTypeStructCrossFramework verifies that a struct owned by another
// framework resolves to a package-qualified value-type reference and updates
// the ImportSet as a side effect.
func TestResolveGoTypeStructCrossFramework(t *testing.T) {
	m := newTestMapper()
	m.StructIndex = map[string]string{"CGSize": "CoreFoundation"}
	m.ModulePrefix = "example.com/frameworks"
	ctx := newTestCtx(m, "Virtualization")
	imports := make(ImportSet)

	got := m.GoType("CGSize", ctx, imports)
	if got != "corefoundation.CGSize" {
		t.Errorf("GoType(CGSize cross-fw) = %q, want corefoundation.CGSize", got)
	}
	if imports["corefoundation"] == "" {
		t.Error("imports[corefoundation] not set as side effect")
	}
}

// TestResolveGoTypeStructPointer verifies that pointer-to-struct resolves to
// `*<pkg>.<Name>` via resolvePointerType for cross-framework, bare `*<Name>` for
// same-framework.
func TestResolveGoTypeStructPointer(t *testing.T) {
	m := newTestMapper()
	m.StructIndex = map[string]string{"CGSize": "CoreFoundation"}
	m.ModulePrefix = "example.com/frameworks"
	ctx := newTestCtx(m, "Virtualization")

	got := m.GoType("CGSize *", ctx, nil)
	if got != "*corefoundation.CGSize" {
		t.Errorf("GoType(CGSize *) = %q, want *corefoundation.CGSize", got)
	}
}

// TestResolveGoTypeUnknownStructFallsBack confirms that a name with no
// StructIndex entry still falls through to unsafe.Pointer.
func TestResolveGoTypeUnknownStructFallsBack(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")

	got := m.GoType("UnregisteredStruct", ctx, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("GoType(UnregisteredStruct) = %q, want unsafe.Pointer", got)
	}
}

// TestResolveGoTypeStructReturnIsTyped verifies that value-type struct returns
// resolve to a typed Go struct (not unsafe.Pointer). The bridge malloc's the
// struct and returns void*; the emitter handles dereference+free at the call site.
func TestResolveGoTypeStructReturnIsTyped(t *testing.T) {
	m := newTestMapper()
	m.StructIndex = map[string]string{"CGSize": "CoreFoundation"}
	m.ModulePrefix = "example.com/frameworks"
	ctx := newTestCtx(m, "Virtualization")
	ctx.IsReturn = true

	got := m.GoType("CGSize", ctx, nil)
	if got != "corefoundation.CGSize" {
		t.Errorf("GoType(CGSize in return position) = %q, want corefoundation.CGSize", got)
	}
}

// TestResolveGoTypeIDProtocolSingleSameFramework verifies that `id<Foo>` with
// Foo owned by the current framework resolves to the bare `Fooable` interface.
func TestResolveGoTypeIDProtocolSingleSameFramework(t *testing.T) {
	m := newTestMapper()
	m.ProtocolIndex = map[string]string{"VZGraphicsDisplayObserver": "Virtualization"}
	ctx := newTestCtx(m, "Virtualization")

	got := m.GoType("id<VZGraphicsDisplayObserver>", ctx, nil)
	if got != "VZGraphicsDisplayObserver" {
		t.Errorf("GoType(id<VZ…Observer>) = %q, want VZGraphicsDisplayObserver", got)
	}
}

// TestResolveGoTypeIDProtocolSingleCrossFramework verifies cross-framework
// package qualification and import side effect for an id<Protocol> reference.
func TestResolveGoTypeIDProtocolSingleCrossFramework(t *testing.T) {
	m := newTestMapper()
	m.ProtocolIndex = map[string]string{"NSCopying": "Foundation"}
	m.ModulePrefix = "example.com/frameworks"
	ctx := newTestCtx(m, "AppKit")
	imports := make(ImportSet)

	got := m.GoType("id<NSCopying>", ctx, imports)
	if got != "foundation.NSCopying" {
		t.Errorf("GoType(id<NSCopying>) = %q, want foundation.NSCopying", got)
	}
	if imports["foundation"] == "" {
		t.Error("imports[foundation] not set as side effect")
	}
}

// TestResolveGoTypeIDProtocolMulti verifies that `id<P1, P2>` resolves to an
// inline anonymous composite interface, preserving both conformance constraints.
func TestResolveGoTypeIDProtocolMulti(t *testing.T) {
	m := newTestMapper()
	m.ProtocolIndex = map[string]string{
		"NSCopying":      "Foundation",
		"NSSecureCoding": "Foundation",
	}
	m.ModulePrefix = "example.com/frameworks"
	ctx := newTestCtx(m, "AppKit")

	got := m.GoType("id<NSCopying, NSSecureCoding>", ctx, nil)
	want := "interface { foundation.NSCopying; foundation.NSSecureCoding }"
	if got != want {
		t.Errorf("GoType(id<NSCopying, NSSecureCoding>) = %q, want %q", got, want)
	}
}

// TestResolveGoTypeIDProtocolUnknownFallsBack verifies that an id<Protocol>
// whose protocol is not in ProtocolIndex falls back to objc.Object rather
// than emitting a dangling interface name.
func TestResolveGoTypeIDProtocolUnknownFallsBack(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Virtualization")

	got := m.GoType("id<MysteryProtocol>", ctx, nil)
	if got != "objc.Object" {
		t.Errorf("GoType(id<MysteryProtocol>) = %q, want objc.Object", got)
	}
}

// TestResolveGoTypeIDBareReturnsObjcObject guards the documented behavior
// that bare `id` (no protocol annotation) resolves to objc.Object.
func TestResolveGoTypeIDBareReturnsObjcObject(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")

	got := m.GoType("id", ctx, nil)
	if got != "objc.Object" {
		t.Errorf("GoType(id) = %q, want objc.Object", got)
	}
}

// TestGoPointerTypeVoidStar covers the void * → unsafe.Pointer branch in
// resolvePointerType via the HasPrefix check.
func TestGoPointerTypeVoidStarVariants(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")

	cases := []struct {
		qt, want string
	}{
		{"void *", "unsafe.Pointer"},
		{"void * restrict", "unsafe.Pointer"}, // normalised: void *
		{"void **", "unsafe.Pointer"},          // double void pointer falls through to class check → unsafe.Pointer
	}
	for _, c := range cases {
		got := m.GoType(c.qt, ctx, nil)
		if got != "unsafe.Pointer" {
			t.Errorf("GoType(%q) = %q, want unsafe.Pointer", c.qt, got)
		}
	}
}

// TestGoPointerTypeGenericUnknownClass verifies the path where a type has a
// generic parameter but the base class is NOT in ClassNameIndex — it should fall
// back to unsafe.Pointer.
func TestGoPointerTypeGenericUnknownClass(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	// NSSet is not in ClassNameIndex for this test mapper.
	got := m.GoType("NSSet<ObjectType> *", ctx, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("GoType(NSSet<ObjectType> *) = %q, want unsafe.Pointer (unknown class)", got)
	}
}

// TestGoPointerTypeGenericKnownNonGenericClass verifies the path where a type
// has a generic param inside angle brackets but the base class IS in ClassNameIndex
// and is NOT a generic class itself.
func TestGoPointerTypeGenericKnownNonGenericClass(t *testing.T) {
	// Create a mapper where NSArray is known but NOT marked as generic.
	m := &Mapper{
		OwnerIndex:   map[string]string{"NSArray": "Foundation"},
		ModulePrefix: "github.com/deploymenttheory/go-bindings-macosplatform/frameworks",
		// GenericClasses intentionally empty
	}
	ctx := m.BaseContext("AppKit", map[string]bool{"NSArray": true})
	// NSArray<NSString *> * — base class is known but not generic: returns qualified type
	// without [T] or [objc.Object]
	got := m.GoType("NSArray<NSString *> *", ctx, nil)
	if got != "*foundation.NSArray" {
		t.Errorf("GoType(NSArray<NSString *> *) with non-generic class = %q, want *foundation.NSArray", got)
	}
}

// TestGoPointerTypeGenericParamIsCurrentClassParam covers the branch where the
// generic type argument IS one of the current class's GenericParams → returns
// the qualified base class with [T].
func TestGoPointerTypeGenericParamInScope(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	ctx.GenericParams = []string{"ElementType"}

	// NSArray<ElementType> * — ElementType is in scope as a class generic param
	got := m.GoType("NSArray<ElementType> *", ctx, nil)
	// In Foundation context NSArray is owned by Foundation → *NSArray[T]
	if got != "*NSArray[T]" {
		t.Errorf("GoType(NSArray<ElementType> *) with param in scope = %q, want *NSArray[T]", got)
	}
}

// TestGoPointerTypeClassIsGenericParam verifies that when a pointer type's class
// name is itself one of the current class's GenericParams (e.g. "ObjectType *"),
// the type resolves to unsafe.Pointer.
func TestGoPointerTypeClassIsGenericParam(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	ctx.GenericParams = []string{"ObjectType"}

	// "ObjectType *" — the class name IS a generic param → unsafe.Pointer
	got := m.GoType("ObjectType *", ctx, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("GoType(ObjectType *) with ObjectType as generic param = %q, want unsafe.Pointer", got)
	}
}

// TestGoTypeImportsNilSafe verifies that GoType does not panic when imports is nil.
func TestGoTypeImportsNilSafe(t *testing.T) {
	m := newTestMapper()
	ctx := m.BaseContext("AppKit", map[string]bool{"NSString": true})
	// imports is nil — cross-framework resolution must not panic.
	got := m.GoType("NSString *", ctx, nil)
	if got != "*foundation.NSString" {
		t.Errorf("GoType(NSString *) with nil imports = %q, want *foundation.NSString", got)
	}
}

// TestGoTypeImportsNilSafeCacheHit verifies the same nil-safety on a cache hit.
func TestGoTypeImportsNilSafeCacheHit(t *testing.T) {
	m := newTestMapper()
	// Warm the cache with a non-nil imports.
	warmCtx := m.BaseContext("AppKit", map[string]bool{"NSString": true})
	m.GoType("NSString *", warmCtx, make(ImportSet))

	// Second call with nil imports must not panic.
	hitCtx := m.BaseContext("AppKit", map[string]bool{"NSString": true})
	got := m.GoType("NSString *", hitCtx, nil)
	if got != "*foundation.NSString" {
		t.Errorf("GoType(NSString *) cache hit with nil imports = %q, want *foundation.NSString", got)
	}
}

// TestGoPointerTypeFinalFallback covers the final `return "unsafe.Pointer"` at
// the bottom of resolvePointerType. This path is reached when the type has a `*` but
// ClassName returns "" and the base type is neither a known primitive, struct,
// enum, nor BSD type. Primitive pointers (int *, uint8_t *, etc.) now resolve
// to typed Go pointer types (*int32, *uint8, etc.) in parameter position.
func TestGoPointerTypeFinalFallback(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")

	// Primitive pointer types (single-word and multi-word) resolve to *T in
	// parameter position.
	primitivePointers := []struct {
		qt   string
		want string
	}{
		{"int *", "*int32"},
		{"uint8_t *", "*uint8"},
		{"uint16_t *", "*uint16"},
		{"uint32_t *", "*uint32"},
		{"unsigned char *", "*uint8"},
		{"unsigned int *", "*uint32"},
		{"unsigned short *", "*uint16"},
	}
	for _, c := range primitivePointers {
		if got := m.GoType(c.qt, ctx, nil); got != c.want {
			t.Errorf("GoType(%q) = %q, want %q", c.qt, got, c.want)
		}
	}

	// Double pointers still fall back to unsafe.Pointer.
	fallbackPointers := []string{
		"char **", // double pointer
	}
	for _, qt := range fallbackPointers {
		if got := m.GoType(qt, ctx, nil); got != "unsafe.Pointer" {
			t.Errorf("GoType(%q) = %q, want unsafe.Pointer (final fallback)", qt, got)
		}
	}
}

// TestCTypeFloat32 covers the float32 → "float" CType branch.
func TestCTypeFloat32(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	got := m.CType("float", ctx, nil)
	if got != "float" {
		t.Errorf("CType(float) = %q, want float", got)
	}
}

// TestCTypeString covers the string → "const char *" CType branch.
func TestCTypeString(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	got := m.CType("SEL", ctx, nil)
	if got != "const char *" {
		t.Errorf("CType(SEL) = %q, want const char *", got)
	}
}

// TestCFTypedefCrossFramework verifies that a CF opaque typedef (e.g. CFStringRef)
// used in a non-CoreFoundation framework produces a qualified cross-framework
// pointer and records the corefoundation import.
func TestCFTypedefCrossFramework(t *testing.T) {
	m := &Mapper{
		ModulePrefix: "github.com/deploymenttheory/go-bindings-macosplatform/frameworks",
	}
	ctx := m.BaseContext("Security", map[string]bool{})
	imports := make(ImportSet)

	got := m.GoType("CFStringRef", ctx, imports)
	want := "*corefoundation.CFStringRef"
	if got != want {
		t.Errorf("GoType(CFStringRef, Security) = %q, want %q", got, want)
	}
	if imports["corefoundation"] == "" {
		t.Error("expected imports to contain a corefoundation import path")
	}
}

// TestCFTypedefSameFramework verifies that a CF typedef used within CoreFoundation
// itself produces a local (unqualified) pointer.
func TestCFTypedefSameFramework(t *testing.T) {
	m := &Mapper{
		ModulePrefix: "github.com/deploymenttheory/go-bindings-macosplatform/frameworks",
	}
	ctx := m.BaseContext("CoreFoundation", map[string]bool{})
	imports := make(ImportSet)

	got := m.GoType("CFStringRef", ctx, imports)
	want := "*CFStringRef"
	if got != want {
		t.Errorf("GoType(CFStringRef, CoreFoundation) = %q, want %q", got, want)
	}
	if len(imports) != 0 {
		t.Errorf("GoType within CoreFoundation should not add imports, got %v", imports)
	}
}

// TestCFTypedefDoublePointerStaysUnsafe verifies that "CFStringRef *" (a double
// pointer out-parameter) is NOT treated as a CF typedef and stays unsafe.Pointer.
func TestCFTypedefDoublePointerStaysUnsafe(t *testing.T) {
	m := &Mapper{
		ModulePrefix: "github.com/deploymenttheory/go-bindings-macosplatform/frameworks",
	}
	ctx := m.BaseContext("Security", map[string]bool{})

	// "CFStringRef *" has a *, so it goes through resolvePointerType (not the CF typedef path).
	// ClassName returns "CFStringRef", which is not in ClassNameIndex → unsafe.Pointer.
	got := m.GoType("CFStringRef *", ctx, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("GoType(CFStringRef *, Security) = %q, want unsafe.Pointer (double pointer out-param)", got)
	}
}

// TestCFTypedefBlockedImport verifies that when the import of CoreFoundation is
// blocked, CF typedef resolution falls back to unsafe.Pointer.
func TestCFTypedefBlockedImport(t *testing.T) {
	m := &Mapper{
		ModulePrefix: "github.com/deploymenttheory/go-bindings-macosplatform/frameworks",
		BlockedImports: map[string]map[string]bool{
			"Security": {"CoreFoundation": true},
		},
	}
	ctx := m.BaseContext("Security", map[string]bool{})

	got := m.GoType("CFStringRef", ctx, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("GoType(CFStringRef, Security [blocked]) = %q, want unsafe.Pointer", got)
	}
}

// TestCFScalarPrimitives verifies that CF scalar typedef names map to the
// correct Go primitive types.
func TestCFScalarPrimitives(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "CoreFoundation")

	cases := []struct {
		qt, want string
	}{
		{"CFIndex", "int64"},
		{"CFOptionFlags", "uint64"},
		{"CFTypeID", "uint64"},
		{"CFHashCode", "uint64"},
		{"CFTimeInterval", "float64"},
		{"CFAbsoluteTime", "float64"},
		{"CFStringEncoding", "uint32"},
	}
	for _, c := range cases {
		got := m.GoType(c.qt, ctx, nil)
		if got != c.want {
			t.Errorf("GoType(%q) = %q, want %q", c.qt, got, c.want)
		}
	}
}

