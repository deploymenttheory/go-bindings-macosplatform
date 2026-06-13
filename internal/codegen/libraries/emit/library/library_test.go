package library

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// libTestMapper builds a minimal Mapper for library emitter tests.
func libTestMapper() *typemap.Mapper {
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

// libTestFM builds a FrameworkMeta with the given classes.
func libTestFM(fw string, classes map[string]macosplatformmetadata.Class) *macosplatformmetadata.FrameworkMeta {
	return &macosplatformmetadata.FrameworkMeta{
		Framework: fw,
		Classes:   classes,
		Protocols: map[string]macosplatformmetadata.Protocol{},
		Enums:     map[string]macosplatformmetadata.Enum{},
		Structs:   map[string]macosplatformmetadata.Struct{},
		Typedefs:  map[string]string{},
	}
}

// ── completionHandlerIndex ─────────────────────────────────────────────────────

func TestCompletionHandlerIndexFound(t *testing.T) {
	args := []macosplatformmetadata.Param{
		{Name: "options", ObjCType: "NSDictionary *"},
		{Name: "handler", ObjCType: "void (^)(NSError *)", IsBlock: true},
	}
	if got := completionHandlerIndex(args); got != 1 {
		t.Errorf("expected index 1; got %d", got)
	}
}

func TestCompletionHandlerIndexNotFound(t *testing.T) {
	args := []macosplatformmetadata.Param{
		{Name: "name", ObjCType: "NSString *"},
	}
	if got := completionHandlerIndex(args); got != -1 {
		t.Errorf("expected -1; got %d", got)
	}
}

func TestCompletionHandlerIndexMultiParamBlockExcluded(t *testing.T) {
	// void (^)(NSData *, NSError *) has 2 params → not a simple completion handler.
	args := []macosplatformmetadata.Param{
		{Name: "handler", ObjCType: "void (^)(NSData *, NSError *)", IsBlock: true},
	}
	if got := completionHandlerIndex(args); got != -1 {
		t.Errorf("multi-param block should not be a completion handler; got %d", got)
	}
}

func TestCompletionHandlerIndexNonBlockSkipped(t *testing.T) {
	args := []macosplatformmetadata.Param{
		{Name: "name", ObjCType: "NSString *"},
		{Name: "x", ObjCType: "NSInteger"},
	}
	if got := completionHandlerIndex(args); got != -1 {
		t.Errorf("no block arg → -1; got %d", got)
	}
}

// ── removeCompletionSuffix ─────────────────────────────────────────────────────

func TestStripCompletionSuffixWithHandler(t *testing.T) {
	got := removeCompletionSuffix("LoadWithCompletionHandler")
	if got != "Load" {
		t.Errorf("removeCompletionSuffix = %q; want %q", got, "Load")
	}
}

func TestStripCompletionSuffixNoWith(t *testing.T) {
	got := removeCompletionSuffix("InstallCompletionHandler")
	if got != "Install" {
		t.Errorf("removeCompletionSuffix = %q; want %q", got, "Install")
	}
}

func TestStripCompletionSuffixNoSuffix(t *testing.T) {
	got := removeCompletionSuffix("DoSomething")
	if got != "" {
		t.Errorf("no CompletionHandler suffix → empty; got %q", got)
	}
}

func TestStripCompletionSuffixJustHandler(t *testing.T) {
	// "CompletionHandler" alone → falls back to the original name (safety).
	got := removeCompletionSuffix("CompletionHandler")
	if got == "" {
		t.Errorf("safety fallback: should not return empty for just 'CompletionHandler'")
	}
}

// ── extractNSArrayElementType ───────────────────────────────────────────────────

func TestParseNSArrayElementTypeNSArray(t *testing.T) {
	elem, ok := extractNSArrayElementType("NSArray<NSString *> *")
	if !ok || elem != "NSString" {
		t.Errorf("extractNSArrayElementType = (%q, %v); want (\"NSString\", true)", elem, ok)
	}
}

func TestParseNSArrayElementTypeNSMutableArray(t *testing.T) {
	elem, ok := extractNSArrayElementType("NSMutableArray<NSView *> *")
	if !ok || elem != "NSView" {
		t.Errorf("extractNSArrayElementType = (%q, %v); want (\"NSView\", true)", elem, ok)
	}
}

func TestParseNSArrayElementTypeNotArray(t *testing.T) {
	_, ok := extractNSArrayElementType("NSDictionary<NSString *, id> *")
	if ok {
		t.Error("NSDictionary should not parse as NSArray")
	}
}

func TestParseNSArrayElementTypeIDArray(t *testing.T) {
	_, ok := extractNSArrayElementType("NSArray<id> *")
	if ok {
		t.Error("id-typed array should return false")
	}
}

func TestParseNSArrayElementTypeBlockType(t *testing.T) {
	// Block type containing "(^" should be excluded.
	_, ok := extractNSArrayElementType("NSArray<Foo *> *(^)(void)")
	if ok {
		t.Error("block type should return false")
	}
}

func TestParseNSArrayElementTypeNullable(t *testing.T) {
	elem, ok := extractNSArrayElementType("NSArray<NSString *> * _Nullable")
	if !ok || elem != "NSString" {
		t.Errorf("extractNSArrayElementType with _Nullable = (%q, %v); want (\"NSString\", true)", elem, ok)
	}
}

// ── propertyGoName ──────────────────────────────────────────────────────────────

func TestPropToGoNameCapitalises(t *testing.T) {
	cases := []struct{ in, want string }{
		{"count", "Count"},
		{"audioDevices", "AudioDevices"},
		{"", ""},
		{"X", "X"},
	}
	for _, c := range cases {
		if got := propertyGoName(c.in); got != c.want {
			t.Errorf("propertyGoName(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

// ── classHasValidateMethod ────────────────────────────────────────────────────

func TestClassHasValidateMethod(t *testing.T) {
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{Selector: "validateWithError:", Return: macosplatformmetadata.ReturnType{ObjCType: "BOOL"}},
		},
	}
	if !classHasValidateMethod(cls) {
		t.Error("expected classHasValidateMethod=true")
	}
}

func TestClassHasValidateMethodFalse(t *testing.T) {
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{Selector: "doThing", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
		},
	}
	if classHasValidateMethod(cls) {
		t.Error("expected classHasValidateMethod=false")
	}
}

func TestClassHasValidateMethodUnavailableExcluded(t *testing.T) {
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{Selector: "validateWithError:", Availability: macosplatformmetadata.Availability{IsUnavailable: true}},
		},
	}
	if classHasValidateMethod(cls) {
		t.Error("unavailable validate method should not count")
	}
}

