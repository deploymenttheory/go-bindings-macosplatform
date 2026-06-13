package ergonomic

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// ergoTestMapper builds a minimal Mapper for ergonomic emitter tests.
func ergoTestMapper() *typemap.Mapper {
	return &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex: map[string]string{
			"NSObject": "Foundation",
			"NSString": "Foundation",
		},
		ModulePrefix:         "github.com/example/fw",
		BlockedImports:       map[string]map[string]bool{},
		TypedefIndex:        map[string]string{},
		StructIndex:         map[string]string{},
		ProtocolIndex:       map[string]string{},
		ProtocolProxyIndex: map[string]string{},
	}
}

// ergoTestFM builds a minimal FrameworkMeta with the given classes map.
func ergoTestFM(fw string, classes map[string]meta.Class) *meta.FrameworkMeta {
	return &meta.FrameworkMeta{
		Framework: fw,
		Classes:   classes,
		Protocols: map[string]meta.Protocol{},
		Enums:     map[string]meta.Enum{},
		Structs:   map[string]meta.Struct{},
		Typedefs:  map[string]string{},
	}
}

// ── NameTracker ───────────────────────────────────────────────────────────────

func TestNameTrackerClaim(t *testing.T) {
	nt := NewNameTracker()
	if !nt.Claim("Foo", "properties") {
		t.Error("first Claim should return true")
	}
	if nt.Claim("Foo", "async") {
		t.Error("second Claim should return false (already taken)")
	}
}

func TestNameTrackerMultipleNames(t *testing.T) {
	nt := NewNameTracker()
	nt.Claim("Foo", "properties")
	nt.Claim("Bar", "properties")
	if !nt.Claim("Baz", "async") {
		t.Error("new name should be claimable")
	}
	if nt.Claim("Foo", "delegates") {
		t.Error("Foo already taken by properties")
	}
}

// ── qualifyWithRawPackage ──────────────────────────────────────────────────────────────

func TestAddRawPrefixCrossFramework(t *testing.T) {
	framework := ergoTestFM("AppKit", map[string]meta.Class{})
	// NSString is a Foundation type — already qualified, no raw. prefix.
	got := qualifyWithRawPackage("*foundation.NSString", framework)
	if got != "*foundation.NSString" {
		t.Errorf("cross-fw type should be unchanged; got %q", got)
	}
}

func TestAddRawPrefixSameFramework(t *testing.T) {
	framework := ergoTestFM("AppKit", map[string]meta.Class{"NSView": {}})
	got := qualifyWithRawPackage("*NSView", framework)
	if got != "*raw.NSView" {
		t.Errorf("same-fw class should get raw. prefix; got %q", got)
	}
}

func TestAddRawPrefixPrimitive(t *testing.T) {
	framework := ergoTestFM("Foundation", map[string]meta.Class{})
	got := qualifyWithRawPackage("int64", framework)
	if got != "int64" {
		t.Errorf("primitive should be unchanged; got %q", got)
	}
}

func TestAddRawPrefixProtocol(t *testing.T) {
	framework := &meta.FrameworkMeta{
		Framework: "AppKit",
		Classes:   map[string]meta.Class{},
		Protocols: map[string]meta.Protocol{"NSTextDelegate": {}},
		Enums:     map[string]meta.Enum{},
		Structs:   map[string]meta.Struct{},
		Typedefs:  map[string]string{},
	}
	got := qualifyWithRawPackage("*NSTextDelegate", framework)
	if got != "*raw.NSTextDelegate" {
		t.Errorf("protocol should get raw. prefix; got %q", got)
	}
}

func TestAddRawPrefixEnum(t *testing.T) {
	framework := &meta.FrameworkMeta{
		Framework: "AppKit",
		Classes:   map[string]meta.Class{},
		Protocols: map[string]meta.Protocol{},
		Enums:     map[string]meta.Enum{"NSComparisonResult": {}},
		Structs:   map[string]meta.Struct{},
		Typedefs:  map[string]string{},
	}
	got := qualifyWithRawPackage("NSComparisonResult", framework)
	if got != "raw.NSComparisonResult" {
		t.Errorf("enum should get raw. prefix; got %q", got)
	}
}

func TestAddRawPrefixCompoundInterface(t *testing.T) {
	framework := &meta.FrameworkMeta{
		Framework: "AppKit",
		Classes:   map[string]meta.Class{},
		Protocols: map[string]meta.Protocol{"P1": {}, "P2": {}},
		Enums:     map[string]meta.Enum{},
		Structs:   map[string]meta.Struct{},
		Typedefs:  map[string]string{},
	}
	got := qualifyWithRawPackage("interface { P1; P2 }", framework)
	if !strings.Contains(got, "raw.P1") {
		t.Errorf("compound interface should prefix protocol names; got %q", got)
	}
}

