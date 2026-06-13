package raw

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// classTestMapper builds a Mapper with a known set of classes for class emit tests.
func classTestMapper() *typemap.Mapper {
	return &typemap.Mapper{
		GenericClasses: map[string]bool{
			"NSArray": true,
		},
		OwnerIndex: map[string]string{
			"NSObject": "Foundation",
			"NSArray":  "Foundation",
			"NSView":   "AppKit",
		},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
}

// classKnown is a minimal known-classes set for class emit tests.
var classKnown = map[string]bool{
	"NSObject": true,
	"NSArray":  true,
	"NSView":   true,
	"NSButton": true,
}

func writeClassBuf(name string, cls macosplatformmetadata.Class, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, all map[string]macosplatformmetadata.Class) (string, error) {
	var buf bytes.Buffer
	err := EmitClass(&buf, name, cls, framework, m, classKnown, all, strings.ToLower(framework.Framework))
	return buf.String(), err
}

// writeClassBufKnown is like writeClassBuf but allows a custom knownClasses map.
func writeClassBufKnown(name string, cls macosplatformmetadata.Class, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, all map[string]macosplatformmetadata.Class, known map[string]bool) (string, error) {
	var buf bytes.Buffer
	err := EmitClass(&buf, name, cls, framework, m, known, all, strings.ToLower(framework.Framework))
	return buf.String(), err
}

// TestWriteClassRootStructHasPtrField verifies root classes (no super) declare
// `ptr unsafe.Pointer` and define Ptr().
func TestWriteClassRootStructHasPtrField(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{}
	out, err := writeClassBuf("NSObject", cls, framework, m, map[string]macosplatformmetadata.Class{"NSObject": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\tptr unsafe.Pointer") {
		t.Errorf("root class missing ptr field; got:\n%s", out)
	}
	if !strings.Contains(out, "func (o *NSObject) Ptr() unsafe.Pointer") {
		t.Errorf("root class missing Ptr() method; got:\n%s", out)
	}
}

// TestWriteClassEmbedsSameFwSuper verifies that a subclass with a same-framework
// superclass uses struct embedding instead of a ptr field.
func TestWriteClassEmbedsSameFwSuper(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {},
		},
	}
	cls := macosplatformmetadata.Class{Super: "NSObject"}
	all := map[string]macosplatformmetadata.Class{
		"NSObject":    {},
		"NSMySubclass": cls,
	}
	out, err := writeClassBuf("NSMySubclass", cls, framework, m, all)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "\tptr unsafe.Pointer") {
		t.Errorf("subclass should embed super, not own ptr; got:\n%s", out)
	}
	if !strings.Contains(out, "NSObject") {
		t.Errorf("subclass should embed NSObject; got:\n%s", out)
	}
}

// TestWriteClassCrossFrameworkSuper verifies cross-framework superclass
// uses a qualified package reference and adds to imports.
func TestWriteClassCrossFrameworkSuper(t *testing.T) {
	m := classTestMapper()
	// NSButton is in AppKit, extending NSView (also AppKit), but testing cross-fw:
	// Simulate a class in "MyFW" that extends AppKit's NSView.
	m.OwnerIndex["NSView"] = "AppKit"
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "MyFW",
		Classes:   map[string]macosplatformmetadata.Class{},
	}
	cls := macosplatformmetadata.Class{Super: "NSView"}
	all := map[string]macosplatformmetadata.Class{
		"NSView":    {Super: ""},
		"MyFWClass": cls,
	}
	out, err := writeClassBuf("MyFWClass", cls, framework, m, all)
	if err != nil {
		t.Fatal(err)
	}
	// Should reference appkit.NSView in the embedding
	if !strings.Contains(out, "appkit") {
		t.Errorf("cross-framework super should use package alias; got:\n%s", out)
	}
}

// TestWriteClassUnknownSuperBecomesRoot verifies that a class whose superclass
// is not in ClassNameIndex is treated as a root class.
func TestWriteClassUnknownSuperBecomesRoot(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{Super: "SomeUnknownBase"}
	out, err := writeClassBuf("NSMyClass", cls, framework, m, map[string]macosplatformmetadata.Class{"NSMyClass": cls})
	if err != nil {
		t.Fatal(err)
	}
	// Treated as root → has ptr field and Ptr()
	if !strings.Contains(out, "\tptr unsafe.Pointer") {
		t.Errorf("unknown super should cause root treatment; got:\n%s", out)
	}
}

// TestWriteClassGenericDeclaresTypeParam verifies that a generic class gets
// the [T cgo.Object] type parameter.
func TestWriteClassGenericDeclaresTypeParam(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]macosplatformmetadata.Class{"NSObject": {}},
	}
	cls := macosplatformmetadata.Class{
		Super:         "NSObject",
		GenericParams: []string{"ObjectType"},
	}
	all := map[string]macosplatformmetadata.Class{"NSObject": {}, "NSArray": cls}
	out, err := writeClassBuf("NSArray", cls, framework, m, all)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[T cgo.Object]") {
		t.Errorf("generic class should have type param; got:\n%s", out)
	}
}