// ── classHasInstanceSetter ────────────────────────────────────────────────────

func TestClassHasInstanceSetter(t *testing.T) {
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{Selector: "setCount:", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
		},
	}
	if !classHasInstanceSetter(cls, "count") {
		t.Error("expected classHasInstanceSetter=true")
	}
}

func TestClassHasInstanceSetterClassMethodExcluded(t *testing.T) {
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{Selector: "setCount:", IsClassMethod: true, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
		},
	}
	if classHasInstanceSetter(cls, "count") {
		t.Error("class-method setter should not count")
	}
}

func TestClassHasInstanceSetterEmptyPropName(t *testing.T) {
	cls := macosplatformmetadata.Class{}
	if classHasInstanceSetter(cls, "") {
		t.Error("empty propName should return false")
	}
}

// ── classGetterIsClassMethod ──────────────────────────────────────────────────

func TestClassGetterIsClassMethod(t *testing.T) {
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{Selector: "sharedInstance", IsClassMethod: true},
		},
	}
	if !classGetterIsClassMethod(cls, "sharedInstance") {
		t.Error("expected true for class-method getter")
	}
}

func TestClassGetterIsClassMethodFalse(t *testing.T) {
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{Selector: "count", IsClassMethod: false},
		},
	}
	if classGetterIsClassMethod(cls, "count") {
		t.Error("instance getter should return false")
	}
}

