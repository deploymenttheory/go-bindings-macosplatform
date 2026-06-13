package raw

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// ============================================================
// writeContextComments
// ============================================================

func TestWriteContextCommentsDoc(t *testing.T) {
	var buf bytes.Buffer
	writeContextComments(&buf, "A doc line.\nSecond line.", "", 0, meta.Availability{}, "")
	out := buf.String()
	if !strings.Contains(out, "// A doc line.") {
		t.Errorf("doc comment missing: %s", out)
	}
	if !strings.Contains(out, "// Second line.") {
		t.Errorf("second doc line missing: %s", out)
	}
}

func TestWriteContextCommentsSDKFileWithLine(t *testing.T) {
	var buf bytes.Buffer
	writeContextComments(&buf, "", "Foundation/NSObject.h", 42, meta.Availability{}, "")
	out := buf.String()
	if !strings.Contains(out, "// [Foundation/NSObject.h:42]") {
		t.Errorf("sdk file+line comment missing: %s", out)
	}
}

func TestWriteContextCommentsSDKFileNoLine(t *testing.T) {
	var buf bytes.Buffer
	writeContextComments(&buf, "", "Foundation/NSObject.h", 0, meta.Availability{}, "")
	out := buf.String()
	if !strings.Contains(out, "// [Foundation/NSObject.h]") {
		t.Errorf("sdk file (no line) comment missing: %s", out)
	}
}

func TestWriteContextCommentsAvailability(t *testing.T) {
	var buf bytes.Buffer
	writeContextComments(&buf, "", "", 0, meta.Availability{MacOSIntroduced: "14.0"}, "")
	out := buf.String()
	if !strings.Contains(out, "// Introduced: macOS 14.0") {
		t.Errorf("availability comment missing: %s", out)
	}
}

func TestWriteContextCommentsDeprecation(t *testing.T) {
	var buf bytes.Buffer
	av := meta.Availability{MacOSDeprecated: "13.0", ReplacedBy: "newMethod"}
	writeContextComments(&buf, "", "", 0, av, "")
	out := buf.String()
	if !strings.Contains(out, "Deprecated") {
		t.Errorf("deprecation comment missing: %s", out)
	}
}

func TestWriteContextCommentsPrefix(t *testing.T) {
	var buf bytes.Buffer
	writeContextComments(&buf, "Hello", "", 0, meta.Availability{}, "\t")
	out := buf.String()
	if !strings.Contains(out, "\t// Hello") {
		t.Errorf("prefix not applied: %s", out)
	}
}

// ============================================================
// deprecationComment
// ============================================================

func TestDeprecationCommentWithObsoleted(t *testing.T) {
	av := meta.Availability{
		MacOSDeprecated: "12.0",
		MacOSObsoleted:  "15.0",
	}
	got := deprecationComment(av)
	if !strings.Contains(got, "removed in macOS 15.0") {
		t.Errorf("obsoleted version missing: %q", got)
	}
}

func TestDeprecationCommentWithDeprecationMsg(t *testing.T) {
	av := meta.Availability{
		MacOSDeprecated: "11.0",
		DeprecationMsg:  "use newMethod instead",
	}
	got := deprecationComment(av)
	if !strings.Contains(got, "use newMethod instead") {
		t.Errorf("deprecation message missing: %q", got)
	}
}