func TestAddRawPrefixSliceType(t *testing.T) {
	framework := ergoTestFM("AppKit", map[string]meta.Class{"NSView": {}})
	got := qualifyWithRawPackage("[]NSView", framework)
	if got != "[]raw.NSView" {
		t.Errorf("slice type should get raw. prefix; got %q", got)
	}
}

// ── recordOpinionatedImports ─────────────────────────────────────────────────

func TestCollectOpinionatedImportsEmpty(t *testing.T) {
	m := ergoTestMapper()
	usedImports := make(map[string]string)
	recordOpinionatedImports("int64", m, usedImports)
	if len(usedImports) != 0 {
		t.Errorf("primitive type should not add imports; got %v", usedImports)
	}
}

func TestCollectOpinionatedImportsCrossFramework(t *testing.T) {
	m := ergoTestMapper()
	usedImports := make(map[string]string)
	recordOpinionatedImports("*foundation.NSString", m, usedImports)
	if _, ok := usedImports["foundation"]; !ok {
		t.Error("cross-fw type should add import")
	}
}

// ── ErgonomicProperties ───────────────────────────────────────────────────────

func TestErgonomicPropertiesEmpty(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitProperties(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for empty framework; got:\n%s", buf.String())
	}
}

func TestErgonomicPropertiesUnavailableClassSkipped(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{
		"NSFoo": {Availability: meta.Availability{IsUnavailable: true}},
	})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitProperties(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for unavailable class; got:\n%s", buf.String())
	}
}

func TestErgonomicPropertiesSkipsReadonlyProperty(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{
		"NSFoo": {
			Methods: []meta.Method{
				{Selector: "count", Return: meta.ReturnType{ObjCType: "NSUInteger"}},
			},
			Properties: []meta.Property{
				{Name: "count", ObjCType: "NSUInteger", IsReadOnly: true},
			},
		},
	})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitProperties(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "SetCount") {
		t.Errorf("readonly property should not emit setter; got:\n%s", buf.String())
	}
}

func TestErgonomicPropertiesBasicPrimitive(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{
		"NSFoo": {
			Methods: []meta.Method{
				{Selector: "count", Return: meta.ReturnType{ObjCType: "NSUInteger"}},
				{Selector: "setCount:", Params: []meta.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: meta.ReturnType{ObjCType: "void"}},
			},
			Properties: []meta.Property{
				{Name: "count", ObjCType: "NSUInteger"},
			},
		},
	})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitProperties(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "func Count(") {
		t.Errorf("expected Count getter; got:\n%s", out)
	}
	if !strings.Contains(out, "func SetCount(") {
		t.Errorf("expected SetCount setter; got:\n%s", out)
	}
}

func TestErgonomicPropertiesNameTrackerDedup(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{
		"NSFoo": {
			Methods: []meta.Method{
				{Selector: "count", Return: meta.ReturnType{ObjCType: "NSUInteger"}},
				{Selector: "setCount:", Params: []meta.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: meta.ReturnType{ObjCType: "void"}},
			},
			Properties: []meta.Property{
				{Name: "count", ObjCType: "NSUInteger"},
			},
		},
	})
	nt := NewNameTracker()
	nt.Claim("Count", "async") // pre-claim Count to test dedup
	var buf bytes.Buffer
	if err := EmitProperties(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "func Count(") {
		t.Errorf("Count already claimed, should be deduped; got:\n%s", buf.String())
	}
}

// ── ErgonomicConstructors ─────────────────────────────────────────────────────

func TestErgonomicConstructorsEmpty(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitConstructors(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output; got:\n%s", buf.String())
	}
}

func TestErgonomicConstructorsBoolNSError(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{
		"NSFoo": {
			Methods: []meta.Method{
				{
					Selector:   "validate:",
					IsNSError: true,
					Return:     meta.ReturnType{ObjCType: "BOOL"},
				},
			},
		},
	})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitConstructors(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "func Validate(") {
		t.Errorf("expected Validate function; got:\n%s", out)
	}
	if !strings.Contains(out, "error") {
		t.Errorf("expected error return; got:\n%s", out)
	}
}

func TestErgonomicConstructorsNSErrorOut(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{
		"NSFoo": {
			Methods: []meta.Method{
				{
					Selector:   "loadData:",
					IsNSError: true,
					Return:     meta.ReturnType{ObjCType: "NSData *"},
				},
			},
		},
	})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitConstructors(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "func LoadData(") {
		t.Errorf("expected LoadData function; got:\n%s", out)
	}
}

// ── ErgonomicCollections ─────────────────────────────────────────────────────