// ── extractObjCBaseTypeName ──────────────────────────────────────────────────────────

func TestObjcBaseTypeName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"CGRect", "CGRect"},
		{"CGRect _Nullable", "CGRect"},
		{"NSString * _Nonnull", "NSString"},
		{"NSArray<NSString *> *", "NSArray"},
		{"const char *", "char"},
		{"__kindof NSView *", "NSView"},
	}
	for _, c := range cases {
		if got := extractObjCBaseTypeName(c.in); got != c.want {
			t.Errorf("extractObjCBaseTypeName(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

// ── qualifyWithRawPackage (library shared.go) ─────────────────────────────────────────

func TestLibraryAddRawPrefixClass(t *testing.T) {
	framework := libTestFM("AppKit", map[string]macosplatformmetadata.Class{"NSView": {}})
	got := qualifyWithRawPackage("*NSView", framework)
	if got != "*raw.NSView" {
		t.Errorf("qualifyWithRawPackage = %q; want \"*raw.NSView\"", got)
	}
}

func TestLibraryAddRawPrefixCrossFramework(t *testing.T) {
	framework := libTestFM("AppKit", map[string]macosplatformmetadata.Class{})
	got := qualifyWithRawPackage("*foundation.NSString", framework)
	if got != "*foundation.NSString" {
		t.Errorf("cross-fw type should be unchanged; got %q", got)
	}
}

// ── writeOpinionatedHeader ────────────────────────────────────────────────────

func TestWriteOpinionatedHeader(t *testing.T) {
	var buf bytes.Buffer
	writeOpinionatedHeader(&buf, "virtualization", "github.com/example/fw/virtualization", nil, nil, false)
	out := buf.String()
	if !strings.Contains(out, "package virtualization") {
		t.Errorf("expected package declaration; got:\n%s", out)
	}
	if !strings.Contains(out, "context") {
		t.Errorf("expected context import; got:\n%s", out)
	}
	if !strings.Contains(out, "Code generated") {
		t.Errorf("expected generated header; got:\n%s", out)
	}
}

func TestWriteOpinionatedHeaderNeedsObjc(t *testing.T) {
	var buf bytes.Buffer
	writeOpinionatedHeader(&buf, "virtualization", "github.com/example/fw/virtualization", nil, nil, true)
	out := buf.String()
	if !strings.Contains(out, "cgo") {
		t.Errorf("expected objc import when needsObjc=true; got:\n%s", out)
	}
}

func TestWriteOpinionatedHeaderWithExtraImport(t *testing.T) {
	var buf bytes.Buffer
	extraImports := map[string]string{
		"foundation": "github.com/example/fw/frameworks/foundation",
	}
	writeOpinionatedHeader(&buf, "mylib", "github.com/example/fw/mylib", extraImports, nil, false)
	out := buf.String()
	if !strings.Contains(out, "foundation") {
		t.Errorf("expected foundation import; got:\n%s", out)
	}
}

// ── Async ──────────────────────────────────────────────────────────────────────

func TestAsyncEmpty(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Foundation", map[string]macosplatformmetadata.Class{})
	var buf bytes.Buffer
	if err := EmitAsync(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output; got:\n%s", buf.String())
	}
}

func TestAsyncBasic(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Foundation", map[string]macosplatformmetadata.Class{
		"NSFoo": {
			Methods: []macosplatformmetadata.Method{
				{
					Selector: "loadWithCompletionHandler:",
					Params: []macosplatformmetadata.Param{
						{Name: "completionHandler", ObjCType: "void (^)(NSError *)", IsBlock: true},
					},
					Return: macosplatformmetadata.ReturnType{ObjCType: "void"},
				},
			},
		},
	})
	var buf bytes.Buffer
	if err := EmitAsync(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "func Load(") {
		t.Errorf("expected Load wrapper; got:\n%s", out)
	}
	if !strings.Contains(out, "chan error") {
		t.Errorf("expected chan error; got:\n%s", out)
	}
	if !strings.Contains(out, "ctx.Done()") {
		t.Errorf("expected ctx.Done() cancellation; got:\n%s", out)
	}
}

func TestAsyncUnavailableClassSkipped(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Foundation", map[string]macosplatformmetadata.Class{
		"NSFoo": {
			Availability: macosplatformmetadata.Availability{IsUnavailable: true},
			Methods: []macosplatformmetadata.Method{
				{
					Selector: "loadWithCompletionHandler:",
					Params:     []macosplatformmetadata.Param{{Name: "handler", ObjCType: "void (^)(NSError *)", IsBlock: true}},
					Return:   macosplatformmetadata.ReturnType{ObjCType: "void"},
				},
			},
		},
	})
	var buf bytes.Buffer
	if err := EmitAsync(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("unavailable class should be skipped; got:\n%s", buf.String())
	}
}