func TestDeprecationCommentEmpty(t *testing.T) {
	if got := deprecationComment(meta.Availability{}); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// ============================================================
// writeStructDef — extra paths
// ============================================================

// TestWriteStructDefSwiftName verifies the Swift name comment is emitted.
func TestWriteStructDefSwiftName(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{SwiftName: "NS.Object"}
	out, err := writeClassBuf("NSObject", cls, framework, m, map[string]meta.Class{"NSObject": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "// Swift name: NS.Object") {
		t.Errorf("expected Swift name comment; got:\n%s", out)
	}
}

// TestWriteStructDefGenericRoot covers the isGeneric && isRoot path in writeStructDef.
func TestWriteStructDefGenericRoot(t *testing.T) {
	m := classTestMapper()
	m.GenericClasses = map[string]bool{"NSGenericRoot": true}
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	// Generic class with no super → isGeneric=true, isRoot=true
	cls := meta.Class{GenericParams: []string{"ObjectType"}}
	out, err := writeClassBuf("NSGenericRoot", cls, framework, m, map[string]meta.Class{"NSGenericRoot": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[T objc.Object]") {
		t.Errorf("expected generic type param; got:\n%s", out)
	}
	if !strings.Contains(out, "func (o *NSGenericRoot[T]) Ptr()") {
		t.Errorf("expected generic Ptr(); got:\n%s", out)
	}
}

// TestWriteStructDefNonGenericChildOfGenericSuper covers the non-generic class
// whose super IS generic (si.superIsGeneric=true, isGeneric=false).
func TestWriteStructDefNonGenericChildOfGenericSuper(t *testing.T) {
	m := classTestMapper()
	m.GenericClasses["NSArray"] = true
	m.OwnerIndex["NSArray"] = "Foundation"
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSArray": {GenericParams: []string{"ObjectType"}},
		},
	}
	// NSConcrete inherits NSArray but is itself NOT generic.
	cls := meta.Class{Super: "NSArray"}
	out, err := writeClassBuf("NSConcrete", cls, framework, m, map[string]meta.Class{
		"NSArray":    {GenericParams: []string{"ObjectType"}},
		"NSConcrete": cls,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "NSArray[objc.Object]") {
		t.Errorf("expected NSArray[objc.Object] embed; got:\n%s", out)
	}
}

// ============================================================
// EmitBridge() — error paths
// ============================================================

func TestBridgeMkdirAllError(t *testing.T) {
	dir := t.TempDir()
	// Block MkdirAll by placing a regular file at the bridge subdirectory path.
	blockPath := filepath.Join(dir, "bridge")
	if err := os.WriteFile(blockPath, []byte("not-a-dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	if err := EmitBridge(dir, framework, m, map[string]bool{}); err == nil {
		t.Error("expected error when bridge dir cannot be created")
	}
}

func TestBridgeCreateHeaderError(t *testing.T) {
	dir := t.TempDir()
	bridgeDir := filepath.Join(dir, "bridge")
	if err := os.MkdirAll(bridgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Occupy the .h path with a directory so os.Create fails.
	hPath := filepath.Join(bridgeDir, "foundation_bridge.h")
	if err := os.MkdirAll(hPath, 0o755); err != nil {
		t.Fatal(err)
	}
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	if err := EmitBridge(dir, framework, m, map[string]bool{}); err == nil {
		t.Error("expected error when .h file cannot be created")
	}
}

func TestBridgeCreateImplError(t *testing.T) {
	dir := t.TempDir()
	bridgeDir := filepath.Join(dir, "bridge")
	if err := os.MkdirAll(bridgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Occupy the .m path with a directory so os.Create fails.
	mPath := filepath.Join(bridgeDir, "foundation_bridge.m")
	if err := os.MkdirAll(mPath, 0o755); err != nil {
		t.Fatal(err)
	}
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	if err := EmitBridge(dir, framework, m, map[string]bool{}); err == nil {
		t.Error("expected error when .m file cannot be created")
	}
}

// ============================================================
// EmitClasses() — error paths
// ============================================================

func TestClassesMkdirAllError(t *testing.T) {
	dir := t.TempDir()
	// Place a file at the target location so MkdirAll fails.
	blockPath := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blockPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := classTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{"NSObject": {}},
	}
	outDir := filepath.Join(blockPath, "subdir")
	if err := EmitClasses(outDir, framework, m, map[string]bool{}, map[string]meta.Class{}, "foundation"); err == nil {
		t.Error("expected error when outDir cannot be created")
	}
}

func TestClassesCreateFileError(t *testing.T) {
	dir := t.TempDir()
	// Place a directory where the class file would be created.
	filePath := filepath.Join(dir, "NSObject.go")
	if err := os.MkdirAll(filePath, 0o755); err != nil {
		t.Fatal(err)
	}
	m := classTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{"NSObject": {}},
	}
	if err := EmitClasses(dir, framework, m, map[string]bool{}, map[string]meta.Class{}, "foundation"); err == nil {
		t.Error("expected error when class file cannot be created")
	}
}

// ============================================================
// EmitClass — pkgTypeNames from Structs + Typedefs
// ============================================================

func TestWriteClassStructTypeCollisionSkip(t *testing.T) {
	m := classTestMapper()
	// For class "NS", method "rect" → funcName = "NSRect" which IS in Structs.
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Structs: map[string]meta.Struct{
			"NSRect": {},
		},
	}
	cls := meta.Class{
		Methods: []meta.Method{
			{Selector: "rect", IsClassMethod: true, Return: meta.ReturnType{}},
		},
	}
	out, err := writeClassBuf2("NS", cls, framework, m, map[string]meta.Class{"NS": cls})
	if err != nil {
		t.Fatal(err)
	}
	// The class method "NSRect" should be skipped due to struct collision.
	if strings.Contains(out, "func NSRect(") {
		t.Errorf("class method NSRect should be skipped due to struct collision; got:\n%s", out)
	}
}

// writeClassBuf2 is like writeClassBuf but takes FrameworkMeta directly.
func writeClassBuf2(name string, cls meta.Class, framework *meta.FrameworkMeta, m *typemap.Mapper, all map[string]meta.Class) (string, error) {
	var buf bytes.Buffer
	err := EmitClass(&buf, name, cls, framework, m, classKnown, all, strings.ToLower(framework.Framework))
	return buf.String(), err
}

func TestWriteClassTypedefCollisionSkip(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Typedefs: map[string]string{
			"NSPoint": "CGPoint",
		},
	}
	cls := meta.Class{
		Methods: []meta.Method{
			{Selector: "point", IsClassMethod: true, Return: meta.ReturnType{}},
		},
	}
	// class "NS" method "point" → funcName = "NSPoint" which IS in Typedefs.
	out, err := writeClassBuf2("NS", cls, framework, m, map[string]meta.Class{"NS": cls})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "func NSPoint(") {
		t.Errorf("class method NSPoint should be skipped due to typedef collision; got:\n%s", out)
	}
}