func TestErgonomicCollectionsEmpty(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitCollections(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output; got:\n%s", buf.String())
	}
}

func TestErgonomicCollectionsNSArray(t *testing.T) {
	m := ergoTestMapper()
	m.OwnerIndex["NSString"] = "Foundation"
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSFoo": {
				Methods: []meta.Method{
					{
						Selector: "items",
						Return:   meta.ReturnType{ObjCType: "NSArray<NSString *> *"},
					},
				},
			},
		},
		Protocols: map[string]meta.Protocol{},
		Enums:     map[string]meta.Enum{},
		Structs:   map[string]meta.Struct{},
		Typedefs:  map[string]string{},
	}
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitCollections(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{"NSString": true}, nt); err != nil {
		t.Fatal(err)
	}
	// NSString is in Foundation (cross-fw from Foundation perspective) but OwnerIndex
	// maps it there — will only emit if NSString is in the same framework.
	// Just verify no error is returned here.
}

func TestErgonomicCollectionsNSMutableArraySkipped(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{
		"NSFoo": {
			Methods: []meta.Method{
				{
					Selector: "items",
					Return:   meta.ReturnType{ObjCType: "NSMutableArray<NSObject *> *"},
				},
			},
		},
	})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitCollections(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "ItemsSlice") {
		t.Errorf("NSMutableArray return should be skipped; got:\n%s", buf.String())
	}
}

// ── ErgonomicAsync ────────────────────────────────────────────────────────────

func TestErgonomicAsyncEmpty(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitAsync(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output; got:\n%s", buf.String())
	}
}

func TestErgonomicAsyncBasic(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{
		"NSFoo": {
			Methods: []meta.Method{
				{
					Selector: "loadWithCompletionHandler:",
					Params: []meta.Param{
						{Name: "completionHandler", ObjCType: "void (^)(NSError *)", IsBlock: true},
					},
					Return: meta.ReturnType{ObjCType: "void"},
				},
			},
		},
	})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitAsync(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "func LoadWithCompletionHandler(") {
		t.Errorf("expected LoadWithCompletionHandler wrapper; got:\n%s", out)
	}
	if !strings.Contains(out, "error") {
		t.Errorf("expected error return; got:\n%s", out)
	}
}

// ── ErgonomicMainThread ───────────────────────────────────────────────────────

func TestErgonomicMainThreadEmpty(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("AppKit", map[string]meta.Class{})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitMainThread(&buf, "appkit", "github.com/example/fw/appkit", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output; got:\n%s", buf.String())
	}
}

func TestErgonomicMainThreadBasic(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("AppKit", map[string]meta.Class{
		"NSView": {
			Methods: []meta.Method{
				{
					Selector:           "setNeedsDisplay:",
					IsMainThreadRequired: true,
					Params:               []meta.Param{{Name: "flag", ObjCType: "BOOL"}},
					Return:             meta.ReturnType{ObjCType: "void"},
				},
			},
		},
	})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitMainThread(&buf, "appkit", "github.com/example/fw/appkit", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "RunOnMainThread") {
		t.Errorf("expected RunOnMainThread dispatch; got:\n%s", out)
	}
	if !strings.Contains(out, "func SetNeedsDisplay(") {
		t.Errorf("expected SetNeedsDisplay function; got:\n%s", out)
	}
}

func TestErgonomicMainThreadNameTrackerDedup(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("AppKit", map[string]meta.Class{
		"NSView": {
			Methods: []meta.Method{
				{
					Selector:           "setNeedsDisplay:",
					IsMainThreadRequired: true,
					Params:               []meta.Param{{Name: "flag", ObjCType: "BOOL"}},
					Return:             meta.ReturnType{ObjCType: "void"},
				},
			},
		},
	})
	nt := NewNameTracker()
	nt.Claim("SetNeedsDisplay", "properties") // pre-claim
	var buf bytes.Buffer
	if err := EmitMainThread(&buf, "appkit", "github.com/example/fw/appkit", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "func SetNeedsDisplay(") {
		t.Errorf("already claimed function should be deduped; got:\n%s", buf.String())
	}
}

// ── ErgonomicDelegates ────────────────────────────────────────────────────────

func TestErgonomicDelegatesEmpty(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("AppKit", map[string]meta.Class{})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitDelegates(&buf, "appkit", "github.com/example/fw/appkit", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output; got:\n%s", buf.String())
	}
}

