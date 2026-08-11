package rawlib

import (
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// These cover the helpers relocated into helpers.go from the retired cgo
// class/bridge emitters, since they remain reached by the live purego path
// (isObjectReturn/isKnownStruct/objectConstructExpr via the purego function
// emitter; isUPPFunction/hasByValueUnknownTypeFor via EmittableFunctions).

func TestIsObjectReturn(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"*NSString", true},
		{"*coregraphics.CGImageRef", true},
		{"NSString", false}, // not a pointer
		{"*int32", false},   // pointer to primitive
		{"*bool", false},
		{"*float64", false},
		{"unsafe.Pointer", false},
	}
	for _, c := range cases {
		if got := isObjectReturn(c.in); got != c.want {
			t.Errorf("isObjectReturn(%q) = %v; want %v", c.in, got, c.want)
		}
	}
}

func TestIsKnownStruct(t *testing.T) {
	m := &typemap.Mapper{StructIndex: map[string]string{"CGRect": "CoreGraphics", "decform": "CarbonCore"}}
	m.RebuildGoNameIndices()
	if !isKnownStruct("CGRect", m) {
		t.Error("isKnownStruct(CGRect) = false; want true (StructIndex)")
	}
	if !isKnownStruct("Decform", m) {
		t.Error("isKnownStruct(Decform) = false; want true (lowercase-first fallback)")
	}
	if isKnownStruct("NotAStruct", m) {
		t.Error("isKnownStruct(NotAStruct) = true; want false")
	}
}

func TestObjectConstructExpr(t *testing.T) {
	m := &typemap.Mapper{}
	cases := []struct{ in, want string }{
		{"NSString", "NewNSString(_ptr)"},                       // same-framework
		{"foundation.NSString", "foundation.NewNSString(_ptr)"}, // cross-framework
		{"NSArray[T]", "NewNSArrayT[T](_ptr)"},                  // same-framework generic T
		{"NSArray[runtime.Object]", "NewNSArray(_ptr)"},         // same-framework non-T generic
		{"foundation.NSArray[T]", "foundation.NewNSArrayT[T](_ptr)"},
	}
	for _, c := range cases {
		if got := objectConstructExpr(c.in, "_ptr", nil, m); got != c.want {
			t.Errorf("objectConstructExpr(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestIsUPPFunction(t *testing.T) {
	for _, n := range []string{"InvokeMenuDrawingUPP", "NewMenuDrawingUPP", "DisposeMenuDrawingUPP"} {
		if !isUPPFunction(n) {
			t.Errorf("isUPPFunction(%q) = false; want true", n)
		}
	}
	for _, n := range []string{"MenuDrawing", "InvokeSomething", "UPPHelper"} {
		if isUPPFunction(n) {
			t.Errorf("isUPPFunction(%q) = true; want false", n)
		}
	}
}

func TestIsVAList(t *testing.T) {
	if !isVAList("va_list") || !isVAList("struct __va_list_tag *") {
		t.Error("isVAList should match va_list / __va_list")
	}
	if isVAList("int") {
		t.Error("isVAList(int) = true; want false")
	}
}

func TestHasByValueUnknownTypeFor(t *testing.T) {
	fw := &macosplatformmetadata.FrameworkMeta{
		Framework: "Compression",
		Enums:     map[string]macosplatformmetadata.Enum{"compression_algorithm": {}},
	}
	// A by-value enum parameter is a plain integer — allowed (not "unknown").
	enumArg := macosplatformmetadata.Function{
		Params: []macosplatformmetadata.Param{{ObjCType: "compression_algorithm"}},
	}
	if hasByValueUnknownTypeFor(fw, enumArg) {
		t.Error("by-value enum param wrongly flagged as unknown")
	}
	// A SIMD/struct-by-value type (no _t/Ref suffix, not an enum) IS unknown.
	simd := macosplatformmetadata.Function{
		Params: []macosplatformmetadata.Param{{ObjCType: "DenseMatrix_Float"}},
	}
	if !hasByValueUnknownTypeFor(fw, simd) {
		t.Error("by-value SIMD type not flagged as unknown")
	}
	// Scalars and pointers are fine.
	ok := macosplatformmetadata.Function{
		Params: []macosplatformmetadata.Param{{ObjCType: "int"}, {ObjCType: "void *"}},
	}
	if hasByValueUnknownTypeFor(fw, ok) {
		t.Error("scalar/pointer params wrongly flagged as unknown")
	}
}
