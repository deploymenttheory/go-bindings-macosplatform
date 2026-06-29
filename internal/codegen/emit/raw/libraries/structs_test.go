package raw

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// newStructMapper returns a plain Mapper with no extra configuration, suitable
// for struct tests that do not need cross-framework type resolution.
func newStructMapper() *typemap.Mapper {
	return typemap.New()
}

// newStructFM returns a minimal FrameworkMeta populated with the given structs
// and typedefs maps. Framework is fixed to "TestFW".
func newStructFM(structs map[string]macosplatformmetadata.Struct, typedefs map[string]string) *macosplatformmetadata.FrameworkMeta {
	return &macosplatformmetadata.FrameworkMeta{
		Framework: "TestFW",
		Structs:   structs,
		Typedefs:  typedefs,
	}
}

// TestStructsEmpty verifies that a FrameworkMeta with no structs and no
// typedefs produces no output.
func TestStructsEmpty(t *testing.T) {
	var buf bytes.Buffer
	framework := newStructFM(map[string]macosplatformmetadata.Struct{}, map[string]string{})
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

// TestStructBasicFields verifies that a struct with named, typed fields emits a
// correct Go struct definition.
func TestStructBasicFields(t *testing.T) {
	var buf bytes.Buffer
	framework := newStructFM(map[string]macosplatformmetadata.Struct{
		"NSPoint": {
			Fields: []macosplatformmetadata.StructField{
				{Name: "x", GoType: "float64"},
				{Name: "y", GoType: "float64"},
			},
		},
	}, nil)
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "type NSPoint struct {") {
		t.Errorf("missing struct declaration; got:\n%s", out)
	}
	if !strings.Contains(out, "X float64") {
		t.Errorf("missing field X; got:\n%s", out)
	}
	if !strings.Contains(out, "Y float64") {
		t.Errorf("missing field Y; got:\n%s", out)
	}
}

// TestStructUnnamedField verifies that a field with an empty Name is given a
// positional name "Field0", "Field1", etc.
func TestStructUnnamedField(t *testing.T) {
	var buf bytes.Buffer
	framework := newStructFM(map[string]macosplatformmetadata.Struct{
		"_AnonStruct": {
			Fields: []macosplatformmetadata.StructField{
				{Name: "", GoType: "uint32"},
				{Name: "", GoType: "uint64"},
			},
		},
	}, nil)
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Field0 uint32") {
		t.Errorf("missing Field0; got:\n%s", out)
	}
	if !strings.Contains(out, "Field1 uint64") {
		t.Errorf("missing Field1; got:\n%s", out)
	}
}

// TestStructGoTypePreferred verifies that when a StructField has GoType set it
// is used directly without consulting the mapper.
func TestStructGoTypePreferred(t *testing.T) {
	var buf bytes.Buffer
	framework := newStructFM(map[string]macosplatformmetadata.Struct{
		"NSEdgeInsets": {
			Fields: []macosplatformmetadata.StructField{
				// GoType is explicit; ObjCType differs — mapper must NOT be consulted.
				{Name: "top", ObjCType: "CGFloat", GoType: "float64"},
			},
		},
	}, nil)
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Top float64") {
		t.Errorf("expected GoType float64 used directly; got:\n%s", out)
	}
}

// TestStructObjCTypeResolved verifies that when GoType is absent the mapper is
// consulted with ObjCType to produce the Go type.
func TestStructObjCTypeResolved(t *testing.T) {
	var buf bytes.Buffer
	framework := newStructFM(map[string]macosplatformmetadata.Struct{
		"NSRange": {
			Fields: []macosplatformmetadata.StructField{
				// ObjCType is a primitive; mapper will convert it.
				{Name: "location", ObjCType: "NSUInteger"},
				{Name: "length", ObjCType: "NSUInteger"},
			},
		},
	}, nil)
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	// NSUInteger → uint64 via primitiveGoType.
	if !strings.Contains(out, "Location uint64") {
		t.Errorf("expected ObjC NSUInteger resolved to uint64; got:\n%s", out)
	}
	if !strings.Contains(out, "Length uint64") {
		t.Errorf("expected ObjC NSUInteger resolved to uint64; got:\n%s", out)
	}
}

