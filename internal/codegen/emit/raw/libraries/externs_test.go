package raw

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// newExternMapper returns a plain Mapper suitable for externs tests.
func newExternMapper() *typemap.Mapper {
	return typemap.New()
}

// newExternFM returns a minimal FrameworkMeta with the given externs slice.
// Framework is fixed to "TestFW".
func newExternFM(externs []macosplatformmetadata.Extern) *macosplatformmetadata.FrameworkMeta {
	return &macosplatformmetadata.FrameworkMeta{
		Framework: "TestFW",
		Externs:   externs,
	}
}

// TestExternsEmpty verifies that an empty externs slice produces no output at
// all (the function returns nil without writing anything).
func TestExternsEmpty(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM(nil)
	m := newExternMapper()
	if err := EmitExterns(&buf, "testfw", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for nil externs, got %q", buf.String())
	}
}

// TestExternsEmptySlice verifies that an explicit but empty externs slice also
// produces no output.
func TestExternsEmptySlice(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{})
	m := newExternMapper()
	if err := EmitExterns(&buf, "testfw", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for empty externs slice, got %q", buf.String())
	}
}

// TestExternsBasicDecl verifies that a single extern produces a var block with
// the extern's Go name and type.
func TestExternsBasicDecl(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{
		{Name: "NSDefaultRunLoopMode", GoType: "string"},
	})
	m := newExternMapper()
	if err := EmitExterns(&buf, "foundation", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "var (") {
		t.Errorf("missing var block; got:\n%s", out)
	}
	if !strings.Contains(out, "NSDefaultRunLoopMode string") {
		t.Errorf("missing extern declaration; got:\n%s", out)
	}
}

// TestExternsFileHeader verifies that the generated file begins with the
// "Code generated" banner, a darwin build tag, and the correct package
// declaration.
func TestExternsFileHeader(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{
		{Name: "SomeExtern", GoType: "string"},
	})
	m := newExternMapper()
	if err := EmitExterns(&buf, "mypkg", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "// Code generated") {
		t.Errorf("missing code-generated banner; got:\n%s", out)
	}
	if !strings.Contains(out, "//go:build darwin") {
		t.Errorf("missing darwin build tag; got:\n%s", out)
	}
	if !strings.Contains(out, "package mypkg") {
		t.Errorf("missing package declaration; got:\n%s", out)
	}
}