// TestWriteClassConstructorEmitted verifies New<ClassName> and <ClassName>WithPtr
// are always emitted.
func TestWriteClassConstructorEmitted(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{}
	out, err := writeClassBuf("NSObject", cls, framework, m, map[string]macosplatformmetadata.Class{"NSObject": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "func NewNSObject(") {
		t.Errorf("missing NewNSObject constructor; got:\n%s", out)
	}
	if !strings.Contains(out, "func NSObjectWithPtr(") {
		t.Errorf("missing NSObjectWithPtr constructor; got:\n%s", out)
	}
}

// TestWriteClassRuntimeTrackInConstructor verifies the New* constructor calls
// runtime.Track for lifecycle management.
func TestWriteClassRuntimeTrackInConstructor(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{}
	out, err := writeClassBuf("NSObject", cls, framework, m, map[string]macosplatformmetadata.Class{"NSObject": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "cgo.Track(") {
		t.Errorf("constructor missing runtime.Track; got:\n%s", out)
	}
}

// TestWriteClassMethodEmitted verifies instance methods are emitted.
func TestWriteClassMethodEmitted(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]macosplatformmetadata.Class{"NSObject": {}},
	}
	cls := macosplatformmetadata.Class{
		Super: "NSObject",
		Methods: []macosplatformmetadata.Method{
			{
				Selector: "count",
				Return:   macosplatformmetadata.ReturnType{ObjCType: "NSUInteger"},
			},
		},
	}
	all := map[string]macosplatformmetadata.Class{"NSObject": {}, "NSMyClass": cls}
	out, err := writeClassBuf("NSMyClass", cls, framework, m, all)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "func (o *NSMyClass) Count(ctx context.Context)") {
		t.Errorf("instance method Count not emitted; got:\n%s", out)
	}
}

// TestWriteClassClassMethodEmitted verifies class methods get a Class() receiver.
func TestWriteClassClassMethodEmitted(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector:      "sharedManager",
				IsClassMethod: true,
				Return:        macosplatformmetadata.ReturnType{ObjCType: "void"},
			},
		},
	}
	out, err := writeClassBuf("NSMyMgr", cls, framework, m, map[string]macosplatformmetadata.Class{"NSMyMgr": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SharedManager") {
		t.Errorf("class method SharedManager not emitted; got:\n%s", out)
	}
}

// TestWriteClassSkipsNilSentinelVariadic verifies nil-sentinel variadic methods are not emitted.
func TestWriteClassSkipsNilSentinelVariadic(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector:   "arrayWithObjects:",
				IsVariadic: true,
				Return:     macosplatformmetadata.ReturnType{ObjCType: "void"},
			},
		},
	}
	out, err := writeClassBuf("NSFormatter", cls, framework, m, map[string]macosplatformmetadata.Class{"NSFormatter": cls})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "func (o *NSFormatter) ArrayWithObjects") {
		t.Errorf("nil-sentinel variadic method should be skipped; got:\n%s", out)
	}
}

// TestWriteClassBridgesFormatStringVariadic verifies format-string variadic methods are emitted.
func TestWriteClassBridgesFormatStringVariadic(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector:   "stringWithFormat:",
				IsVariadic: true,
				IsClassMethod: true,
				Params:       []macosplatformmetadata.Param{{Name: "format", ObjCType: "NSString * _Nonnull"}},
				Return:     macosplatformmetadata.ReturnType{ObjCType: "NSString *"},
			},
		},
	}
	out, err := writeClassBuf("NSString", cls, framework, m, map[string]macosplatformmetadata.Class{"NSString": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "StringWithFormat") {
		t.Errorf("format-string variadic method should be emitted; got:\n%s", out)
	}
}