func TestAsyncClassMethod(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Foundation", map[string]macosplatformmetadata.Class{
		"NSFoo": {
			Methods: []macosplatformmetadata.Method{
				{
					Selector:      "loadWithCompletionHandler:",
					IsClassMethod: true,
					Params:          []macosplatformmetadata.Param{{Name: "handler", ObjCType: "void (^)(NSError *)", IsBlock: true}},
					Return:        macosplatformmetadata.ReturnType{ObjCType: "void"},
				},
			},
		},
	})
	var buf bytes.Buffer
	if err := EmitAsync(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "raw.NSFoo") {
		t.Errorf("class method should use raw.NSFoo prefix; got:\n%s", out)
	}
}

// ── Slices ─────────────────────────────────────────────────────────────────────

func TestSlicesEmpty(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Foundation", map[string]macosplatformmetadata.Class{})
	var buf bytes.Buffer
	if err := EmitSlices(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output; got:\n%s", buf.String())
	}
}

func TestSlicesBasic(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Virtualization", map[string]macosplatformmetadata.Class{
		"VZConfiguration": {
			Methods: []macosplatformmetadata.Method{
				{Selector: "storages", Return: macosplatformmetadata.ReturnType{ObjCType: "NSArray<VZDiskImage *> *"}},
				{Selector: "setStorages:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSArray<VZDiskImage *> *"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			},
			Properties: []macosplatformmetadata.Property{
				{Name: "storages", ObjCType: "NSArray<VZDiskImage *> *"},
			},
		},
		"VZDiskImage": {},
	})
	var buf bytes.Buffer
	if err := EmitSlices(&buf, "virtualization", "github.com/example/fw/virtualization", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "StoragesList") {
		t.Errorf("expected StoragesList getter; got:\n%s", out)
	}
	if !strings.Contains(out, "SetStoragesList") {
		t.Errorf("expected SetStoragesList setter; got:\n%s", out)
	}
}

func TestSlicesElemClassNotInFrameworkSkipped(t *testing.T) {
	m := libTestMapper()
	// NSString is in Foundation, not in this fw → skip.
	framework := libTestFM("AppKit", map[string]macosplatformmetadata.Class{
		"NSView": {
			Properties: []macosplatformmetadata.Property{
				{Name: "subviewNames", ObjCType: "NSArray<NSString *> *"},
			},
		},
	})
	var buf bytes.Buffer
	if err := EmitSlices(&buf, "appkit", "github.com/example/fw/appkit", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "SubviewNamesList") {
		t.Errorf("element class not in framework should be skipped; got:\n%s", buf.String())
	}
}

// ── Specs ─────────────────────────────────────────────────────────────────────

func TestSpecsEmpty(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Foundation", map[string]macosplatformmetadata.Class{})
	var buf bytes.Buffer
	if err := EmitSpecs(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output; got:\n%s", buf.String())
	}
}