// TestExternsUnsafeImport verifies that when an extern resolves to
// "unsafe.Pointer" (because ObjCType is unknown and GoType is unset) the
// output contains an import for "unsafe".
func TestExternsUnsafeImport(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{
		// Neither GoType nor a resolvable ObjCType → fallback to unsafe.Pointer.
		{Name: "SomeOpaqueRef", ObjCType: "UnresolvableCType *"},
	})
	m := newExternMapper()
	if err := EmitExterns(&buf, "testfw", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"unsafe"`) {
		t.Errorf("expected import of unsafe; got:\n%s", out)
	}
	if !strings.Contains(out, "unsafe.Pointer") {
		t.Errorf("expected unsafe.Pointer as type; got:\n%s", out)
	}
}

// TestExternsSortOrder verifies that multiple externs appear in alphabetical
// order by their Name field.
func TestExternsSortOrder(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{
		{Name: "ZebraConst", GoType: "string"},
		{Name: "AlphaConst", GoType: "string"},
		{Name: "MidConst", GoType: "string"},
	})
	m := newExternMapper()
	if err := EmitExterns(&buf, "testfw", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	alphaIdx := strings.Index(out, "AlphaConst")
	midIdx := strings.Index(out, "MidConst")
	zebraIdx := strings.Index(out, "ZebraConst")
	if alphaIdx == -1 || midIdx == -1 || zebraIdx == -1 {
		t.Fatalf("one or more extern names missing:\n%s", out)
	}
	if !(alphaIdx < midIdx && midIdx < zebraIdx) {
		t.Errorf("externs not in alphabetical order: alpha=%d mid=%d zebra=%d",
			alphaIdx, midIdx, zebraIdx)
	}
}

// TestExternsDuplicate verifies that two externs with the same Go name (after
// capitalisation) result in only one declaration — the first one wins.
func TestExternsDuplicate(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{
		// Both names capitalise to "NSDefaultRunLoopMode".
		{Name: "NSDefaultRunLoopMode", GoType: "string"},
		{Name: "NSDefaultRunLoopMode", GoType: "int64"},
	})
	m := newExternMapper()
	if err := EmitExterns(&buf, "testfw", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	count := strings.Count(out, "NSDefaultRunLoopMode")
	if count != 1 {
		t.Errorf("expected exactly 1 occurrence of NSDefaultRunLoopMode, got %d:\n%s", count, out)
	}
}

// TestExternsGoTypePreferred verifies that when GoType is set on an extern it
// is used directly without invoking the mapper's ObjCType resolution.
func TestExternsGoTypePreferred(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{
		{
			Name:     "NSFoundationVersionNumber",
			ObjCType: "double", // mapper would return "float64" for this
			GoType:   "float64",
		},
	})
	m := newExternMapper()
	if err := EmitExterns(&buf, "testfw", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "NSFoundationVersionNumber float64") {
		t.Errorf("expected GoType used directly; got:\n%s", out)
	}
}

// TestExternsAvailabilityComment verifies that MacOSIntroduced on an extern
// produces an "Introduced: macOS X.Y" comment in the var block.
func TestExternsAvailabilityComment(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{
		{
			Name:   "NSURLIsExcludedFromBackupKey",
			GoType: "string",
			Availability: macosplatformmetadata.Availability{
				MacOSIntroduced: "10.8",
			},
		},
	})
	m := newExternMapper()
	if err := EmitExterns(&buf, "testfw", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "// Introduced: macOS 10.8") {
		t.Errorf("missing availability comment; got:\n%s", out)
	}
}

// TestExternsDeprecatedComment verifies that MacOSDeprecated on an extern
// produces the appropriate deprecated comment inside the var block.
func TestExternsDeprecatedComment(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{
		{
			Name:   "OldConstant",
			GoType: "string",
			Availability: macosplatformmetadata.Availability{
				MacOSDeprecated: "11.0",
			},
		},
	})
	m := newExternMapper()
	if err := EmitExterns(&buf, "testfw", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Deprecated: Deprecated in macOS 11.0") {
		t.Errorf("missing deprecated comment; got:\n%s", out)
	}
}

// TestExternsDeprecatedWithReplacement verifies that a deprecated extern with
// a ReplacedBy value emits "use X instead" in the comment.
func TestExternsDeprecatedWithReplacement(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{
		{
			Name:   "OldKey",
			GoType: "string",
			Availability: macosplatformmetadata.Availability{
				MacOSDeprecated: "12.0",
				ReplacedBy:      "NewKey",
			},
		},
	})
	m := newExternMapper()
	if err := EmitExterns(&buf, "testfw", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Use NewKey instead") {
		t.Errorf("missing replacement hint; got:\n%s", out)
	}
}

// TestExternsObjCTypeResolved verifies that when GoType is absent the mapper
// resolves a known ObjC primitive type to its Go equivalent.
func TestExternsObjCTypeResolved(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{
		{Name: "NSIntegerMax", ObjCType: "NSInteger"},
	})
	m := newExternMapper()
	if err := EmitExterns(&buf, "testfw", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	// NSInteger → int64 via primitiveGoType.
	if !strings.Contains(out, "NSIntegerMax int64") {
		t.Errorf("expected NSInteger resolved to int64; got:\n%s", out)
	}
}

// TestExternsVarBlockClosing verifies that the var block is properly closed
// with a ')' and that the output ends with a trailing newline.
func TestExternsVarBlockClosing(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{
		{Name: "SomeConst", GoType: "string"},
	})
	m := newExternMapper()
	if err := EmitExterns(&buf, "testfw", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, ")\n") {
		t.Errorf("var block not properly closed; got:\n%s", out)
	}
}

// TestExternsHeaderBeforeBody verifies that the file header (Code generated,
// build tag, package) appears before the var block in the output.
func TestExternsHeaderBeforeBody(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{
		{Name: "ExternA", GoType: "string"},
	})
	m := newExternMapper()
	if err := EmitExterns(&buf, "mypkg", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	codeGenIdx := strings.Index(out, "// Code generated")
	varIdx := strings.Index(out, "var (")
	if codeGenIdx == -1 || varIdx == -1 {
		t.Fatalf("missing expected sections:\n%s", out)
	}
	if codeGenIdx >= varIdx {
		t.Errorf("file header should precede var block: header=%d var=%d", codeGenIdx, varIdx)
	}
}

// TestExternsDuplicateCaseInsensitive verifies that deduplication operates on
// the Go name (capitalised), so two externs that differ only in casing of the
// first letter are treated as the same declaration.
func TestExternsDuplicateCaseInsensitive(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{
		// "myConst" → GoTypeName → "MyConst"
		// "MyConst" → GoTypeName → "MyConst" (duplicate)
		{Name: "myConst", GoType: "string"},
		{Name: "MyConst", GoType: "int64"},
	})
	m := newExternMapper()
	if err := EmitExterns(&buf, "testfw", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	// After dedup, exactly one "MyConst" should appear in the var block.
	count := strings.Count(out, "MyConst")
	if count != 1 {
		t.Errorf("expected exactly 1 MyConst after dedup, got %d:\n%s", count, out)
	}
}

// TestExternsNoImportWhenTypesResolved verifies that when all externs have
// explicit Go types that are not "unsafe.Pointer", no import block is emitted.
func TestExternsNoImportWhenTypesResolved(t *testing.T) {
	var buf bytes.Buffer
	framework := newExternFM([]macosplatformmetadata.Extern{
		{Name: "KeyA", GoType: "string"},
		{Name: "KeyB", GoType: "int64"},
	})
	m := newExternMapper()
	if err := EmitExterns(&buf, "testfw", framework, m, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "import") {
		t.Errorf("unexpected import statement when no unsafe types present; got:\n%s", out)
	}
}