func TestErgonomicDelegatesBasic(t *testing.T) {
	m := ergoTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "AppKit",
		Classes: map[string]meta.Class{
			"NSTextField": {
				Properties: []meta.Property{
					{Name: "delegate", ObjCType: "id<NSTextFieldDelegate>"},
				},
			},
		},
		Protocols: map[string]meta.Protocol{
			"NSTextFieldDelegate": {
				Methods: []meta.Method{
					{Selector: "textDidChange:", IsOptional: true, Return: meta.ReturnType{ObjCType: "void"}},
					{Selector: "textDidBeginEditing:", IsOptional: false, Return: meta.ReturnType{ObjCType: "void"}},
				},
			},
		},
		Enums:    map[string]meta.Enum{},
		Structs:  map[string]meta.Struct{},
		Typedefs: map[string]string{},
	}
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitDelegates(&buf, "appkit", "github.com/example/fw/appkit", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "NSTextFieldDelegate") {
		t.Errorf("expected NSTextFieldDelegate interface; got:\n%s", out)
	}
	if !strings.Contains(out, "DefaultNSTextFieldDelegate") {
		t.Errorf("expected DefaultNSTextFieldDelegate struct; got:\n%s", out)
	}
	if !strings.Contains(out, "SetNSTextFieldDelegate") {
		t.Errorf("expected SetNSTextFieldDelegate function; got:\n%s", out)
	}
}

func TestErgonomicDelegatesNoOptionalMethodsNoDefault(t *testing.T) {
	m := ergoTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "AppKit",
		Classes: map[string]meta.Class{
			"NSTextField": {
				Properties: []meta.Property{
					{Name: "delegate", ObjCType: "id<NSTextFieldDelegate>"},
				},
			},
		},
		Protocols: map[string]meta.Protocol{
			"NSTextFieldDelegate": {
				Methods: []meta.Method{
					// only required methods → no Default struct
					{Selector: "textDidChange:", IsOptional: false, Return: meta.ReturnType{ObjCType: "void"}},
				},
			},
		},
		Enums:    map[string]meta.Enum{},
		Structs:  map[string]meta.Struct{},
		Typedefs: map[string]string{},
	}
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitDelegates(&buf, "appkit", "github.com/example/fw/appkit", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "DefaultNSTextFieldDelegate") {
		t.Errorf("no optional methods → no Default struct; got:\n%s", buf.String())
	}
}

// ── shared: sortedKeys ────────────────────────────────────────────────────────

func TestSortedKeys(t *testing.T) {
	m := map[string]meta.Class{"C": {}, "A": {}, "B": {}}
	got := sortedKeys(m)
	if len(got) != 3 {
		t.Fatalf("expected 3 keys; got %d", len(got))
	}
	if got[0] != "A" || got[1] != "B" || got[2] != "C" {
		t.Errorf("expected sorted [A B C]; got %v", got)
	}
}

// ── writeErgonomicHeader ──────────────────────────────────────────────────────

func TestWriteErgonomicHeader(t *testing.T) {
	var buf bytes.Buffer
	writeErgonomicHeader(&buf, "appkit", "github.com/example/fw/appkit", nil, false)
	out := buf.String()
	if !strings.Contains(out, "package appkit") {
		t.Errorf("expected package declaration; got:\n%s", out)
	}
	if !strings.Contains(out, "Code generated") {
		t.Errorf("expected generated header; got:\n%s", out)
	}
	if !strings.Contains(out, "context") {
		t.Errorf("expected context import; got:\n%s", out)
	}
}

func TestWriteErgonomicHeaderWithContextImport(t *testing.T) {
	var buf bytes.Buffer
	writeErgonomicHeader(&buf, "appkit", "github.com/example/fw/appkit", map[string]string{
		"foundation": "github.com/example/fw/foundation",
	}, true)
	out := buf.String()
	if !strings.Contains(out, "foundation") {
		t.Errorf("expected foundation import; got:\n%s", out)
	}
	if !strings.Contains(out, "objc") {
		t.Errorf("expected objc import when needsObjc=true; got:\n%s", out)
	}
}

// ── buildZeroValue ─────────────────────────────────────────────────────────────────

func TestZeroValuePointer(t *testing.T) {
	if got := buildZeroValue("*NSObject"); got != "nil" {
		t.Errorf("buildZeroValue(*NSObject) = %q; want nil", got)
	}
}

func TestZeroValueBool(t *testing.T) {
	if got := buildZeroValue("bool"); got != "false" {
		t.Errorf("buildZeroValue(bool) = %q; want false", got)
	}
}

func TestZeroValueString(t *testing.T) {
	if got := buildZeroValue("string"); got != `""` {
		t.Errorf(`buildZeroValue(string) = %q; want ""`, got)
	}
}

func TestZeroValueError(t *testing.T) {
	if got := buildZeroValue("error"); got != "nil" {
		t.Errorf("buildZeroValue(error) = %q; want nil", got)
	}
}

