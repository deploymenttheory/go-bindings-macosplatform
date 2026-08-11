package rawlib

import (
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// testMapper returns a minimal Mapper suitable for unit tests.
func testMapper() *typemap.Mapper {
	return &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex:     map[string]string{},
		ModulePrefix:   "github.com/example/fw",
	}
}

// testCtx returns a base Context for the given framework with an empty ClassNameIndex set.
func testCtx(framework string) typemap.Context {
	m := testMapper()
	return m.BaseContext(framework, map[string]bool{})
}

// --------------------------------------------------------------------------
// buildParamNames
// --------------------------------------------------------------------------

func TestResolveArgNamesBasic(t *testing.T) {
	args := []macosplatformmetadata.Param{
		{Name: "url"},
		{Name: "encoding"},
	}
	got := buildParamNames(args)
	want := []string{"url", "encoding"}
	if len(got) != len(want) {
		t.Fatalf("len: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestResolveArgNamesUnnamed(t *testing.T) {
	args := []macosplatformmetadata.Param{
		{Name: ""},
		{Name: ""},
	}
	got := buildParamNames(args)
	want := []string{"arg0", "arg1"}
	if len(got) != len(want) {
		t.Fatalf("len: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestResolveArgNamesCollision(t *testing.T) {
	// Two args sharing the same name must both be renamed.
	args := []macosplatformmetadata.Param{
		{Name: "index"},
		{Name: "index"},
	}
	got := buildParamNames(args)
	if len(got) != 2 {
		t.Fatalf("len: got %d, want 2", len(got))
	}
	if got[0] == got[1] {
		t.Errorf("expected distinct names, both are %q", got[0])
	}
	// Both should contain the base "index".
	for _, n := range got {
		if n == "index" {
			t.Errorf("bare %q should have been disambiguated", n)
		}
	}
}

func TestResolveArgNamesGoKeyword(t *testing.T) {
	// "default" is a Go keyword; ParamName appends "_".
	args := []macosplatformmetadata.Param{
		{Name: "Default"},
	}
	got := buildParamNames(args)
	if len(got) != 1 {
		t.Fatalf("len: got %d, want 1", len(got))
	}
	if got[0] != "default_" {
		t.Errorf("got %q, want %q", got[0], "default_")
	}
}

// --------------------------------------------------------------------------
// buildGoArgs
// --------------------------------------------------------------------------

func TestBuildGoArgsEmpty(t *testing.T) {
	m := testMapper()
	ctx := testCtx("Foundation")
	got := buildGoArgs(nil, false, ctx, m, nil)
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestBuildGoArgsWithTypes(t *testing.T) {
	m := testMapper()
	ctx := testCtx("Foundation")
	args := []macosplatformmetadata.Param{
		{Name: "count", ObjCType: "NSUInteger"},
	}
	got := buildGoArgs(args, false, ctx, m, nil)
	if len(got) != 1 {
		t.Fatalf("len: got %d, want 1", len(got))
	}
	// NSUInteger → uint64
	want := "count uint64"
	if got[0] != want {
		t.Errorf("got %q, want %q", got[0], want)
	}
}

func TestBuildGoArgsObjCTypeResolution(t *testing.T) {
	m := testMapper()
	ctx := testCtx("Foundation")
	// ObjC primitive types are resolved to Go types by the typemap.
	args := []macosplatformmetadata.Param{
		{Name: "value", ObjCType: "int32_t"},
	}
	got := buildGoArgs(args, false, ctx, m, nil)
	if len(got) != 1 {
		t.Fatalf("len: got %d, want 1", len(got))
	}
	if got[0] != "value int32" {
		t.Errorf("got %q, want %q", got[0], "value int32")
	}
}

func TestBuildGoArgsDefaultsToUnsafePointer(t *testing.T) {
	m := testMapper()
	ctx := testCtx("Foundation")
	// An ObjC type unknown to the mapper resolves to unsafe.Pointer.
	args := []macosplatformmetadata.Param{
		{Name: "thing", ObjCType: "SomeCompletelyUnknownObjCType"},
	}
	got := buildGoArgs(args, false, ctx, m, nil)
	if len(got) != 1 {
		t.Fatalf("len: got %d, want 1", len(got))
	}
	if got[0] != "thing unsafe.Pointer" {
		t.Errorf("got %q, want %q", got[0], "thing unsafe.Pointer")
	}
}

// --------------------------------------------------------------------------
// buildGoReturn
// --------------------------------------------------------------------------

func TestBuildGoReturnVoid(t *testing.T) {
	m := testMapper()
	ctx := testCtx("Foundation")
	method := macosplatformmetadata.Method{
		Selector: "doSomething",
		Return:   macosplatformmetadata.ReturnType{ObjCType: ""},
	}
	got := buildGoReturn(method, ctx, m, "", nil)
	if got != "" {
		t.Errorf("void method: got %q, want empty string", got)
	}
}

func TestBuildGoReturnPrimitive(t *testing.T) {
	m := testMapper()
	ctx := testCtx("Foundation")
	method := macosplatformmetadata.Method{
		Selector: "count",
		Return:   macosplatformmetadata.ReturnType{ObjCType: "uint64_t"},
	}
	got := buildGoReturn(method, ctx, m, "", nil)
	if got != "uint64" {
		t.Errorf("got %q, want %q", got, "uint64")
	}
}

func TestBuildGoReturnInstancetype(t *testing.T) {
	m := testMapper()
	ctx := testCtx("Foundation")
	method := macosplatformmetadata.Method{
		Selector: "init",
		Return:   macosplatformmetadata.ReturnType{IsInstancetype: true},
	}
	got := buildGoReturn(method, ctx, m, "NSObject", nil)
	if got != "*NSObject" {
		t.Errorf("got %q, want %q", got, "*NSObject")
	}
}

func TestBuildGoReturnInstancetypeGeneric(t *testing.T) {
	m := testMapper()
	ctx := testCtx("Foundation")
	// GenericParams in scope → returns *ClassName[T]
	ctx.GenericParams = []string{"ObjectType"}
	method := macosplatformmetadata.Method{
		Selector: "init",
		Return:   macosplatformmetadata.ReturnType{IsInstancetype: true},
	}
	got := buildGoReturn(method, ctx, m, "NSArray", nil)
	if got != "*NSArray[T]" {
		t.Errorf("got %q, want %q", got, "*NSArray[T]")
	}
}

func TestBuildGoReturnInstancetypeNoClass(t *testing.T) {
	m := testMapper()
	ctx := testCtx("Foundation")
	method := macosplatformmetadata.Method{
		Selector: "init",
		Return:   macosplatformmetadata.ReturnType{IsInstancetype: true},
	}
	// No className supplied → fallback to unsafe.Pointer.
	got := buildGoReturn(method, ctx, m, "", nil)
	if got != "unsafe.Pointer" {
		t.Errorf("got %q, want %q", got, "unsafe.Pointer")
	}
}

func TestBuildGoReturnNSError(t *testing.T) {
	m := testMapper()
	ctx := testCtx("Foundation")
	method := macosplatformmetadata.Method{
		Selector:  "readWithError:",
		IsNSError: true,
		Return:    macosplatformmetadata.ReturnType{ObjCType: "NSUInteger"},
	}
	got := buildGoReturn(method, ctx, m, "", nil)
	// uint64 + error → "(uint64, error)"
	want := "(uint64, error)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildGoReturnNSErrorOnly(t *testing.T) {
	m := testMapper()
	ctx := testCtx("Foundation")
	method := macosplatformmetadata.Method{
		Selector:  "doThingWithError:",
		IsNSError: true,
		Return:    macosplatformmetadata.ReturnType{ObjCType: ""},
	}
	// void + NSError → bare "error"
	got := buildGoReturn(method, ctx, m, "", nil)
	if got != "error" {
		t.Errorf("got %q, want %q", got, "error")
	}
}

// --------------------------------------------------------------------------
// availabilityComment
// --------------------------------------------------------------------------

func TestAvailabilityCommentPresent(t *testing.T) {
	av := macosplatformmetadata.Availability{MacOSIntroduced: "12.0"}
	got := availabilityComment(av)
	want := "Introduced: macOS 12.0"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAvailabilityCommentAbsent(t *testing.T) {
	av := macosplatformmetadata.Availability{}
	got := availabilityComment(av)
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

// ── isPrimitiveGoType ─────────────────────────────────────────────────────────

func TestIsPrimitiveGoTypeBool(t *testing.T) {
	if !isPrimitiveGoType("bool") {
		t.Error("bool should be a primitive Go type")
	}
}

func TestIsPrimitiveGoTypeInt8(t *testing.T) {
	if !isPrimitiveGoType("int8") {
		t.Error("int8 should be a primitive Go type")
	}
}

func TestIsPrimitiveGoTypeFloat64(t *testing.T) {
	if !isPrimitiveGoType("float64") {
		t.Error("float64 should be a primitive Go type")
	}
}

func TestIsPrimitiveGoTypeNonPrimitive(t *testing.T) {
	if isPrimitiveGoType("*NSView") {
		t.Error("*NSView should not be a primitive Go type")
	}
	if isPrimitiveGoType("string") {
		t.Error("string should not be a primitive Go type")
	}
	if isPrimitiveGoType("unsafe.Pointer") {
		t.Error("unsafe.Pointer should not be a primitive Go type")
	}
}