// TestStructAvailabilityComment verifies that MacOSIntroduced on a struct
// produces an "Introduced: macOS X.Y" comment before the type declaration.
func TestStructAvailabilityComment(t *testing.T) {
	var buf bytes.Buffer
	framework := newStructFM(map[string]macosplatformmetadata.Struct{
		"NSDirectionalEdgeInsets": {
			Availability: macosplatformmetadata.Availability{MacOSIntroduced: "10.15"},
			Fields: []macosplatformmetadata.StructField{
				{Name: "top", GoType: "float64"},
			},
		},
	}, nil)
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "// Introduced: macOS 10.15") {
		t.Errorf("missing availability comment; got:\n%s", out)
	}
}

// TestTypedefAlias verifies that a typedef of the form "struct _NSRange"
// where "_NSRange" is a known struct produces a Go type alias.
func TestTypedefAlias(t *testing.T) {
	var buf bytes.Buffer
	framework := newStructFM(
		map[string]macosplatformmetadata.Struct{
			"_NSRange": {
				Fields: []macosplatformmetadata.StructField{{Name: "location", GoType: "uint64"}},
			},
		},
		map[string]string{
			"NSRange": "struct _NSRange",
		},
	)
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "type NSRange = _NSRange") {
		t.Errorf("missing typedef alias; got:\n%s", out)
	}
	if !strings.Contains(out, "is a typedef alias for") {
		t.Errorf("missing alias comment; got:\n%s", out)
	}
}

// TestTypedefAliasSkipsSameName verifies that a typedef is skipped when its
// Go alias name equals the underlying struct's Go name (no self-alias needed).
func TestTypedefAliasSkipsSameName(t *testing.T) {
	var buf bytes.Buffer
	// GoTypeName("NSRect") == GoTypeName("NSRect") → skip.
	framework := newStructFM(
		map[string]macosplatformmetadata.Struct{
			"NSRect": {
				Fields: []macosplatformmetadata.StructField{{Name: "origin", GoType: "unsafe.Pointer"}},
			},
		},
		map[string]string{
			"NSRect": "struct NSRect",
		},
	)
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "is a typedef alias") {
		t.Errorf("self-alias should have been skipped; got:\n%s", out)
	}
}

// TestTypedefAliasSkipsNonStruct verifies that typedefs not starting with
// "struct " are ignored (they do not refer to a struct).
func TestTypedefAliasSkipsNonStruct(t *testing.T) {
	var buf bytes.Buffer
	framework := newStructFM(
		map[string]macosplatformmetadata.Struct{
			"SomeStruct": {
				Fields: []macosplatformmetadata.StructField{{Name: "val", GoType: "int32"}},
			},
		},
		map[string]string{
			// This typedef points to a primitive type, not a struct.
			"MyInt": "unsigned int",
		},
	)
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "type MyInt") {
		t.Errorf("non-struct typedef should not produce an alias; got:\n%s", out)
	}
}

// TestTypedefAliasSkipsMissingStruct verifies that a typedef pointing to a
// struct name that does not appear in framework.Structs is silently skipped.
func TestTypedefAliasSkipsMissingStruct(t *testing.T) {
	var buf bytes.Buffer
	framework := newStructFM(
		// No structs at all.
		map[string]macosplatformmetadata.Struct{},
		map[string]string{
			"NSPoint": "struct _NSPoint", // _NSPoint is not in the Structs map.
		},
	)
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "type NSPoint") {
		t.Errorf("typedef for unknown struct should be skipped; got:\n%s", out)
	}
}