func TestZeroValueInt(t *testing.T) {
	if got := buildZeroValue("int64"); got != "0" {
		t.Errorf("buildZeroValue(int64) = %q; want 0", got)
	}
}

func TestZeroValueSlice(t *testing.T) {
	if got := buildZeroValue("[]byte"); got != "nil" {
		t.Errorf("buildZeroValue([]byte) = %q; want nil", got)
	}
}

func TestZeroValueQualified(t *testing.T) {
	if got := buildZeroValue("foundation.NSObject"); got != "nil" {
		t.Errorf("buildZeroValue(foundation.NSObject) = %q; want nil", got)
	}
}

// ── extractBlockParams ────────────────────────────────────────────────────────

func TestExtractBlockParamsBasic(t *testing.T) {
	got := extractBlockParams("void (^)(NSData *, NSError *)")
	if got != "NSData *, NSError *" {
		t.Errorf("extractBlockParams = %q; want %q", got, "NSData *, NSError *")
	}
}

func TestExtractBlockParamsVoid(t *testing.T) {
	got := extractBlockParams("void (^)(void)")
	if got != "void" {
		t.Errorf("extractBlockParams(void) = %q; want void", got)
	}
}

func TestExtractBlockParamsNotABlock(t *testing.T) {
	got := extractBlockParams("NSString *")
	if got != "" {
		t.Errorf("extractBlockParams(non-block) = %q; want empty", got)
	}
}

// ── ErgonomicBlockEnum ────────────────────────────────────────────────────────

// TestErgonomicBlockEnumEmpty verifies no output when there are no block
// enumeration methods.
func TestErgonomicBlockEnumEmpty(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{
		"NSArray": {Methods: []meta.Method{
			{Selector: "count", Return: meta.ReturnType{ObjCType: "NSUInteger"}},
		}},
	})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitBlockEnum(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for non-enumeration methods; got:\n%s", buf.String())
	}
}

// TestErgonomicBlockEnumBasic verifies output for an enumerate-using-block method.
func TestErgonomicBlockEnumBasic(t *testing.T) {
	m := ergoTestMapper()
	// Method that classifies as BlockEnumeration: selector contains "enumerate",
	// last arg is a block with "BOOL *".
	// ObjC type strings don't include parameter names.
	framework := ergoTestFM("Foundation", map[string]meta.Class{
		"NSArray": {
			Methods: []meta.Method{
				{
					Selector: "enumerateObjectsUsingBlock:",
					Params: []meta.Param{
						{
							Name:     "block",
							ObjCType: "void (^)(id, NSUInteger, BOOL *)",
							IsBlock:  true,
						},
					},
					Return: meta.ReturnType{ObjCType: "void"},
				},
			},
		},
	})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitBlockEnum(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{"NSArray": true}, nt); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// naming.MethodName strips "Block" suffix from selectors.
	if !strings.Contains(out, "EnumerateObjectsUsing") {
		t.Errorf("expected EnumerateObjectsUsing wrapper; got:\n%s", out)
	}
}

// TestParseEnumBlockParamsBasic verifies that BOOL * is excluded from enum args.
// ObjC block type strings don't include variable names.
func TestParseEnumBlockParamsBasic(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{})
	ctx := m.BaseContext("Foundation", map[string]bool{})
	params := extractEnumBlockParams("void (^)(id, NSUInteger, BOOL *)", ctx, m, framework, nil)
	// Should have 2 params (id and NSUInteger), BOOL * excluded.
	if len(params) != 2 {
		t.Errorf("expected 2 enum params (BOOL * excluded); got %d: %+v", len(params), params)
	}
}

func TestParseEnumBlockParamsVoid(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{})
	ctx := m.BaseContext("Foundation", map[string]bool{})
	params := extractEnumBlockParams("void (^)(void)", ctx, m, framework, nil)
	if len(params) != 0 {
		t.Errorf("expected 0 params for void block; got %d", len(params))
	}
}

// ── buildDelegateMethodGoSig / extractDelegateReturnParts ──────────────────────────────

func TestDelegateMethodGoSigWithReturnType(t *testing.T) {
	m := ergoTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "AppKit",
		Classes:   map[string]meta.Class{"NSFoo": {}},
		Protocols: map[string]meta.Protocol{},
		Enums:     map[string]meta.Enum{},
		Structs:   map[string]meta.Struct{},
		Typedefs:  map[string]string{},
	}
	ctx := m.BaseContext("AppKit", map[string]bool{})
	pm := meta.Method{
		Selector:   "windowShouldClose:",
		Params:       []meta.Param{{Name: "sender", ObjCType: "id"}},
		Return:     meta.ReturnType{ObjCType: "BOOL"},
		IsOptional: true,
	}
	sig := buildDelegateMethodGoSig(pm, ctx, m, framework, nil)
	if !strings.Contains(sig, "WindowShouldClose") {
		t.Errorf("expected WindowShouldClose in sig; got %q", sig)
	}
}