func TestSpecsNonConfigurationClassSkipped(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Foundation", map[string]macosplatformmetadata.Class{
		"NSFoo": { // not a "Configuration" class
			Properties: []macosplatformmetadata.Property{
				{Name: "a", ObjCType: "NSUInteger"},
				{Name: "b", ObjCType: "NSUInteger"},
				{Name: "c", ObjCType: "NSUInteger"},
			},
		},
	})
	var buf bytes.Buffer
	if err := EmitSpecs(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("non-Configuration class should be skipped; got:\n%s", buf.String())
	}
}

func TestSpecsBasic(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Virtualization", map[string]macosplatformmetadata.Class{
		"VZMachineConfiguration": {
			Methods: []macosplatformmetadata.Method{
				{Selector: "setCpuCount:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				{Selector: "setMemorySize:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "uint64_t"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				{Selector: "setBootLoader:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			},
			Properties: []macosplatformmetadata.Property{
				{Name: "cpuCount", ObjCType: "NSUInteger"},
				{Name: "memorySize", ObjCType: "uint64_t"},
				{Name: "bootLoader", ObjCType: "NSUInteger"},
			},
		},
	})
	var buf bytes.Buffer
	if err := EmitSpecs(&buf, "virtualization", "github.com/example/fw/virtualization", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "VZMachineConfigurationSpec") {
		t.Errorf("expected VZMachineConfigurationSpec; got:\n%s", out)
	}
	if !strings.Contains(out, "ApplyVZMachineConfiguration") {
		t.Errorf("expected ApplyVZMachineConfiguration; got:\n%s", out)
	}
}

func TestSpecsWithValidateMethod(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Virtualization", map[string]macosplatformmetadata.Class{
		"VZFooConfiguration": {
			Methods: []macosplatformmetadata.Method{
				{Selector: "setA:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				{Selector: "setB:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				{Selector: "setC:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				{Selector: "validateWithError:", IsNSError: true, Return: macosplatformmetadata.ReturnType{ObjCType: "BOOL"}},
			},
			Properties: []macosplatformmetadata.Property{
				{Name: "a", ObjCType: "NSUInteger"},
				{Name: "b", ObjCType: "NSUInteger"},
				{Name: "c", ObjCType: "NSUInteger"},
			},
		},
	})
	var buf bytes.Buffer
	if err := EmitSpecs(&buf, "virtualization", "github.com/example/fw/virtualization", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "ValidateWithError") {
		t.Errorf("expected ValidateWithError call in Apply; got:\n%s", out)
	}
}

func TestSpecsFewerThan3PropsSkipped(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Virtualization", map[string]macosplatformmetadata.Class{
		"VZFooConfiguration": {
			Methods: []macosplatformmetadata.Method{
				{Selector: "setA:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				{Selector: "setB:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			},
			Properties: []macosplatformmetadata.Property{
				{Name: "a", ObjCType: "NSUInteger"},
				{Name: "b", ObjCType: "NSUInteger"},
			},
		},
	})
	var buf bytes.Buffer
	if err := EmitSpecs(&buf, "virtualization", "github.com/example/fw/virtualization", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("fewer than 3 props should be skipped; got:\n%s", buf.String())
	}
}

// ── resolveOpinionatedArgType ─────────────────────────────────────────────────

func TestResolveOpinionatedArgTypeObjcObject(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Foundation", map[string]macosplatformmetadata.Class{"NSString": {}})
	// NSString must be in knownClasses for GoType to resolve it.
	ctx := m.BaseContext("Foundation", map[string]bool{"NSString": true})
	got := resolveOpinionatedArgType("NSString *", ctx, m, framework, nil)
	// NSString is in framework.Classes → should be prefixed with "raw."
	if !strings.Contains(got, "raw.") {
		t.Errorf("expected raw. prefix for own-framework type; got %q", got)
	}
}

func TestResolveOpinionatedArgTypePrimitive(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Foundation", map[string]macosplatformmetadata.Class{})
	ctx := m.BaseContext("Foundation", map[string]bool{})
	got := resolveOpinionatedArgType("NSUInteger", ctx, m, framework, nil)
	if got == "" {
		t.Error("expected non-empty type for NSUInteger")
	}
}