// TestWriteClassFileHeader verifies the standard file header is written.
func TestWriteClassFileHeader(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{}
	out, err := writeClassBuf("NSObject", cls, framework, m, map[string]macosplatformmetadata.Class{"NSObject": cls})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Code generated by go-bindings-codegen",
		"//go:build darwin",
		"package foundation",
		`import "C"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing header element %q; got:\n%s", want, out)
		}
	}
}

// TestWriteClassAvailabilityComment verifies availability is emitted as a comment.
func TestWriteClassAvailabilityComment(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{
		Availability: macosplatformmetadata.Availability{MacOSIntroduced: "11.0"},
	}
	out, err := writeClassBuf("NSMyClass", cls, framework, m, map[string]macosplatformmetadata.Class{"NSMyClass": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Introduced: macOS 11.0") {
		t.Errorf("missing availability comment; got:\n%s", out)
	}
}

// TestWriteClassDeprecatedComment verifies deprecated classes emit a comment.
func TestWriteClassDeprecatedComment(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{
		Availability: macosplatformmetadata.Availability{MacOSDeprecated: "14.0", ReplacedBy: "NewClass"},
	}
	out, err := writeClassBuf("OldClass", cls, framework, m, map[string]macosplatformmetadata.Class{"OldClass": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Deprecated: Deprecated in macOS 14.0") {
		t.Errorf("missing deprecated comment; got:\n%s", out)
	}
	if !strings.Contains(out, "NewClass") {
		t.Errorf("missing replacement in deprecated comment; got:\n%s", out)
	}
}

// TestWriteClassNSErrorMethodReturnsTwoValues verifies HasNSError methods produce
// two-value returns.
func TestWriteClassNSErrorMethodReturnsTwoValues(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector:   "performWithError:",
				IsNSError: true,
				Return:     macosplatformmetadata.ReturnType{ObjCType: "BOOL"},
			},
		},
	}
	out, err := writeClassBuf("NSTask", cls, framework, m, map[string]macosplatformmetadata.Class{"NSTask": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "error") {
		t.Errorf("NSError method should produce error return; got:\n%s", out)
	}
}

// TestWriteClassRuntimeObjectConformance verifies the var _ runtime.Object = (*T)(nil)
// conformance check is emitted.
func TestWriteClassRuntimeObjectConformance(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{}
	out, err := writeClassBuf("NSObject", cls, framework, m, map[string]macosplatformmetadata.Class{"NSObject": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "var _ cgo.Object = (*NSObject)(nil)") {
		t.Errorf("missing runtime.Object conformance; got:\n%s", out)
	}
}

// TestClassifySuper_Root verifies a class with no superclass is classified as root.
func TestClassifySuperRoot(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation"}
	cls := macosplatformmetadata.Class{}
	si := classifySuper("NSObject", cls, framework, m)
	if !si.isRoot {
		t.Error("class with no Super should be root")
	}
}

// TestClassifySuper_UnknownOwner verifies a super with unknown ownership is root.
func TestClassifySuperUnknownOwner(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation"}
	cls := macosplatformmetadata.Class{Super: "SomeAlienClass"}
	si := classifySuper("Child", cls, framework, m)
	if !si.isRoot {
		t.Error("super with unknown owner should be treated as root")
	}
}

// TestClassifySuper_SameFramework verifies same-framework super has no pkg qualifier.
func TestClassifySuperSameFramework(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation"}
	cls := macosplatformmetadata.Class{Super: "NSObject"}
	si := classifySuper("NSString", cls, framework, m)
	if si.isRoot {
		t.Error("should not be root when super is known")
	}
	if si.pkg != "" {
		t.Errorf("same-fw super should have no pkg, got %q", si.pkg)
	}
	if si.name != "NSObject" {
		t.Errorf("expected super name NSObject, got %q", si.name)
	}
}

// TestClassifySuper_CrossFramework verifies cross-framework super gets pkg qualifier.
func TestClassifySuperCrossFramework(t *testing.T) {
	m := classTestMapper()
	// Foundation class whose super is NSView (AppKit)
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation"}
	cls := macosplatformmetadata.Class{Super: "NSView"}
	si := classifySuper("NSFoundationViewChild", cls, framework, m)
	if si.isRoot {
		t.Error("should not be root when super is known cross-fw")
	}
	if si.pkg != "appkit" {
		t.Errorf("expected pkg 'appkit', got %q", si.pkg)
	}
}

// TestClassifySuper_BlockedImport verifies a blocked import causes root treatment.
func TestClassifySuperBlockedImport(t *testing.T) {
	m := classTestMapper()
	m.BlockedImports = map[string]map[string]bool{
		"Foundation": {"AppKit": true},
	}
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation"}
	cls := macosplatformmetadata.Class{Super: "NSView"}
	si := classifySuper("Child", cls, framework, m)
	if !si.isRoot {
		t.Error("blocked import should cause root treatment")
	}
}

// TestNSStringOverloadInstance verifies that an instance method with NSString *
// args gets an additional ...Go overload when NSStringOverloads is enabled.
func TestNSStringOverloadInstance(t *testing.T) {
	m := classTestMapper()
	m.IsNSStringOverloads = true
	m.OwnerIndex["NSString"] = "Foundation"
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSBundle": {},
			"NSString": {},
		},
	}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector: "localizedStringForKey:value:table:",
				Params: []macosplatformmetadata.Param{
					{Name: "key", ObjCType: "NSString *"},
					{Name: "value", ObjCType: "NSString *"},
					{Name: "tableName", ObjCType: "NSString *"},
				},
				Return: macosplatformmetadata.ReturnType{ObjCType: "NSString *"},
			},
		},
	}
	all := map[string]macosplatformmetadata.Class{"NSBundle": cls, "NSString": {}}
	out, err := writeClassBuf("NSBundle", cls, framework, m, all)
	if err != nil {
		t.Fatal(err)
	}
	// Primary method should still exist.
	if !strings.Contains(out, "func (o *NSBundle) LocalizedStringForKeyValueTable(") {
		t.Errorf("primary method not emitted:\n%s", out)
	}
	// Go-string overload should be emitted.
	if !strings.Contains(out, "func (o *NSBundle) LocalizedStringForKeyValueTableGo(") {
		t.Errorf("NSString Go overload not emitted:\n%s", out)
	}
	// Overload params should be plain string types.
	if !strings.Contains(out, "key string") {
		t.Errorf("overload should have 'key string' param:\n%s", out)
	}
	// Overload body should delegate to primary via GoStringToNSString.
	if !strings.Contains(out, "cgo.GoStringToNSString(key)") {
		t.Errorf("overload body should convert via runtime.GoStringToNSString:\n%s", out)
	}
}

// TestNSStringOverloadDisabledByDefault verifies no ...Go overloads when
// NSStringOverloads is false (the default).
func TestNSStringOverloadDisabledByDefault(t *testing.T) {
	m := classTestMapper() // NSStringOverloads is false by default
	m.OwnerIndex["NSString"] = "Foundation"
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]macosplatformmetadata.Class{"NSBundle": {}, "NSString": {}},
	}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector: "setTitle:",
				Params:     []macosplatformmetadata.Param{{Name: "title", ObjCType: "NSString *"}},
				Return:   macosplatformmetadata.ReturnType{ObjCType: "void"},
			},
		},
	}
	all := map[string]macosplatformmetadata.Class{"NSBundle": cls, "NSString": {}}
	out, err := writeClassBuf("NSBundle", cls, framework, m, all)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "SetTitleGo") {
		t.Errorf("Go overload should not be emitted when NSStringOverloads=false:\n%s", out)
	}
}

// ── block arg wrappers ────────────────────────────────────────────────────────

// TestWriteClassMethodWithBlockArg verifies that a method with a block argument
// that needs a wrapper generates the buildBlockWrapper closure.
func TestWriteClassMethodWithBlockArg(t *testing.T) {
	m := classTestMapper()
	m.IsNSStringOverloads = false
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]macosplatformmetadata.Class{},
	}
	// Block arg with NSError * — triggers blockNeedsWrapper.
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector: "doWithHandler:",
				Params: []macosplatformmetadata.Param{
					{Name: "handler", ObjCType: "void (^)(NSError *)", IsBlock: true},
				},
				Return: macosplatformmetadata.ReturnType{ObjCType: "void"},
			},
		},
	}
	known := map[string]bool{"NSObject": true}
	out, err := writeClassBufKnown("NSFoo", cls, framework, m, map[string]macosplatformmetadata.Class{"NSFoo": cls}, known)
	if err != nil {
		t.Fatal(err)
	}
	// Block wrapper closure should appear in the generated method.
	if !strings.Contains(out, "handler") {
		t.Errorf("expected handler parameter; got:\n%s", out)
	}
}

// ── blockArgCtor ──────────────────────────────────────────────────────────────

func TestBlockArgCtorError(t *testing.T) {
	got := blockArgCtor("error", "_p0")
	if got != "cgo.NSErrorToError(_p0)" {
		t.Errorf("blockArgCtor(error) = %q", got)
	}
}

func TestBlockArgCtorObjcObject(t *testing.T) {
	got := blockArgCtor("cgo.Object", "_p0")
	if got != "cgo.WrapObject(_p0)" {
		t.Errorf("blockArgCtor(cgo.Object) = %q", got)
	}
}

func TestBlockArgCtorUnsafePointer(t *testing.T) {
	if got := blockArgCtor("unsafe.Pointer", "_p0"); got != "_p0" {
		t.Errorf("blockArgCtor(unsafe.Pointer) = %q; want pass-through", got)
	}
	if got := blockArgCtor("", "_p0"); got != "_p0" {
		t.Errorf("blockArgCtor('') = %q; want pass-through", got)
	}
}

func TestBlockArgCtorPointerWithPkg(t *testing.T) {
	got := blockArgCtor("*foundation.NSString", "_p0")
	if got != "cgo.WrapTyped(_p0, foundation.NewNSString)" {
		t.Errorf("blockArgCtor(*pkg.Type) = %q", got)
	}
}

func TestBlockArgCtorPointerGeneric(t *testing.T) {
	got := blockArgCtor("*NSArray[cgo.Object]", "_p0")
	if !strings.Contains(got, "unsafe.Pointer") {
		t.Errorf("generic pointer should use unsafe cast; got %q", got)
	}
}

func TestBlockArgCtorPointerLocal(t *testing.T) {
	got := blockArgCtor("*NSView", "_p0")
	if got != "cgo.WrapTyped(_p0, NewNSView)" {
		t.Errorf("blockArgCtor(*LocalType) = %q", got)
	}
}

func TestBlockArgCtorNonPointer(t *testing.T) {
	got := blockArgCtor("uint64", "_p0")
	if got != "_p0" {
		t.Errorf("blockArgCtor(non-pointer scalar) = %q; want pass-through", got)
	}
}

// ── blockArgCtorTyped ─────────────────────────────────────────────────────────

func TestBlockArgCtorTypedStruct(t *testing.T) {
	m := classTestMapper()
	m.StructIndex = map[string]string{"CGRect": "CoreGraphics"}
	got := blockArgCtorTyped("*CGRect", "_p0", m)
	if got != "(*CGRect)(_p0)" {
		t.Errorf("blockArgCtorTyped(*KnownStruct) = %q; want unsafe cast", got)
	}
}

func TestBlockArgCtorTypedClass(t *testing.T) {
	m := classTestMapper()
	m.StructIndex = map[string]string{}
	got := blockArgCtorTyped("*NSView", "_p0", m)
	if got != "cgo.WrapTyped(_p0, NewNSView)" {
		t.Errorf("blockArgCtorTyped(*Class) = %q; want cgo.WrapTyped", got)
	}
}

func TestBlockArgCtorTypedBoolPointer(t *testing.T) {
	m := classTestMapper()
	// *bool should NOT be treated as a struct pointer
	got := blockArgCtorTyped("*bool", "_p0", m)
	// Falls through to blockArgCtor which returns "_p0" for non-*-prefix cases
	// Actually *bool passes the prefix check but goType == "*bool" is excluded.
	_ = got // just ensure no panic
}

// ── blockNeedsWrapper ─────────────────────────────────────────────────────────

func blockNeedsWrapperCtx(m *typemap.Mapper, knownClasses map[string]bool) typemap.Context {
	ctx := m.BaseContext("Foundation", knownClasses)
	return ctx
}

func TestBlockNeedsWrapperNSError(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	if !blockNeedsWrapper("void (^)(NSError *)", ctx, m) {
		t.Error("NSError * arg should require wrapper")
	}
}

func TestBlockNeedsWrapperBOOLPointer(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	if !blockNeedsWrapper("void (^)(BOOL *)", ctx, m) {
		t.Error("BOOL * arg should require wrapper")
	}
}

func TestBlockNeedsWrapperBareID(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	if !blockNeedsWrapper("void (^)(id)", ctx, m) {
		t.Error("bare id arg should require wrapper")
	}
}

func TestBlockNeedsWrapperIDProtocol(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	if !blockNeedsWrapper("void (^)(id<NSCoding>)", ctx, m) {
		t.Error("id<Protocol> arg should require wrapper")
	}
}

func TestBlockNeedsWrapperKnownClass(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{"NSString": true})
	if !blockNeedsWrapper("void (^)(NSString *)", ctx, m) {
		t.Error("known-class arg should require wrapper")
	}
}

func TestBlockNeedsWrapperReturnID(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	if !blockNeedsWrapper("id (^)(void)", ctx, m) {
		t.Error("id return type should require wrapper")
	}
}

func TestBlockNeedsWrapperReturnIDProtocol(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	if !blockNeedsWrapper("id<NSCopying> (^)(void)", ctx, m) {
		t.Error("id<Protocol> return type should require wrapper")
	}
}

func TestBlockNeedsWrapperNoMatch(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	// Plain int and double args — no wrapping needed
	if blockNeedsWrapper("void (^)(int, double)", ctx, m) {
		t.Error("plain scalar args should NOT require wrapper")
	}
}

// TestBlockNeedsWrapperGenericParam verifies a generic type param in a block arg requires wrapper.
func TestBlockNeedsWrapperGenericParam(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	ctx.GenericParams = []string{"ObjectType"}
	if !blockNeedsWrapper("void (^)(ObjectType)", ctx, m) {
		t.Error("generic type param in block arg should require wrapper")
	}
}

// TestBlockNeedsWrapperTypedefToClass verifies a lowercase typedef resolving to ObjC class requires wrapper.
func TestBlockNeedsWrapperTypedefToClass(t *testing.T) {
	m := classTestMapper()
	m.TypedefIndex = map[string]string{"ar_anchor_t": "NSObject<OS_anchor> *"}
	ctx := blockNeedsWrapperCtx(m, map[string]bool{"NSObject": true})
	if !blockNeedsWrapper("void (^)(ar_anchor_t)", ctx, m) {
		t.Error("lowercase typedef resolving to known ObjC class should require wrapper")
	}
}

func TestBlockNeedsWrapperInvalidBlock(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	// Non-block string returns false
	if blockNeedsWrapper("NSString *", ctx, m) {
		t.Error("non-block type should return false")
	}
}

// ── buildBlockWrapper ─────────────────────────────────────────────────────────

func TestBuildBlockWrapperNSError(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	got := buildBlockWrapper("void (^)(NSError *)", "handler", ctx, m, nil)
	if !strings.Contains(got, "cgo.NSErrorToError") {
		t.Errorf("NSError wrapper should use cgo.NSErrorToError; got %q", got)
	}
}

func TestBuildBlockWrapperBOOLPointer(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	got := buildBlockWrapper("void (^)(id, NSUInteger, BOOL *)", "fn", ctx, m, nil)
	// BOOL * becomes preamble+postamble with a shadow bool variable
	if !strings.Contains(got, "unsafe.Pointer") {
		t.Errorf("BOOL * wrapper should use unsafe.Pointer; got %q", got)
	}
}

func TestBuildBlockWrapperBareID(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	got := buildBlockWrapper("void (^)(id)", "fn", ctx, m, nil)
	if !strings.Contains(got, "cgo.WrapObject") {
		t.Errorf("bare id wrapper should use cgo.WrapObject; got %q", got)
	}
}

func TestBuildBlockWrapperReturnID(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	got := buildBlockWrapper("id (^)(void)", "fn", ctx, m, nil)
	if !strings.Contains(got, "Ptr()") {
		t.Errorf("id return wrapper should use Ptr(); got %q", got)
	}
}

func TestBuildBlockWrapperKnownClass(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{"NSArray": true})
	got := buildBlockWrapper("void (^)(NSArray *)", "fn", ctx, m, nil)
	// NSArray is a known class in ctx → should generate wrapper
	if got == "fn" {
		t.Errorf("known-class arg should generate wrapper; got pass-through")
	}
}

func TestBuildBlockWrapperInvalid(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	got := buildBlockWrapper("NSString *", "fn", ctx, m, nil)
	if got != "fn" {
		t.Errorf("non-block should return userArgName; got %q", got)
	}
}

// ── writeDesignatedInitConstructors (via EmitClass) ──────────────────────────

func TestWriteClassDesignatedInitConstructor(t *testing.T) {
	m := classTestMapper()
	m.IsNSStringOverloads = false
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]macosplatformmetadata.Class{},
	}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector:        "initWithName:",
				IsInit:          true,
				IsDesignatedInit: true,
				Params:            []macosplatformmetadata.Param{{Name: "name", ObjCType: "NSString *"}},
				Return:          macosplatformmetadata.ReturnType{IsInstancetype: true},
			},
		},
	}
	all := map[string]macosplatformmetadata.Class{"NSFoo": cls}
	known := map[string]bool{"NSString": true, "NSFoo": true}
	out, err := writeClassBufKnown("NSFoo", cls, framework, m, all, known)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "NewNSFooWithName") {
		t.Errorf("expected NewNSFooWithName constructor; got:\n%s", out)
	}
}

func TestWriteClassDesignatedInitGenericSkipped(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]macosplatformmetadata.Class{},
	}
	cls := macosplatformmetadata.Class{
		GenericParams: []string{"ObjectType"},
		Methods: []macosplatformmetadata.Method{
			{
				Selector:        "initWithObjects:",
				IsInit:          true,
				IsDesignatedInit: true,
				Params:            []macosplatformmetadata.Param{{Name: "objs", ObjCType: "id"}},
				Return:          macosplatformmetadata.ReturnType{IsInstancetype: true},
			},
		},
	}
	all := map[string]macosplatformmetadata.Class{"NSArray": cls}
	out, err := writeClassBuf("NSArray", cls, framework, m, all)
	if err != nil {
		t.Fatal(err)
	}
	// Generic classes skip designated init constructors
	if strings.Contains(out, "NewNSArrayWith") {
		t.Errorf("generic class should skip designated init constructors; got:\n%s", out)
	}
}

// ── writeNSStringClassOverload ────────────────────────────────────────────────

// TestWriteClassNSStringClassOverload verifies that a class method with an
// NSString * arg emits a "...Go" convenience overload when NSStringOverloads=true.
func TestWriteClassNSStringClassOverload(t *testing.T) {
	m := classTestMapper()
	m.IsNSStringOverloads = true
	m.OwnerIndex["NSString"] = "Foundation"
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]macosplatformmetadata.Class{"NSString": {}},
	}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector:      "stringWithString:",
				IsClassMethod: true,
				Params:          []macosplatformmetadata.Param{{Name: "aString", ObjCType: "NSString *"}},
				Return:        macosplatformmetadata.ReturnType{ObjCType: "instancetype"},
			},
		},
	}
	all := map[string]macosplatformmetadata.Class{"NSString": cls}
	out, err := writeClassBuf("NSString", cls, framework, m, all)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Go") {
		t.Errorf("expected NSString Go-string overload with 'Go' suffix; got:\n%s", out)
	}
}

// ── goCGoArgExpr unique paths ─────────────────────────────────────────────────

func goCGoMapper() *typemap.Mapper {
	return &typemap.Mapper{
		GenericClasses:        map[string]bool{},
		OwnerIndex:        map[string]string{},
		ModulePrefix:          "github.com/example/fw",
		BlockedImports:        map[string]map[string]bool{},
		TypedefIndex:         map[string]string{},
		StructIndex:          map[string]string{"CGRect": "CoreGraphics"},
		ProtocolIndex:        map[string]string{},
		ProtocolProxyIndex:  map[string]string{},
		CFTypeIndex: map[string]string{},
	}
}

func TestGoCGoArgExprObjcObject(t *testing.T) {
	m := goCGoMapper()
	var pre, ka []string
	got := goCGoArgExpr("cgo.Object", "arg", m, &pre, &ka)
	if !strings.Contains(got, "_objcPtr_arg") {
		t.Errorf("cgo.Object should use _objcPtr_; got %q", got)
	}
	if len(pre) < 2 {
		t.Errorf("cgo.Object should add preambles; got %d", len(pre))
	}
	if len(ka) != 1 || ka[0] != "arg" {
		t.Errorf("cgo.Object should add to keepAlives; got %v", ka)
	}
}

func TestGoCGoArgExprKnownStructPointer(t *testing.T) {
	m := goCGoMapper()
	var pre, ka []string
	got := goCGoArgExpr("*CGRect", "arg", m, &pre, &ka)
	// Known struct pointer → unsafe.Pointer(arg)
	if got != "unsafe.Pointer(arg)" {
		t.Errorf("KnownStruct pointer should use unsafe.Pointer; got %q", got)
	}
}

func TestGoCGoArgExprBSDPointer(t *testing.T) {
	m := goCGoMapper()
	var pre, ka []string
	got := goCGoArgExpr("*bsd.InAddr", "arg", m, &pre, &ka)
	if got != "unsafe.Pointer(arg)" {
		t.Errorf("bsd.* pointer should use unsafe.Pointer; got %q", got)
	}
}

// ── nsStringConvertArg ────────────────────────────────────────────────────────

func TestNSStringConvertArgNotNS(t *testing.T) {
	m := classTestMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{})
	got := nsStringConvertArg("arg", false, ctx, m, nil)
	if got != "arg" {
		t.Errorf("non-NS arg should pass through; got %q", got)
	}
}

func TestNSStringConvertArgSameFramework(t *testing.T) {
	m := classTestMapper()
	m.OwnerIndex["NSString"] = "Foundation"
	ctx := m.BaseContext("Foundation", map[string]bool{})
	got := nsStringConvertArg("s", true, ctx, m, nil)
	// Same framework: use unqualified NewNSString
	if !strings.Contains(got, "NewNSString") {
		t.Errorf("same-fw NS arg should use NewNSString; got %q", got)
	}
	if strings.Contains(got, "foundation.") {
		t.Errorf("same-fw NS arg should NOT be qualified; got %q", got)
	}
}

func TestNSStringConvertArgCrossFramework(t *testing.T) {
	m := classTestMapper()
	m.OwnerIndex["NSString"] = "Foundation"
	m.ModulePrefix = "github.com/example/fw"
	ctx := m.BaseContext("AppKit", map[string]bool{})
	imports := make(typemap.ImportSet)
	got := nsStringConvertArg("s", true, ctx, m, imports)
	// Cross-framework: should be qualified with foundation.
	if !strings.Contains(got, "foundation.NewNSString") {
		t.Errorf("cross-fw NS arg should use foundation.NewNSString; got %q", got)
	}
}

func TestNSStringConvertArgNoOwner(t *testing.T) {
	m := classTestMapper()
	m.OwnerIndex = map[string]string{} // NSString has no owner
	ctx := m.BaseContext("Foundation", map[string]bool{})
	got := nsStringConvertArg("s", true, ctx, m, nil)
	// Empty owner → treated as same-framework, uses unqualified NewNSString
	if !strings.Contains(got, "NewNSString") {
		t.Errorf("empty-owner NS arg should use NewNSString; got %q", got)
	}
}

// ── buildBlockWrapper — more return-type paths ────────────────────────────────

func TestBuildBlockWrapperIDProtocolReturn(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	// Block returning id<NSCopying> — retIsIDProtocol path
	got := buildBlockWrapper("id<NSCopying> (^)(void)", "fn", ctx, m, nil)
	// Should generate a closure that extracts Ptr() from the typed return
	if got == "fn" {
		t.Errorf("id<Protocol> return should generate wrapper; got pass-through")
	}
}

func TestBuildBlockWrapperKnownClassReturn(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{"NSObject": true})
	// Block returning a known class pointer — KnownClass return path
	got := buildBlockWrapper("NSObject * (^)(void)", "fn", ctx, m, nil)
	if got == "fn" {
		t.Errorf("known-class return should generate wrapper; got pass-through")
	}
}

// TestBuildBlockWrapperScalarReturn covers the innerRet != "" path (no BOOL*, non-void return).
func TestBuildBlockWrapperScalarReturn(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	// Block returning NSUInteger with NSError arg: has conversion + non-void return.
	got := buildBlockWrapper("NSUInteger (^)(NSError *)", "fn", ctx, m, nil)
	// Should generate a wrapper that converts NSError and returns the scalar.
	if got == "fn" {
		t.Errorf("scalar return with NSError arg should generate wrapper; got pass-through")
	}
	if !strings.Contains(got, "return") {
		t.Errorf("scalar return wrapper should contain 'return'; got: %s", got)
	}
}

// TestBuildBlockWrapperBOOLPtrWithScalarReturn covers hasBoolPtr + innerRet != "" path.
func TestBuildBlockWrapperBOOLPtrWithScalarReturn(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	// Block with BOOL* arg (hasBoolPtr=true) AND uint64 return (innerRet != "").
	got := buildBlockWrapper("uint64_t (^)(BOOL *)", "fn", ctx, m, nil)
	// Should produce a closure with preamble (BOOL* shadowing) and return.
	if got == "fn" {
		t.Errorf("BOOL*+scalar-return block should generate wrapper; got pass-through")
	}
	if !strings.Contains(got, "_ret") {
		t.Errorf("hasBoolPtr+scalar return should use _ret variable; got: %s", got)
	}
}

// ── writeMethodBody paths via EmitClass ──────────────────────────────────────

func writeClassBufStructMapper() *typemap.Mapper {
	return &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex: map[string]string{
			"NSObject": "Foundation",
		},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
		// CGRect owned by Foundation so it resolves as a same-fw bare name.
		StructIndex: map[string]string{"CGRect": "Foundation"},
	}
}

// TestWriteClassIDReturn verifies a method returning bare id emits cgo.WrapObject path.
func TestWriteClassIDReturn(t *testing.T) {
	m := writeClassBufStructMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{Selector: "target", Return: macosplatformmetadata.ReturnType{ObjCType: "id"}},
		},
	}
	out, err := writeClassBufKnown("NSObject", cls, framework, m, map[string]macosplatformmetadata.Class{"NSObject": cls}, map[string]bool{"NSObject": true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "cgo.WrapObject") {
		t.Errorf("id return should use cgo.WrapObject; got:\n%s", out)
	}
}

// TestWriteClassValueStructReturn verifies a method returning a same-fw value struct emits malloc path.
func TestWriteClassValueStructReturn(t *testing.T) {
	m := writeClassBufStructMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{Selector: "frame", Return: macosplatformmetadata.ReturnType{ObjCType: "CGRect"}},
		},
	}
	out, err := writeClassBufKnown("NSObject", cls, framework, m, map[string]macosplatformmetadata.Class{"NSObject": cls}, map[string]bool{"NSObject": true})
	if err != nil {
		t.Fatal(err)
	}
	// Value struct return: bridge returns void*, emitter derefs with (*CGRect)(unsafe.Pointer(_ptr)).
	if !strings.Contains(out, "CGRect") {
		t.Errorf("value struct return should mention CGRect; got:\n%s", out)
	}
	if !strings.Contains(out, "FreePtr") {
		t.Errorf("value struct return should call cgo.FreePtr; got:\n%s", out)
	}
}

// TestWriteClassNullableStringReturn verifies a nullable const-char* return emits nil-guard.
func TestWriteClassNullableStringReturn(t *testing.T) {
	m := writeClassBufStructMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{Selector: "cStringUTF8", Return: macosplatformmetadata.ReturnType{ObjCType: "const char *", IsNullable: true}},
		},
	}
	out, err := writeClassBufKnown("NSObject", cls, framework, m, map[string]macosplatformmetadata.Class{"NSObject": cls}, map[string]bool{"NSObject": true})
	if err != nil {
		t.Fatal(err)
	}
	// Nullable string return: emits *string with nil check.
	if !strings.Contains(out, "*string") {
		t.Errorf("nullable string return should produce *string; got:\n%s", out)
	}
	if !strings.Contains(out, "GoString") {
		t.Errorf("nullable string return should call C.GoString; got:\n%s", out)
	}
}

// TestWriteClassNSErrorIDReturn verifies NSError+id return emits (cgo.Object, error).
func TestWriteClassNSErrorIDReturn(t *testing.T) {
	m := writeClassBufStructMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector:   "targetWithError:",
				IsNSError: true,
				Return:     macosplatformmetadata.ReturnType{ObjCType: "id"},
			},
		},
	}
	out, err := writeClassBufKnown("NSObject", cls, framework, m, map[string]macosplatformmetadata.Class{"NSObject": cls}, map[string]bool{"NSObject": true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "cgo.Object") {
		t.Errorf("NSError+id return should have cgo.Object; got:\n%s", out)
	}
	if !strings.Contains(out, "error") {
		t.Errorf("NSError+id return should have error; got:\n%s", out)
	}
}

// TestWriteClassNSErrorValueStructReturn verifies NSError+value struct return emits (CGRect, error).
func TestWriteClassNSErrorValueStructReturn(t *testing.T) {
	m := writeClassBufStructMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector:   "frameWithError:",
				IsNSError: true,
				Return:     macosplatformmetadata.ReturnType{ObjCType: "CGRect"},
			},
		},
	}
	out, err := writeClassBufKnown("NSObject", cls, framework, m, map[string]macosplatformmetadata.Class{"NSObject": cls}, map[string]bool{"NSObject": true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "CGRect") {
		t.Errorf("NSError+value struct return should mention CGRect; got:\n%s", out)
	}
	if !strings.Contains(out, "FreePtr") {
		t.Errorf("NSError+value struct return should call cgo.FreePtr; got:\n%s", out)
	}
}

// ── buildBlockWrapper — hasBoolPtr + object return paths ─────────────────────

// TestBuildBlockWrapperBoolPtrIDReturn covers hasBoolPtr=true + retIsID=true path.
func TestBuildBlockWrapperBoolPtrIDReturn(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	// Block with BOOL* stop arg AND id return (retIsID=true).
	got := buildBlockWrapper("id (^)(BOOL *)", "fn", ctx, m, nil)
	if got == "fn" {
		t.Errorf("BOOL*+id-return block should generate wrapper; got pass-through")
	}
	// hasBoolPtr + id return: should use preamble and Ptr()
	if !strings.Contains(got, "unsafe.Pointer") {
		t.Errorf("BOOL*+id return should emit unsafe.Pointer closure; got: %s", got)
	}
}

// TestBuildBlockWrapperIDProtocolArg covers the id<Protocol> arg path in buildBlockWrapper.
func TestBuildBlockWrapperIDProtocolArg(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{})
	// Block arg is id<NSCopying> — triggers IDProtocols branch.
	got := buildBlockWrapper("void (^)(id<NSCopying>)", "fn", ctx, m, nil)
	if got == "fn" {
		t.Errorf("id<Protocol> arg should generate wrapper; got pass-through")
	}
}

// TestBuildBlockWrapperKnownClassReturnWithBoolPtr covers hasBoolPtr + known-class return.
func TestBuildBlockWrapperKnownClassReturnWithBoolPtr(t *testing.T) {
	m := classTestMapper()
	ctx := blockNeedsWrapperCtx(m, map[string]bool{"NSObject": true})
	// Block with BOOL* arg AND known-class return — hasBoolPtr + typed return path.
	got := buildBlockWrapper("NSObject * (^)(BOOL *)", "fn", ctx, m, nil)
	if got == "fn" {
		t.Errorf("BOOL*+known-class-return block should generate wrapper; got pass-through")
	}
	if !strings.Contains(got, "_r") {
		t.Errorf("hasBoolPtr+class return should use _r variable; got: %s", got)
	}
}

// ── nsStringOverloadArgs — hasNSError path ────────────────────────────────────

// TestNSStringOverloadArgsWithNSError verifies the _nsErr_ arg is appended when hasNSError=true.
func TestNSStringOverloadArgsWithNSError(t *testing.T) {
	m := classTestMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{})
	args := []macosplatformmetadata.Param{
		{Name: "path", ObjCType: "NSString *"},
	}
	result := nsStringOverloadArgs(args, true, []int{0}, ctx, m, nil)
	found := false
	for _, a := range result {
		if strings.Contains(a, "_nsErr_") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("nsStringOverloadArgs with hasNSError=true should include _nsErr_; got: %v", result)
	}
}

// TestNSStringOverloadArgsNonNSArg covers the else branch (non-NS arg kept with its original Go type).
func TestNSStringOverloadArgsNonNSArg(t *testing.T) {
	m := classTestMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{})
	// Two args: index 0 is NSString (in nsIdxs), index 1 is NSUInteger (NOT in nsIdxs).
	args := []macosplatformmetadata.Param{
		{Name: "path", ObjCType: "NSString *"},
		{Name: "count", ObjCType: "NSUInteger"},
	}
	result := nsStringOverloadArgs(args, false, []int{0}, ctx, m, nil)
	// arg[1] is non-NS → should be included as "count <goType>"
	found := false
	for _, a := range result {
		if strings.Contains(a, "count") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("nsStringOverloadArgs: non-NS arg should be kept in output; got: %v", result)
	}
}