func TestDelegateMethodRetPartsNSError(t *testing.T) {
	m := ergoTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "AppKit",
		Classes:   map[string]meta.Class{},
		Protocols: map[string]meta.Protocol{},
		Enums:     map[string]meta.Enum{},
		Structs:   map[string]meta.Struct{},
		Typedefs:  map[string]string{},
	}
	ctx := m.BaseContext("AppKit", map[string]bool{})
	pm := meta.Method{
		Selector:   "doThing:",
		IsNSError: true,
		Return:     meta.ReturnType{ObjCType: "BOOL"},
		IsOptional: true,
	}
	parts := extractDelegateReturnParts(pm, ctx, m, framework, nil)
	// Should have error in parts
	found := false
	for _, p := range parts {
		if p == "error" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error in return parts; got %v", parts)
	}
}

// TestErgonomicDelegatesWithReturnType covers the optionalMethod codepath where
// a non-void return makes DefaultXxx use var declarations.
func TestErgonomicDelegatesWithReturnType(t *testing.T) {
	m := ergoTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "AppKit",
		Classes: map[string]meta.Class{
			"NSWindow": {
				Properties: []meta.Property{
					{Name: "delegate", ObjCType: "id<NSWindowDelegate>"},
				},
			},
		},
		Protocols: map[string]meta.Protocol{
			"NSWindowDelegate": {
				Methods: []meta.Method{
					{Selector: "windowShouldClose:", IsOptional: true, Return: meta.ReturnType{ObjCType: "BOOL"},
						Params: []meta.Param{{Name: "sender", ObjCType: "id"}}},
				},
			},
		},
		Enums:    map[string]meta.Enum{},
		Structs:  map[string]meta.Struct{},
		Typedefs: map[string]string{},
	}
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitDelegates(&buf, "appkit", "github.com/example/fw/appkit", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "DefaultNSWindowDelegate") {
		t.Errorf("expected DefaultNSWindowDelegate; got:\n%s", out)
	}
}

// ── buildRawReceiverType ────────────────────────────────────────────────────────────

func TestRawReceiverTypeGeneric(t *testing.T) {
	m := ergoTestMapper()
	m.GenericClasses["NSArray"] = true
	got := buildRawReceiverType("NSArray", m)
	if got != "raw.NSArray[objc.Object]" {
		t.Errorf("buildRawReceiverType(NSArray) = %q; want raw.NSArray[objc.Object]", got)
	}
}

func TestRawReceiverTypeNonGeneric(t *testing.T) {
	m := ergoTestMapper()
	got := buildRawReceiverType("NSView", m)
	if got != "raw.NSView" {
		t.Errorf("buildRawReceiverType(NSView) = %q; want raw.NSView", got)
	}
}

// ── ErgonomicCollections (with actual output) ─────────────────────────────────

func TestErgonomicCollectionsWithSameFWElement(t *testing.T) {
	m := ergoTestMapper()
	// VZFoo is in the Virtualization framework; a method returning NSArray<VZFoo *> *
	// should produce a typed collection wrapper.
	framework := &meta.FrameworkMeta{
		Framework: "Virtualization",
		Classes: map[string]meta.Class{
			"VZFoo": {},
			"VZConfig": {
				Methods: []meta.Method{
					// Instance method returning NSArray<VZFoo *> * → CollectionReturn tag.
					{Selector: "items", Return: meta.ReturnType{ObjCType: "NSArray<VZFoo *> *"}},
				},
			},
		},
		Protocols: map[string]meta.Protocol{},
		Enums:     map[string]meta.Enum{},
		Structs:   map[string]meta.Struct{},
		Typedefs:  map[string]string{},
	}
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitCollections(&buf, "virtualization", "github.com/example/fw/virtualization", framework, m, map[string]bool{"VZFoo": true, "VZConfig": true}, nt); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Items") {
		t.Errorf("expected Items accessor; got:\n%s", out)
	}
}

// ── ErgonomicMainThread with return type ──────────────────────────────────────

func TestErgonomicMainThreadWithReturn(t *testing.T) {
	m := ergoTestMapper()
	m.OwnerIndex["NSString"] = "Foundation"
	framework := &meta.FrameworkMeta{
		Framework: "AppKit",
		Classes: map[string]meta.Class{
			"NSView": {
				Methods: []meta.Method{
					{
						Selector:           "title",
						Return:             meta.ReturnType{ObjCType: "NSUInteger"},
						IsMainThreadRequired: true,
					},
				},
			},
		},
		Protocols: map[string]meta.Protocol{},
		Enums:     map[string]meta.Enum{},
		Structs:   map[string]meta.Struct{},
		Typedefs:  map[string]string{},
	}
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitMainThread(&buf, "appkit", "github.com/example/fw/appkit", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Title") {
		t.Errorf("expected Title wrapper; got:\n%s", out)
	}
}