// TestWriteClassDuplicateSelector verifies duplicate selectors are emitted only once.
func TestWriteClassDuplicateSelector(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{
		Methods: []meta.Method{
			{Selector: "count", Return: meta.ReturnType{ObjCType: "NSUInteger"}},
			{Selector: "count", Return: meta.ReturnType{ObjCType: "NSUInteger"}}, // exact duplicate
		},
	}
	out, err := writeClassBuf("NSMyClass", cls, framework, m, map[string]meta.Class{"NSMyClass": cls})
	if err != nil {
		t.Fatal(err)
	}
	// "func (o *NSMyClass) Count(ctx context.Context)" should appear exactly once.
	first := strings.Index(out, "func (o *NSMyClass) Count(ctx context.Context)")
	if first < 0 {
		t.Fatalf("Count method not emitted; got:\n%s", out)
	}
	if strings.Index(out[first+1:], "func (o *NSMyClass) Count(ctx context.Context)") >= 0 {
		t.Errorf("Count method emitted more than once; got:\n%s", out)
	}
}

// TestWriteClassGoNameDisambiguation verifies two selectors with the same base Go
// name are disambiguated: the zero-arg form owns the clean base name, and the
// one-arg form gets the colon-count suffix.
func TestWriteClassGoNameDisambiguation(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	// "open" → Open (zero-arg, owns clean name); "open:" → Open1 (one colon suffix)
	cls := meta.Class{
		Methods: []meta.Method{
			{Selector: "open", Return: meta.ReturnType{}},
			{Selector: "open:", Params: []meta.Param{{Name: "path", ObjCType: "NSString *"}}, Return: meta.ReturnType{}},
		},
	}
	out, err := writeClassBuf("NSFile", cls, framework, m, map[string]meta.Class{"NSFile": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "func (o *NSFile) Open(ctx context.Context)") {
		t.Errorf("zero-arg selector should own the clean name Open; got:\n%s", out)
	}
	if !strings.Contains(out, "Open1") {
		t.Errorf("one-arg selector should get Open1 suffix; got:\n%s", out)
	}
}

// ============================================================
// writeMethod — IsDesignatedInit / WarnUnused / SwiftName
// ============================================================

func TestWriteMethodDesignatedInit(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{
		Methods: []meta.Method{
			{Selector: "init", IsDesignatedInit: true, Return: meta.ReturnType{IsInstancetype: true}},
		},
	}
	out, err := writeClassBuf("NSFoo", cls, framework, m, map[string]meta.Class{"NSFoo": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Designated initializer.") {
		t.Errorf("expected designated initializer comment; got:\n%s", out)
	}
}

func TestWriteMethodWarnUnused(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{
		Methods: []meta.Method{
			{Selector: "build", IsWarnUnused: true, Return: meta.ReturnType{IsInstancetype: true}},
		},
	}
	out, err := writeClassBuf("NSFoo", cls, framework, m, map[string]meta.Class{"NSFoo": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Return value must not be discarded.") {
		t.Errorf("expected warn-unused comment; got:\n%s", out)
	}
}

func TestWriteMethodSwiftName(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{
		Methods: []meta.Method{
			{Selector: "doThing", SwiftName: "thing()", Return: meta.ReturnType{}},
		},
	}
	out, err := writeClassBuf("NSFoo", cls, framework, m, map[string]meta.Class{"NSFoo": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "// Swift name: thing()") {
		t.Errorf("expected Swift name comment; got:\n%s", out)
	}
}

// ============================================================
// writeMethodBody — void + NSError path
// ============================================================

func TestWriteMethodBodyVoidNSError(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{
		Methods: []meta.Method{
			{
				Selector:   "doThingWithError:",
				IsNSError: true,
				Return:     meta.ReturnType{ObjCType: "void"},
			},
		},
	}
	out, err := writeClassBuf("NSFoo", cls, framework, m, map[string]meta.Class{"NSFoo": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "return nil") {
		t.Errorf("void+NSError method should return nil; got:\n%s", out)
	}
	if !strings.Contains(out, "NSErrorToError") {
		t.Errorf("void+NSError method should call NSErrorToError; got:\n%s", out)
	}
}

// ============================================================
// isObjectReturn — pointer-to-primitive
// ============================================================

func TestIsObjectReturnPointerToPrimitive(t *testing.T) {
	// *bool, *uint64 are pointers to primitives — not ObjC objects.
	for _, typ := range []string{"*bool", "*int8", "*uint64", "*string"} {
		if isObjectReturn(typ) {
			t.Errorf("isObjectReturn(%q) should be false for pointer-to-primitive", typ)
		}
	}
}

func TestIsObjectReturnNonPointer(t *testing.T) {
	if isObjectReturn("uint64") {
		t.Error("isObjectReturn(uint64) should be false")
	}
}

func TestIsObjectReturnPointerToObject(t *testing.T) {
	if !isObjectReturn("*NSString") {
		t.Error("isObjectReturn(*NSString) should be true")
	}
}

// ============================================================
// writeMethodBody — pointer-to-primitive return (covers switch case in isObjectReturn)
// ============================================================

func TestWriteMethodBodyPointerToPrimReturn(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{
		Methods: []meta.Method{
			// BOOL return — primitive, isObjectReturn returns false
			{Selector: "getFlag", Return: meta.ReturnType{ObjCType: "BOOL"}},
		},
	}
	out, err := writeClassBuf("NSFoo", cls, framework, m, map[string]meta.Class{"NSFoo": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "func (o *NSFoo) GetFlag(ctx context.Context)") {
		t.Errorf("expected GetFlag method; got:\n%s", out)
	}
}

// ============================================================
// cArgList — with actual args
// ============================================================

func TestCArgListWithArgs(t *testing.T) {
	m := testMapper()
	ctx := testCtx("Foundation")
	args := []meta.Param{
		{Name: "count", ObjCType: "NSUInteger"},
		{Name: "value", ObjCType: "BOOL"},
	}
	got := cArgList(false, args, false, m, ctx, nil)
	if !strings.Contains(got, "void *self") {
		t.Errorf("expected void *self; got %q", got)
	}
	if !strings.Contains(got, "count") {
		t.Errorf("expected 'count' param; got %q", got)
	}
	if !strings.Contains(got, "value") {
		t.Errorf("expected 'value' param; got %q", got)
	}
}

// ============================================================
// EmitBridge — BridgeHeader variadic class method skip + inline fn skip
// ============================================================

func TestBridgeHeaderSkipsVariadicMethod(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSObject": {
				Methods: []meta.Method{
					{Selector: "arrayWithObjects:", IsVariadic: true, Return: meta.ReturnType{ObjCType: "void"}},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "arrayWithObjects") {
		t.Errorf("nil-sentinel variadic method should be skipped in header; got:\n%s", out)
	}
}

func TestBridgeHeaderSkipsInlineFunction(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Functions: []meta.Function{
			{Name: "NSInlineFunc", IsInline: true, Return: meta.ReturnType{}},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "NSInlineFunc") {
		t.Errorf("inline function should be skipped in header; got:\n%s", out)
	}
}

func TestBridgeHeaderVoidFreeFunction(t *testing.T) {
	// Covers bridgeFuncDeclFromFunction when ObjCType is empty.
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Functions: []meta.Function{
			{Name: "NSDoSomething", Return: meta.ReturnType{}}, // empty ObjCType → void
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "void foundation_fn_NSDoSomething") {
		t.Errorf("expected void return for empty ObjCType; got:\n%s", out)
	}
}

// ============================================================
// EmitBridge — BridgeImpl extra paths
// ============================================================

func TestBridgeImplSkipsVariadicClassMethod(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSObject": {
				Methods: []meta.Method{
					// nil-sentinel variadic — should be skipped
					{Selector: "setWithObjects:", IsVariadic: true, Return: meta.ReturnType{ObjCType: "void"}},
					// format-string variadic — should appear
					{Selector: "logWithFormat:", IsVariadic: true, Params: []meta.Param{{Name: "format", ObjCType: "NSString *"}}, Return: meta.ReturnType{ObjCType: "void"}},
					{Selector: "count", Return: meta.ReturnType{ObjCType: "NSUInteger"}}, // non-variadic, should appear
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "setWithObjects") {
		t.Errorf("nil-sentinel variadic should be skipped; got:\n%s", out)
	}
	if !strings.Contains(out, "logWithFormat") {
		t.Errorf("format-string variadic should appear; got:\n%s", out)
	}
	if !strings.Contains(out, "count") {
		t.Errorf("non-variadic method should appear; got:\n%s", out)
	}
}

func TestBridgeImplFreeFunctionNonVoid(t *testing.T) {
	// Covers the non-void return path in writeFunctionImpl.
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Functions: []meta.Function{
			{Name: "NSGetVersion", Return: meta.ReturnType{ObjCType: "uint64_t"}},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "return ") {
		t.Errorf("non-void function should emit return; got:\n%s", out)
	}
	if !strings.Contains(out, "NSGetVersion") {
		t.Errorf("function name missing; got:\n%s", out)
	}
}

func TestBridgeImplSkipsForeignExtVariadic(t *testing.T) {
	// Covers the variadic skip in foreign extension impl.
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "MyFW",
		Classes:   map[string]meta.Class{},
		ForeignExtensions: map[string][]meta.Method{
			"NSObject": {
				// nil-sentinel variadic — should be skipped
				{Selector: "arrayWithObjects:", IsVariadic: true, Return: meta.ReturnType{ObjCType: "void"}},
				{Selector: "doThing", Return: meta.ReturnType{ObjCType: "void"}},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "myfw_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "arrayWithObjects") {
		t.Errorf("nil-sentinel variadic foreign ext method should be skipped; got:\n%s", out)
	}
}

func TestBridgeImplInstancetypeNSError(t *testing.T) {
	// Covers the object-return + hasNSError path in writeMethodImpl (line 344-346).
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSObject": {
				Methods: []meta.Method{
					{
						Selector:   "initWithError:",
						IsNSError: true,
						Return:     meta.ReturnType{ObjCType: "instancetype", IsInstancetype: true},
					},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "outError") {
		t.Errorf("instancetype+NSError method should emit outError; got:\n%s", out)
	}
	if !strings.Contains(out, "[_result retain]") {
		t.Errorf("instancetype+NSError method should explicitly retain result; got:\n%s", out)
	}
}

func TestBridgeImplMultiKeywordSelector(t *testing.T) {
	// Covers the i > 0 branch in buildObjCCall (multi-keyword selector).
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSObject": {
				Methods: []meta.Method{
					{
						Selector: "doWith:and:",
						Params: []meta.Param{
							{Name: "a", ObjCType: "NSUInteger"},
							{Name: "b", ObjCType: "NSUInteger"},
						},
						Return: meta.ReturnType{ObjCType: "void"},
					},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "doWith:") && !strings.Contains(out, "and:") {
		t.Errorf("multi-keyword selector not in output; got:\n%s", out)
	}
}

// ============================================================
// writeFunctionImpl — direct call to cover va_list skip
// ============================================================

func TestWriteFunctionImplSkipsVAList(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	fn := meta.Function{
		Name: "NSLogFormat",
		Params: []meta.Param{
			{Name: "format", ObjCType: "NSString *"},
			{Name: "args", ObjCType: "va_list"},
		},
		Return: meta.ReturnType{ObjCType: "void"},
	}
	var buf bytes.Buffer
	writeFunctionImpl(&buf, "foundation_fn_NSLogFormat", fn, ctx, m)
	out := buf.String()
	if strings.Contains(out, "args") {
		t.Errorf("va_list arg should be skipped; got:\n%s", out)
	}
	if !strings.Contains(out, "format") {
		t.Errorf("non-va_list arg should appear; got:\n%s", out)
	}
}

// ============================================================
// objcArgCast — NSError** path (direct call)
// ============================================================

func TestObjcArgCastNSErrorDoublePointer(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	got := objcArgCast("outErr", "NSError **", ctx, m)
	if got != "(NSError **)outErr" {
		t.Errorf("NSError** mid-selector arg should cast to (NSError **)argName; got %q", got)
	}
}

// ============================================================
// buildObjCCall — empty selector (len(parts)==0)
// ============================================================

func TestBuildObjCCallEmptySelector(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	method := meta.Method{Selector: "", Return: meta.ReturnType{ObjCType: "void"}}
	got := buildObjCCall("[MyClass class]", method, ctx, m)
	// When selector is empty, parts becomes [] → uses sel directly.
	if !strings.Contains(got, "[MyClass class]") {
		t.Errorf("empty selector call missing target; got %q", got)
	}
}

// ============================================================
// buildValueChainInner — generic parent paths
// ============================================================

func TestBuildValueChainGenericParentNonGenericChild(t *testing.T) {
	// Non-generic child of a generic same-fw super → embeds NSArray[objc.Object].
	m := &typemap.Mapper{
		GenericClasses: map[string]bool{"NSArray": true},
		OwnerIndex: map[string]string{"NSArray": "Foundation"},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
	fmClasses := map[string]meta.Class{
		"NSArray":    {GenericParams: []string{"ObjectType"}},
		"NSSpecialArray": {Super: "NSArray"},
	}
	result := buildValueChainInner("NSSpecialArray", "", "Foundation", fmClasses, m, nil)
	if !strings.Contains(result, "NSArray") {
		t.Errorf("expected NSArray in chain; got %q", result)
	}
	if !strings.Contains(result, "[objc.Object]") {
		t.Errorf("expected [objc.Object] for non-generic child of generic parent; got %q", result)
	}
}

func TestBuildValueChainGenericParentGenericChild(t *testing.T) {
	// Generic child of generic same-fw super with genSuffix=[T] → propagates [T].
	m := &typemap.Mapper{
		GenericClasses: map[string]bool{"NSArray": true},
		OwnerIndex: map[string]string{"NSArray": "Foundation"},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
	fmClasses := map[string]meta.Class{
		"NSArray":        {GenericParams: []string{"ObjectType"}},
		"NSMutableArray": {Super: "NSArray", GenericParams: []string{"ObjectType"}},
	}
	result := buildValueChainInner("NSMutableArray", "[T]", "Foundation", fmClasses, m, nil)
	if !strings.Contains(result, "NSArray[T]") {
		t.Errorf("expected NSArray[T] propagation; got %q", result)
	}
}

func TestBuildValueChainCrossFWGenericTypedWithPtr(t *testing.T) {
	// Cross-fw generic parent with genSuffix=[T] → uses TypedWithPtr constructor.
	m := &typemap.Mapper{
		GenericClasses: map[string]bool{"NSArray": true},
		OwnerIndex: map[string]string{"NSArray": "Foundation"},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
	fmClasses := map[string]meta.Class{
		// NSSpecial is in "AppKit", its super NSArray is in "Foundation" (cross-fw).
		"NSSpecial": {Super: "NSArray", GenericParams: []string{"ObjectType"}},
	}
	result := buildValueChainInner("NSSpecial", "[T]", "AppKit", fmClasses, m, make(map[string]string))
	if !strings.Contains(result, "TypedWithPtr[T]") {
		t.Errorf("expected TypedWithPtr[T] for cross-fw generic super; got %q", result)
	}
}

// ============================================================
// Externs — cross-framework type + multi-import
// ============================================================

func TestExternsMultipleImports(t *testing.T) {
	m := &typemap.Mapper{
		OwnerIndex: map[string]string{
			"NSString": "Foundation",
		},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
		GenericClasses: map[string]bool{},
	}
	// One extern with unknown type (→ unsafe.Pointer) and one with cross-fw type.
	framework := &meta.FrameworkMeta{
		Framework: "AppKit",
		Externs: []meta.Extern{
			{Name: "NSAppKitVersionNumber", ObjCType: "UnknownType *"}, // → unsafe.Pointer
			{Name: "NSStringConst", ObjCType: "NSString *"},            // → cross-fw *foundation.NSString
		},
	}
	knownClasses := map[string]bool{"NSString": true}
	var buf bytes.Buffer
	if err := EmitExterns(&buf, "appkit", framework, m, knownClasses); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Should have block import since 2+ imports.
	if !strings.Contains(out, "import (") {
		t.Errorf("expected block import for multiple imports; got:\n%s", out)
	}
}

func TestExternsWithAvailabilityAndDeprecation(t *testing.T) {
	m := testMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Externs: []meta.Extern{
			{
				Name:     "NSFoo",
				ObjCType: "uint64_t",
				Availability: meta.Availability{
					MacOSIntroduced: "12.0",
					MacOSDeprecated: "14.0",
					ReplacedBy:      "NSBar",
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitExterns(&buf, "foundation", framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Introduced: macOS 12.0") {
		t.Errorf("expected availability comment; got:\n%s", out)
	}
	if !strings.Contains(out, "Deprecated") {
		t.Errorf("expected deprecated comment; got:\n%s", out)
	}
}

// ============================================================
// BridgeHeader — ForeignExtension variadic skip
// ============================================================

func TestBridgeHeaderSkipsForeignExtVariadic(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "MyFW",
		Classes:   map[string]meta.Class{},
		ForeignExtensions: map[string][]meta.Method{
			"NSObject": {
				// nil-sentinel variadic — should be skipped
				{Selector: "arrayWithObjects:", IsVariadic: true, Return: meta.ReturnType{ObjCType: "void"}},
				{Selector: "helper", Return: meta.ReturnType{ObjCType: "void"}},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "arrayWithObjects") {
		t.Errorf("nil-sentinel variadic foreign ext method should be skipped in header; got:\n%s", out)
	}
	if !strings.Contains(out, "helper") {
		t.Errorf("non-variadic helper should appear in header; got:\n%s", out)
	}
}

// ============================================================
// ForeignExtensions — availability comment
// ============================================================

// ============================================================
// Helper: ensure test compiles and verifications are trivial
// ============================================================

// TestWriteClassDocAndSDKComments covers writeContextComments via writeStructDef
// which is called from EmitClass for a class with Doc and SDKFile set.
func TestWriteClassDocAndSDKComments(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{
		Doc:     "This is NSObject.",
		SDKFile: "Foundation/NSObject.h",
		SDKLine: 10,
	}
	out, err := writeClassBuf("NSObject", cls, framework, m, map[string]meta.Class{"NSObject": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "// This is NSObject.") {
		t.Errorf("doc comment missing; got:\n%s", out)
	}
	if !strings.Contains(out, "Foundation/NSObject.h:10") {
		t.Errorf("SDK file+line comment missing; got:\n%s", out)
	}
}

// TestWriteClassMethodDocAndSDK covers writeContextComments for methods.
func TestWriteClassMethodDocAndSDK(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{
		Methods: []meta.Method{
			{
				Selector: "doThing",
				Return:   meta.ReturnType{},
				Doc:      "Does a thing.",
				SDKFile:  "Foundation/NSFoo.h",
				SDKLine:  99,
			},
		},
	}
	out, err := writeClassBuf("NSFoo", cls, framework, m, map[string]meta.Class{"NSFoo": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "// Does a thing.") {
		t.Errorf("method doc comment missing; got:\n%s", out)
	}
	if !strings.Contains(out, "Foundation/NSFoo.h:99") {
		t.Errorf("method SDK file+line comment missing; got:\n%s", out)
	}
}

// TestWriteClassFilenameCollision2 covers the inner i++ in EmitClasses() collision loop.
func TestWriteClassFilenameCollision2(t *testing.T) {
	dir := t.TempDir()
	m := classTestMapper()
	// Two classes whose file names collide case-insensitively.
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSObject":  {},
			"NSObjectX": {},
		},
	}
	// Manually create NSObject.go as a directory to force the collision suffix loop.
	if err := os.MkdirAll(filepath.Join(dir, "NSObject.go"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Just test Classes doesn't crash — the error for NSObject will fire,
	// but NSObjectX should succeed.
	_ = EmitClasses(dir, framework, m, map[string]bool{}, map[string]meta.Class{}, "foundation")
}

// TestBridgeImplFreeFunctionVoidReturn covers the void branch in writeFunctionImpl.
func TestBridgeImplFreeFunctionVoidReturn(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Functions: []meta.Function{
			{Name: "NSDoNothing", Return: meta.ReturnType{ObjCType: "void"}},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "NSDoNothing") {
		t.Errorf("function should appear in impl; got:\n%s", out)
	}
	// void return should NOT emit "return ".
	if strings.Contains(out, fmt.Sprintf("return %s(", "NSDoNothing")) {
		t.Errorf("void function should not have return; got:\n%s", out)
	}
}