// TestAsyncWithNonBlockArg triggers resolveOpinionatedArgType via Async.
func TestAsyncWithNonBlockArg(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Foundation", map[string]macosplatformmetadata.Class{
		"NSFoo": {
			Methods: []macosplatformmetadata.Method{
				{
					Selector: "loadURL:withCompletionHandler:",
					Params: []macosplatformmetadata.Param{
						{Name: "url", ObjCType: "NSString *"},
						{Name: "completionHandler", ObjCType: "void (^)(NSError *)", IsBlock: true},
					},
					Return: macosplatformmetadata.ReturnType{ObjCType: "void"},
				},
			},
		},
	})
	var buf bytes.Buffer
	if err := EmitAsync(&buf, "foundation", "github.com/example/fw/foundation", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "LoadURL") {
		t.Errorf("expected LoadURL function (CompletionHandler stripped); got:\n%s", out)
	}
	if !strings.Contains(out, "url") {
		t.Errorf("expected url parameter in wrapper; got:\n%s", out)
	}
}

// ── recordOpinionatedImports ─────────────────────────────────────────────────

func TestCollectOpinionatedImportsCrossFramework(t *testing.T) {
	m := libTestMapper()
	m.ModulePrefix = "github.com/example/fw"
	m.OwnerIndex["NSString"] = "Foundation"
	usedImports := make(map[string]string)
	recordOpinionatedImports("*foundation.NSString", m, usedImports)
	if _, ok := usedImports["foundation"]; !ok {
		t.Errorf("expected foundation import collected; got %v", usedImports)
	}
}

func TestCollectOpinionatedImportsNoImports(t *testing.T) {
	m := libTestMapper()
	usedImports := make(map[string]string)
	recordOpinionatedImports("uint64", m, usedImports)
	if len(usedImports) != 0 {
		t.Errorf("expected no imports for primitive type; got %v", usedImports)
	}
}

// ── qualifyWithRawPackage edge cases ────────────────────────────────────────────────────

func TestAddRawPrefixCrossFramework(t *testing.T) {
	framework := libTestFM("Virtualization", map[string]macosplatformmetadata.Class{})
	got := qualifyWithRawPackage("foundation.NSString", framework)
	if got != "foundation.NSString" {
		t.Errorf("cross-framework type should be unchanged; got %q", got)
	}
}

func TestAddRawPrefixSliceType(t *testing.T) {
	framework := libTestFM("Virtualization", map[string]macosplatformmetadata.Class{"VZFoo": {}})
	got := qualifyWithRawPackage("[]VZFoo", framework)
	if got != "[]raw.VZFoo" {
		t.Errorf("slice of own-fw type should get raw. prefix; got %q", got)
	}
}

func TestAddRawPrefixProtocolType(t *testing.T) {
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "AppKit",
		Classes:   map[string]macosplatformmetadata.Class{},
		Protocols: map[string]macosplatformmetadata.Protocol{"NSTextFieldDelegate": {}},
		Enums:     map[string]macosplatformmetadata.Enum{},
		Structs:   map[string]macosplatformmetadata.Struct{},
		Typedefs:  map[string]string{},
	}
	got := qualifyWithRawPackage("*NSTextFieldDelegate", framework)
	if got != "*raw.NSTextFieldDelegate" {
		t.Errorf("protocol type should get raw. prefix; got %q", got)
	}
}

func TestAddRawPrefixEnumType(t *testing.T) {
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "AppKit",
		Classes:   map[string]macosplatformmetadata.Class{},
		Protocols: map[string]macosplatformmetadata.Protocol{},
		Enums:     map[string]macosplatformmetadata.Enum{"NSControlSize": {}},
		Structs:   map[string]macosplatformmetadata.Struct{},
		Typedefs:  map[string]string{},
	}
	got := qualifyWithRawPackage("NSControlSize", framework)
	if got != "raw.NSControlSize" {
		t.Errorf("enum type should get raw. prefix; got %q", got)
	}
}