// ── ErgonomicAsync with data results ─────────────────────────────────────────

func TestErgonomicAsyncWithDataResult(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{
		"NSURLSession": {
			Methods: []meta.Method{
				{
					Selector: "dataTaskWithURL:completionHandler:",
					Params: []meta.Param{
						{Name: "url", ObjCType: "NSURL *"},
						{Name: "completionHandler", ObjCType: "void (^)(NSData *, NSURLResponse *, NSError *)", IsBlock: true},
					},
					Return: meta.ReturnType{ObjCType: "void"},
				},
			},
		},
	})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitAsync(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	// Even if the classify test doesn't find AsyncCompletion, no error expected.
	_ = buf.String()
}

// ── extractFuncParams ─────────────────────────────────────────────────────────

func TestExtractFuncParamsBasic(t *testing.T) {
	params := extractFuncParams("func(*NSData, *NSURL, error)")
	if len(params) != 3 {
		t.Errorf("expected 3 params; got %d: %v", len(params), params)
	}
}

func TestExtractFuncParamsEmpty(t *testing.T) {
	params := extractFuncParams("func()")
	if len(params) != 0 {
		t.Errorf("expected 0 params for func(); got %d: %v", len(params), params)
	}
}

func TestExtractFuncParamsNotAFunc(t *testing.T) {
	params := extractFuncParams("*NSString")
	if len(params) != 0 {
		t.Errorf("non-func should return nil; got %d: %v", len(params), params)
	}
}

func TestExtractFuncParamsNested(t *testing.T) {
	// Nested generic params should be handled correctly
	params := extractFuncParams("func(*NSArray[objc.Object], error)")
	if len(params) != 2 {
		t.Errorf("expected 2 params with generic; got %d: %v", len(params), params)
	}
}

// ── ErgonomicCollections additional paths ─────────────────────────────────────

func TestErgonomicCollectionsMutableArraySkipped(t *testing.T) {
	m := ergoTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSFoo": {},
			"NSBar": {
				Methods: []meta.Method{
					// NSMutableArray return should be skipped
					{Selector: "items", Return: meta.ReturnType{ObjCType: "NSMutableArray<NSFoo *> *"}},
				},
			},
		},
		Protocols: map[string]meta.Protocol{},
		Enums:     map[string]meta.Enum{},
		Structs:   map[string]meta.Struct{},
		Typedefs:  map[string]string{},
	}
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitCollections(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{"NSFoo": true, "NSBar": true}, nt); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("NSMutableArray return should be skipped; got:\n%s", buf.String())
	}
}

func TestErgonomicCollectionsWithErrorMethod(t *testing.T) {
	m := ergoTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Virtualization",
		Classes: map[string]meta.Class{
			"VZDisk": {},
			"VZConfig": {
				Methods: []meta.Method{
					{
						Selector:   "disksWithError:",
						IsNSError: true,
						Return:     meta.ReturnType{ObjCType: "NSArray<VZDisk *> *"},
					},
				},
			},
		},
		Protocols: map[string]meta.Protocol{},
		Enums:     map[string]meta.Enum{},
		Structs:   map[string]meta.Struct{},
		Typedefs:  map[string]string{},
	}
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitCollections(&buf, "virtualization", "github.com/example/fw/virtualization", framework, m, map[string]bool{"VZDisk": true, "VZConfig": true}, nt); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "DisksWithError") {
		t.Errorf("expected DisksWithError wrapper; got:\n%s", out)
	}
	if !strings.Contains(out, "error") {
		t.Errorf("expected error return in wrapper; got:\n%s", out)
	}
}

// ── ErgonomicAsync additional cases ──────────────────────────────────────────

func TestErgonomicAsyncVoidBlock(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{
		"NSFoo": {
			Methods: []meta.Method{
				{
					Selector: "saveWithCompletionHandler:",
					Params: []meta.Param{
						{Name: "completionHandler", ObjCType: "void (^)(NSError *)", IsBlock: true},
					},
					Return: meta.ReturnType{ObjCType: "void"},
				},
			},
		},
	})
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitAsync(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{"NSFoo": true}, nt); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Save") {
		t.Errorf("expected Save async wrapper; got:\n%s", out)
	}
}

// ── buildZeroValue additional cases ────────────────────────────────────────────────