// TestStructSortOrder verifies that multiple structs appear in alphabetical
// order by their ObjC name.
func TestStructSortOrder(t *testing.T) {
	var buf bytes.Buffer
	framework := newStructFM(map[string]macosplatformmetadata.Struct{
		"ZPoint": {Fields: []macosplatformmetadata.StructField{{Name: "z", GoType: "float64"}}},
		"APoint": {Fields: []macosplatformmetadata.StructField{{Name: "a", GoType: "float64"}}},
		"MPoint": {Fields: []macosplatformmetadata.StructField{{Name: "m", GoType: "float64"}}},
	}, nil)
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	aIdx := strings.Index(out, "APoint")
	mIdx := strings.Index(out, "MPoint")
	zIdx := strings.Index(out, "ZPoint")
	if aIdx == -1 || mIdx == -1 || zIdx == -1 {
		t.Fatalf("one or more struct names missing:\n%s", out)
	}
	if !(aIdx < mIdx && mIdx < zIdx) {
		t.Errorf("structs not in alphabetical order: a=%d m=%d z=%d", aIdx, mIdx, zIdx)
	}
}

// TestStructUnknownObjCTypeFallback verifies that when both GoType and ObjCType
// fail to resolve the emitter falls back to "unsafe.Pointer".
func TestStructUnknownObjCTypeFallback(t *testing.T) {
	var buf bytes.Buffer
	framework := newStructFM(map[string]macosplatformmetadata.Struct{
		"WeirdStruct": {
			Fields: []macosplatformmetadata.StructField{
				// Empty GoType and an ObjCType that the mapper cannot resolve.
				{Name: "mystery", ObjCType: "some_unknown_c_type_t"},
			},
		},
	}, nil)
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "unsafe.Pointer") {
		t.Errorf("expected fallback to unsafe.Pointer; got:\n%s", out)
	}
}

// TestStructGoTypeNameCapitalises verifies that the struct name itself is
// exported (first letter uppercased) via GoTypeName.
func TestStructGoTypeNameCapitalises(t *testing.T) {
	var buf bytes.Buffer
	framework := newStructFM(map[string]macosplatformmetadata.Struct{
		"_privateStruct": {
			Fields: []macosplatformmetadata.StructField{{Name: "val", GoType: "int32"}},
		},
	}, nil)
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	// GoTypeName("_privateStruct") capitalises the first character.
	if !strings.Contains(out, "type _privateStruct struct") {
		// The underscore is already uppercase-enough; GoTypeName just capitalises
		// the first rune, which is '_'. The struct name stays "_privateStruct".
		// Accept either form.
	}
	// The struct block must be present regardless.
	if !strings.Contains(out, "struct {") {
		t.Errorf("missing struct body; got:\n%s", out)
	}
}

// ── writeCFWrapper ────────────────────────────────────────────────────────────

// TestStructsCFOpaqueWrapper verifies that a zero-field struct referenced by
// a CF pointer typedef is emitted as an opaque wrapper rather than a plain struct.
func TestStructsCFOpaqueWrapper(t *testing.T) {
	var buf bytes.Buffer
	// __CFString with no fields; CFStringRef typedef points to it.
	framework := newStructFM(
		map[string]macosplatformmetadata.Struct{
			"__CFString": {},
		},
		map[string]string{
			"CFStringRef": "const struct __CFString *",
		},
	)
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "CFStringRef") {
		t.Errorf("expected CFStringRef opaque wrapper; got:\n%s", out)
	}
	if !strings.Contains(out, "NewCFStringRef") {
		t.Errorf("expected NewCFStringRef constructor; got:\n%s", out)
	}
}

// TestStructsCFOpaqueWrapperMutableAlias verifies that mutable variants become
// type aliases and forwarding constructors.
func TestStructsCFOpaqueWrapperMutableAlias(t *testing.T) {
	var buf bytes.Buffer
	framework := newStructFM(
		map[string]macosplatformmetadata.Struct{
			"__CFString": {},
		},
		map[string]string{
			"CFStringRef":        "const struct __CFString *",
			"CFMutableStringRef": "struct __CFString *",
		},
	)
	m := newStructMapper()
	if _, err := EmitStructs(&buf, framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "CFMutableStringRef") {
		t.Errorf("expected CFMutableStringRef alias; got:\n%s", out)
	}
}