func TestAddRawPrefixStructType(t *testing.T) {
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "CoreGraphics",
		Classes:   map[string]macosplatformmetadata.Class{},
		Protocols: map[string]macosplatformmetadata.Protocol{},
		Enums:     map[string]macosplatformmetadata.Enum{},
		Structs:   map[string]macosplatformmetadata.Struct{"CGRect": {}},
		Typedefs:  map[string]string{},
	}
	got := qualifyWithRawPackage("*CGRect", framework)
	if got != "*raw.CGRect" {
		t.Errorf("struct type should get raw. prefix; got %q", got)
	}
}

func TestAddRawPrefixTypedefType(t *testing.T) {
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]macosplatformmetadata.Class{},
		Protocols: map[string]macosplatformmetadata.Protocol{},
		Enums:     map[string]macosplatformmetadata.Enum{},
		Structs:   map[string]macosplatformmetadata.Struct{},
		Typedefs:  map[string]string{"NSTimeInterval": "double"},
	}
	got := qualifyWithRawPackage("NSTimeInterval", framework)
	if got != "raw.NSTimeInterval" {
		t.Errorf("typedef type should get raw. prefix; got %q", got)
	}
}

func TestAddRawPrefixCompoundInterface(t *testing.T) {
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "AppKit",
		Classes:   map[string]macosplatformmetadata.Class{},
		Protocols: map[string]macosplatformmetadata.Protocol{"NSFoo": {}, "NSBar": {}},
		Enums:     map[string]macosplatformmetadata.Enum{},
		Structs:   map[string]macosplatformmetadata.Struct{},
		Typedefs:  map[string]string{},
	}
	got := qualifyWithRawPackage("interface { NSFoo; NSBar }", framework)
	if !strings.Contains(got, "raw.NSFoo") {
		t.Errorf("compound interface protocol should get raw. prefix; got %q", got)
	}
}

// ── Slices with audio devices ─────────────────────────────────────────────────