func TestZeroValueQualifiedType(t *testing.T) {
	// Package-qualified non-pointer type → nil
	got := buildZeroValue("objc.Object")
	if got != "nil" {
		t.Errorf("buildZeroValue(objc.Object) = %q; want nil", got)
	}
}

func TestZeroValueFuncType(t *testing.T) {
	got := buildZeroValue("func(error)")
	if got != "nil" {
		t.Errorf("buildZeroValue(func type) = %q; want nil", got)
	}
}

func TestZeroValueInterfaceType(t *testing.T) {
	got := buildZeroValue("interface { Foo() }")
	if got != "nil" {
		t.Errorf("buildZeroValue(interface) = %q; want nil", got)
	}
}

func TestZeroValueUnsafePointer(t *testing.T) {
	got := buildZeroValue("unsafe.Pointer")
	if got != "nil" {
		t.Errorf("buildZeroValue(unsafe.Pointer) = %q; want nil", got)
	}
}

// ── ErgonomicMainThread with class method excluded ────────────────────────────

func TestErgonomicMainThreadClassMethodExcluded(t *testing.T) {
	m := ergoTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "AppKit",
		Classes: map[string]meta.Class{
			"NSApp": {
				Methods: []meta.Method{
					{
						Selector:           "run",
						IsClassMethod:      true,
						Return:             meta.ReturnType{ObjCType: "void"},
						IsMainThreadRequired: true,
					},
				},
			},
		},
		Protocols: map[string]meta.Protocol{},
		Enums:     map[string]meta.Enum{},
		Structs:   map[string]meta.Struct{},
		Typedefs:  map[string]string{},
	}
	nt := NewNameTracker()
	var buf bytes.Buffer
	if err := EmitMainThread(&buf, "appkit", "github.com/example/fw/appkit", framework, m, map[string]bool{}, nt); err != nil {
		t.Fatal(err)
	}
	// Class methods are excluded from ergonomic main-thread wrappers.
	if buf.Len() != 0 {
		t.Errorf("class method should be excluded from main-thread wrappers; got:\n%s", buf.String())
	}
}

// ── isObjcObjectType ──────────────────────────────────────────────────────────

func TestIsObjcObjectTypePointer(t *testing.T) {
	if !isObjcObjectType("*NSView") {
		t.Error("pointer type should be an ObjC object type")
	}
}

func TestIsObjcObjectTypeObjcObject(t *testing.T) {
	if !isObjcObjectType("objc.Object") {
		t.Error("objc.Object should be an ObjC object type")
	}
}

func TestIsObjcObjectTypeScalar(t *testing.T) {
	if isObjcObjectType("uint64") {
		t.Error("scalar should NOT be an ObjC object type")
	}
}

func TestIsObjcObjectTypeFunc(t *testing.T) {
	if isObjcObjectType("func(error)") {
		t.Error("func type should NOT be an ObjC object type")
	}
}

// ── looksLikeNSArray ──────────────────────────────────────────────────────────

func TestLooksLikeNSArrayTrue(t *testing.T) {
	if !looksLikeNSArray("NSArray<NSString *> *") {
		t.Error("NSArray type should look like NSArray")
	}
}

func TestLooksLikeNSArrayFalse(t *testing.T) {
	if looksLikeNSArray("NSDictionary *") {
		t.Error("NSDictionary should NOT look like NSArray")
	}
}

func TestLooksLikeNSArrayBlockTypeExcluded(t *testing.T) {
	if looksLikeNSArray("NSArray<Foo *> *(^)(void)") {
		t.Error("block type referencing NSArray should be excluded")
	}
}

// ── extractNSArrayElementGoType ───────────────────────────────────────────────

func TestExtractNSArrayElementGoTypeBasic(t *testing.T) {
	m := ergoTestMapper()
	m.OwnerIndex["NSString"] = "Foundation"
	framework := ergoTestFM("Foundation", map[string]meta.Class{"NSString": {}})
	ctx := m.BaseContext("Foundation", map[string]bool{"NSString": true})
	elemType := extractNSArrayElementGoType("NSArray<NSString *> *", ctx, m, framework, nil)
	if !strings.Contains(elemType, "NSString") {
		t.Errorf("expected NSString element type; got %q", elemType)
	}
}

func TestExtractNSArrayElementGoTypeNonArray(t *testing.T) {
	m := ergoTestMapper()
	framework := ergoTestFM("Foundation", map[string]meta.Class{})
	ctx := m.BaseContext("Foundation", map[string]bool{})
	elemType := extractNSArrayElementGoType("NSDictionary *", ctx, m, framework, nil)
	if elemType != "" {
		t.Errorf("non-array should return empty string; got %q", elemType)
	}
}