// TestSpecsBoolField verifies a bool property generates "if spec.X { c.SetX(...) }".
func TestSpecsBoolField(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Virtualization", map[string]macosplatformmetadata.Class{
		"VZFooConfiguration": {
			Methods: []macosplatformmetadata.Method{
				{Selector: "setEnabled:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "BOOL"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				{Selector: "setA:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				{Selector: "setB:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			},
			Properties: []macosplatformmetadata.Property{
				{Name: "enabled", ObjCType: "BOOL"},
				{Name: "a", ObjCType: "NSUInteger"},
				{Name: "b", ObjCType: "NSUInteger"},
			},
		},
	})
	var buf bytes.Buffer
	if err := EmitSpecs(&buf, "virtualization", "github.com/example/fw/virtualization", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Bool field: "if spec.Enabled { c.SetEnabled(ctx, spec.Enabled) }"
	if !strings.Contains(out, "spec.Enabled") {
		t.Errorf("expected bool field Enabled in spec; got:\n%s", out)
	}
}

// TestSpecsValueStructField verifies a value struct property generates != zero-value check.
func TestSpecsValueStructField(t *testing.T) {
	m := &typemap.Mapper{
		GenericClasses:       map[string]bool{},
		OwnerIndex:       map[string]string{},
		ModulePrefix:         "github.com/example/fw",
		BlockedImports:       map[string]map[string]bool{},
		TypedefIndex:        map[string]string{},
		StructIndex:         map[string]string{"CGRect": "CoreGraphics"},
		ProtocolIndex:       map[string]string{},
		ProtocolProxyIndex: map[string]string{},
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "CoreGraphics",
		Classes: map[string]macosplatformmetadata.Class{
			"CGFooConfiguration": {
				Methods: []macosplatformmetadata.Method{
					{Selector: "setFrame:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "CGRect"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
					{Selector: "setA:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
					{Selector: "setB:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				},
				Properties: []macosplatformmetadata.Property{
					{Name: "frame", ObjCType: "CGRect"},
					{Name: "a", ObjCType: "NSUInteger"},
					{Name: "b", ObjCType: "NSUInteger"},
				},
			},
		},
		Protocols: map[string]macosplatformmetadata.Protocol{},
		Enums:     map[string]macosplatformmetadata.Enum{},
		Structs:   map[string]macosplatformmetadata.Struct{},
		Typedefs:  map[string]string{},
	}
	var buf bytes.Buffer
	if err := EmitSpecs(&buf, "coregraphics", "github.com/example/fw/coregraphics", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Value struct field uses != zero-value comparison.
	if !strings.Contains(out, "spec.Frame") {
		t.Errorf("expected Frame field in spec; got:\n%s", out)
	}
}

// TestSpecsSliceField verifies an NSArray<Element*>* property with in-fw element generates slice field.
func TestSpecsSliceField(t *testing.T) {
	m := libTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Virtualization",
		Classes: map[string]macosplatformmetadata.Class{
			"VZStorage":           {},
			"VZFooConfiguration": {
				Methods: []macosplatformmetadata.Method{
					{Selector: "setStorages:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSArray<VZStorage *> *"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
					{Selector: "setA:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
					{Selector: "setB:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSUInteger"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				},
				Properties: []macosplatformmetadata.Property{
					{Name: "storages", ObjCType: "NSArray<VZStorage *> *"},
					{Name: "a", ObjCType: "NSUInteger"},
					{Name: "b", ObjCType: "NSUInteger"},
				},
			},
		},
		Protocols: map[string]macosplatformmetadata.Protocol{},
		Enums:     map[string]macosplatformmetadata.Enum{},
		Structs:   map[string]macosplatformmetadata.Struct{},
		Typedefs:  map[string]string{},
	}
	var buf bytes.Buffer
	if err := EmitSpecs(&buf, "virtualization", "github.com/example/fw/virtualization", framework, m, map[string]bool{"VZStorage": true, "VZFooConfiguration": true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Storages") {
		t.Errorf("expected Storages slice field in spec; got:\n%s", out)
	}
	// Slice field uses len() check.
	if !strings.Contains(out, "len(spec.Storages)") {
		t.Errorf("expected len(spec.Storages) check; got:\n%s", out)
	}
}

// TestResolveOpinionatedArgTypeVoidFallback covers the empty-GoType → unsafe.Pointer fallback.
func TestResolveOpinionatedArgTypeVoidFallback(t *testing.T) {
	m := libTestMapper()
	framework := libTestFM("Foundation", map[string]macosplatformmetadata.Class{})
	ctx := m.BaseContext("Foundation", map[string]bool{})
	// "void" resolves to "" in GoType → should return "unsafe.Pointer".
	got := resolveOpinionatedArgType("void", ctx, m, framework, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("void type should resolve to unsafe.Pointer; got %q", got)
	}
}

func TestSlicesAudioDevices(t *testing.T) {
	m := libTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Virtualization",
		Classes: map[string]macosplatformmetadata.Class{
			"VZAudioDevice": {},
			"VZMachineConfig": {
				Methods: []macosplatformmetadata.Method{
					{Selector: "setAudioDevices:", Params: []macosplatformmetadata.Param{{Name: "v", ObjCType: "NSArray *"}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				},
				Properties: []macosplatformmetadata.Property{
					{Name: "audioDevices", ObjCType: "NSArray<VZAudioDevice *> *"},
				},
			},
		},
		Protocols: map[string]macosplatformmetadata.Protocol{},
		Enums:     map[string]macosplatformmetadata.Enum{},
		Structs:   map[string]macosplatformmetadata.Struct{},
		Typedefs:  map[string]string{},
	}
	var buf bytes.Buffer
	if err := EmitSlices(&buf, "virtualization", "github.com/example/fw/virtualization", framework, m, map[string]bool{"VZAudioDevice": true, "VZMachineConfig": true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "AudioDevices") {
		t.Errorf("expected AudioDevices accessor; got:\n%s", out)
	}
}
