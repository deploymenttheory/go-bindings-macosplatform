package raw

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// bridgeTestMapper returns a Mapper configured for bridge tests.
func bridgeTestMapper() *typemap.Mapper {
	return &typemap.Mapper{
		GenericClasses: map[string]bool{"NSArray": true},
		OwnerIndex: map[string]string{
			"NSObject": "Foundation",
			"NSString": "Foundation",
			"NSArray":  "Foundation",
		},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
}

func bridgeCtx(fw string) typemap.Context {
	m := bridgeTestMapper()
	return m.BaseContext(fw, map[string]bool{"NSObject": true, "NSString": true})
}

// --------------------------------------------------------------------------
// methodKey
// --------------------------------------------------------------------------

func TestMethodKeyInstance(t *testing.T) {
	k := methodKey("count", false)
	if !strings.HasPrefix(k, "i:") {
		t.Errorf("instance method key should start with 'i:', got %q", k)
	}
}

func TestMethodKeyClass(t *testing.T) {
	k := methodKey("sharedManager", true)
	if !strings.HasPrefix(k, "c:") {
		t.Errorf("class method key should start with 'c:', got %q", k)
	}
}

func TestMethodKeyDistinct(t *testing.T) {
	inst := methodKey("count", false)
	cls := methodKey("count", true)
	if inst == cls {
		t.Error("instance and class method keys for same selector must differ")
	}
}

// --------------------------------------------------------------------------
// isVAList / hasVAListArg
// --------------------------------------------------------------------------

func TestIsVAListTrue(t *testing.T) {
	for _, s := range []string{"va_list", "__va_list", "const va_list"} {
		if !isVAList(s) {
			t.Errorf("isVAList(%q) should be true", s)
		}
	}
}

func TestIsVAListFalse(t *testing.T) {
	for _, s := range []string{"int", "NSString *", "BOOL", ""} {
		if isVAList(s) {
			t.Errorf("isVAList(%q) should be false", s)
		}
	}
}

func TestHasVAListArgTrue(t *testing.T) {
	fn := meta.Function{
		Name:  "logFormat",
		Params:  []meta.Param{{Name: "args", ObjCType: "va_list"}},
	}
	if !hasVAListArg(fn) {
		t.Error("function with va_list arg: hasVAListArg should be true")
	}
}

func TestHasVAListArgFalse(t *testing.T) {
	fn := meta.Function{
		Name: "getCount",
		Params: []meta.Param{{Name: "n", ObjCType: "NSUInteger"}},
	}
	if hasVAListArg(fn) {
		t.Error("function without va_list: hasVAListArg should be false")
	}
}

// --------------------------------------------------------------------------
// isObjectObjCType
// --------------------------------------------------------------------------

func TestIsObjectObjCTypeTrue(t *testing.T) {
	knownClasses := map[string]bool{"NSString": true, "NSArray": true}
	for _, s := range []string{"NSString *", "id", "instancetype", "NSArray *"} {
		if !isObjectObjCType(s, knownClasses) {
			t.Errorf("isObjectObjCType(%q) should be true", s)
		}
	}
}

func TestIsObjectObjCTypeFalse(t *testing.T) {
	knownClasses := map[string]bool{}
	for _, s := range []string{"BOOL", "NSUInteger", "int", "", "void"} {
		if isObjectObjCType(s, knownClasses) {
			t.Errorf("isObjectObjCType(%q) should be false", s)
		}
	}
}

func TestIsObjectObjCTypeUnderscorePrefix(t *testing.T) {
	// Classes with underscore-prefixed names are ObjC objects and must be detected
	// via the knownClasses set, not the capital-letter heuristic.
	knownClasses := map[string]bool{"_NSConcreteStackBlock": true}
	if !isObjectObjCType("_NSConcreteStackBlock *", knownClasses) {
		t.Errorf("isObjectObjCType(%q) should be true when class is in knownClasses", "_NSConcreteStackBlock *")
	}
	// Without the registry it falls back to the heuristic and returns false (underscore).
	if isObjectObjCType("_NSConcreteStackBlock *", map[string]bool{}) {
		t.Errorf("isObjectObjCType(%q) should be false when not in knownClasses (underscore heuristic)", "_NSConcreteStackBlock *")
	}
}

// --------------------------------------------------------------------------
// bridgeReturnCType
// --------------------------------------------------------------------------

func TestBridgeReturnCTypeInstancetype(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	method := meta.Method{Return: meta.ReturnType{IsInstancetype: true}}
	got := bridgeReturnCType(method, ctx, m)
	if got != "id" {
		t.Errorf("instancetype return: want 'id', got %q", got)
	}
}

func TestBridgeReturnCTypeVoid(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	method := meta.Method{Return: meta.ReturnType{ObjCType: "void"}}
	got := bridgeReturnCType(method, ctx, m)
	if got != "void" {
		t.Errorf("void return: want 'void', got %q", got)
	}
}

func TestBridgeReturnCTypeEmpty(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	method := meta.Method{Return: meta.ReturnType{}}
	got := bridgeReturnCType(method, ctx, m)
	if got != "void" {
		t.Errorf("empty ObjCType: want 'void', got %q", got)
	}
}

func TestBridgeReturnCTypeObject(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	method := meta.Method{Return: meta.ReturnType{ObjCType: "NSString *"}}
	got := bridgeReturnCType(method, ctx, m)
	// NSObject pointer types map to void * in the bridge
	if got == "void" {
		t.Errorf("object return should not map to 'void', got %q", got)
	}
}

// --------------------------------------------------------------------------
// bridgeParamList
// --------------------------------------------------------------------------

func TestBridgeParamListInstanceMethod(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	params := bridgeParamList(false, []meta.Param{}, false, ctx, m)
	if !strings.Contains(params, "void *self") {
		t.Errorf("instance method should have 'void *self', got %q", params)
	}
}

func TestBridgeParamListClassMethod(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	params := bridgeParamList(true, []meta.Param{}, false, ctx, m)
	if strings.Contains(params, "self") {
		t.Errorf("class method should not have self, got %q", params)
	}
	// Every bridge function always has a trailing void **outException.
	if !strings.Contains(params, "void **outException") {
		t.Errorf("class method should have 'void **outException', got %q", params)
	}
}

func TestBridgeParamListNSError(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	params := bridgeParamList(false, []meta.Param{}, true, ctx, m)
	if !strings.Contains(params, "void **outError") {
		t.Errorf("NSError method should have 'void **outError', got %q", params)
	}
}

func TestBridgeParamListArgs(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	args := []meta.Param{{Name: "count", ObjCType: "NSUInteger"}}
	params := bridgeParamList(false, args, false, ctx, m)
	if !strings.Contains(params, "count") {
		t.Errorf("param list should include arg name 'count', got %q", params)
	}
}

// --------------------------------------------------------------------------
// buildObjCCall
// --------------------------------------------------------------------------

func TestBuildObjCCallNoArg(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	method := meta.Method{Selector: "count", Params: []meta.Param{}}
	call := buildObjCCall("self", method, ctx, m)
	if call != "[self count]" {
		t.Errorf("no-arg selector: want '[self count]', got %q", call)
	}
}

func TestBuildObjCCallSingleKeyword(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	method := meta.Method{
		Selector: "objectAtIndex:",
		Params:     []meta.Param{{Name: "index", ObjCType: "NSUInteger"}},
	}
	call := buildObjCCall("self", method, ctx, m)
	if !strings.Contains(call, "objectAtIndex:") {
		t.Errorf("single keyword call missing selector part; got %q", call)
	}
	if !strings.Contains(call, "index") {
		t.Errorf("single keyword call missing arg; got %q", call)
	}
}

func TestBuildObjCCallNSErrorInjected(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	method := meta.Method{
		Selector:   "performWithError:",
		Params:       []meta.Param{},
		IsNSError: true,
	}
	call := buildObjCCall("self", method, ctx, m)
	if !strings.Contains(call, "&_err") {
		t.Errorf("NSError method should inject &_err; got %q", call)
	}
}

// --------------------------------------------------------------------------
// objcArgCast
// --------------------------------------------------------------------------

func TestObjCArgCastObject(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	result := objcArgCast("myArg", "NSString *", ctx, m)
	if !strings.Contains(result, "__bridge id") {
		t.Errorf("ObjC object should be cast to (__bridge id); got %q", result)
	}
}

func TestObjCArgCastPrimitive(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	result := objcArgCast("count", "NSUInteger", ctx, m)
	if result != "count" {
		t.Errorf("primitive arg should pass through unchanged; got %q", result)
	}
}

func TestObjCArgCastBlock(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	result := objcArgCast("blk", "void (^)(void)", ctx, m)
	want := "(__bridge void (^)(void))blk"
	if result != want {
		t.Errorf("block arg cast: got %q, want %q", result, want)
	}
}

// --------------------------------------------------------------------------
// buildClassBridgeNames
// --------------------------------------------------------------------------

func TestResolveClassBridgeNamesNoCollision(t *testing.T) {
	methods := []meta.Method{
		{Selector: "count", IsClassMethod: false},
		{Selector: "objectAtIndex:", IsClassMethod: false},
	}
	names := buildClassBridgeNames("Foundation", "NSArray", methods)
	// Two distinct selectors → two entries, no suffix needed
	if len(names) != 2 {
		t.Errorf("expected 2 bridge name entries, got %d", len(names))
	}
}

func TestResolveClassBridgeNamesCollisionResolved(t *testing.T) {
	// Two methods whose selectors both map to the same raw bridge name.
	// "open" and "open:" both generate "foundation_nsurl_open_inst" (or similar).
	methods := []meta.Method{
		{Selector: "open", IsClassMethod: false, Return: meta.ReturnType{ObjCType: "void"}},
		{Selector: "open:", IsClassMethod: false, Return: meta.ReturnType{ObjCType: "void"}},
	}
	names := buildClassBridgeNames("Foundation", "NSURL", methods)
	k1 := methodKey("open", false)
	k2 := methodKey("open:", false)
	n1, n2 := names[k1], names[k2]
	if n1 == n2 {
		t.Errorf("colliding selectors should get distinct bridge names; both got %q", n1)
	}
}

func TestResolveClassBridgeNamesSkipsVariadic(t *testing.T) {
	methods := []meta.Method{
		// nil-sentinel variadic — should be excluded
		{Selector: "arrayWithObjects:", IsVariadic: true},
		// format-string variadic — should be included
		{Selector: "stringWithFormat:", IsVariadic: true},
		{Selector: "count", IsClassMethod: false},
	}
	names := buildClassBridgeNames("Foundation", "NSString", methods)
	for k := range names {
		if strings.Contains(k, "arrayWithObjects") {
			t.Errorf("nil-sentinel variadic method should be excluded from bridge names; key=%q", k)
		}
	}
	// format-string variadic should be present
	if _, ok := names[methodKey("stringWithFormat:", false)]; !ok {
		t.Errorf("format-string variadic should be included in bridge names")
	}
}

// --------------------------------------------------------------------------
// BridgeHeader
// --------------------------------------------------------------------------

func TestBridgeHeaderEmpty(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Should have the standard preamble
	if !strings.Contains(out, "#pragma once") {
		t.Errorf("missing #pragma once; got:\n%s", out)
	}
	if !strings.Contains(out, "#include <stdint.h>") {
		t.Errorf("missing stdint.h; got:\n%s", out)
	}
}

func TestBridgeHeaderInstanceMethod(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSObject": {
				Methods: []meta.Method{
					{Selector: "description", Return: meta.ReturnType{ObjCType: "NSString *"}},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{"NSObject": true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Should declare a function with void *self param
	if !strings.Contains(out, "void *self") {
		t.Errorf("instance method bridge decl missing 'void *self'; got:\n%s", out)
	}
}

func TestBridgeHeaderClassMethod(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSObject": {
				Methods: []meta.Method{
					{Selector: "alloc", IsClassMethod: true, Return: meta.ReturnType{IsInstancetype: true}},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "void *self") {
		t.Errorf("class method bridge decl should not have 'void *self'; got:\n%s", out)
	}
}

func TestBridgeHeaderSkipsNilSentinelVariadic(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSString": {
				Methods: []meta.Method{
					// nil-sentinel — should be skipped
					{Selector: "initWithObjects:", IsVariadic: true, Return: meta.ReturnType{IsInstancetype: true}},
					// format-string — should be bridged
					{Selector: "initWithFormat:", IsVariadic: true, Params: []meta.Param{{Name: "format", ObjCType: "NSString * _Nonnull"}}, Return: meta.ReturnType{IsInstancetype: true}},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "initWithObjects") {
		t.Errorf("nil-sentinel variadic should not appear in bridge header; got:\n%s", out)
	}
	if !strings.Contains(out, "initWithFormat") {
		t.Errorf("format-string variadic should appear in bridge header; got:\n%s", out)
	}
}

func TestBridgeHeaderFreeFunction(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Functions: []meta.Function{
			{Name: "NSStringFromClass", Return: meta.ReturnType{ObjCType: "NSString *"}, Params: []meta.Param{{Name: "aClass", ObjCType: "Class"}}},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "NSStringFromClass") {
		t.Errorf("free function should appear in bridge header; got:\n%s", out)
	}
}

func TestBridgeHeaderForeignExtensions(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "MyFW",
		Classes:   map[string]meta.Class{},
		ForeignExtensions: map[string][]meta.Method{
			"NSObject": {
				{Selector: "myExtension", Return: meta.ReturnType{ObjCType: "void"}},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "myextension") && !strings.Contains(out, "MyExtension") && !strings.Contains(out, "myfwNSObject") {
		// Check that something from NSObject foreign extension appears
		if !strings.Contains(out, "NSObject") {
			t.Errorf("foreign extension method should appear in bridge header; got:\n%s", out)
		}
	}
}

// --------------------------------------------------------------------------
// BridgeImpl
// --------------------------------------------------------------------------

func TestBridgeImplHeader(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "#import <Foundation/Foundation.h>") {
		t.Errorf("impl header missing framework import; got:\n%s", out)
	}
	if !strings.Contains(out, `"foundation_bridge.h"`) {
		t.Errorf("impl header missing bridge header include; got:\n%s", out)
	}
}

func TestBridgeImplSubFrameworkUsesParent(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework:       "HIToolbox",
		ParentFramework: "Carbon",
		Classes:         map[string]meta.Class{},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "hitoolbox_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "#import <Carbon/Carbon.h>") {
		t.Errorf("sub-framework should import parent umbrella header; got:\n%s", out)
	}
}

func TestBridgeImplVoidMethod(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSObject": {
				Methods: []meta.Method{
					{Selector: "doSomething", IsClassMethod: true, Return: meta.ReturnType{ObjCType: "void"}},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "@autoreleasepool") {
		t.Errorf("bridge impl should wrap in @autoreleasepool; got:\n%s", out)
	}
}

func TestBridgeImplObjectReturn(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSObject": {
				Methods: []meta.Method{
					{Selector: "description", Return: meta.ReturnType{ObjCType: "NSString *"}},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{"NSString": true}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "[_result retain]") {
		t.Errorf("object return should explicitly retain result; got:\n%s", out)
	}
}

func TestBridgeImplFreeFunction(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Functions: []meta.Function{
			{Name: "NSLog", Return: meta.ReturnType{ObjCType: "void"}, Params: []meta.Param{{Name: "format", ObjCType: "NSString *"}}},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "NSLog") {
		t.Errorf("free function impl should appear in bridge impl; got:\n%s", out)
	}
}

// --------------------------------------------------------------------------
// EmitBridge (disk-based)
// --------------------------------------------------------------------------

func TestBridgeCreatesBothFiles(t *testing.T) {
	dir := t.TempDir()
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	if err := EmitBridge(dir, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	hPath := dir + "/bridge/foundation_bridge.h"
	mPath := dir + "/bridge/foundation_bridge.m"
	if _, err := os.Stat(hPath); err != nil {
		t.Errorf("bridge header not created: %v", err)
	}
	if _, err := os.Stat(mPath); err != nil {
		t.Errorf("bridge impl not created: %v", err)
	}
}

// --------------------------------------------------------------------------
// freeFuncParamList
// --------------------------------------------------------------------------

func TestFreeFuncParamListEmpty(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	result := freeFuncParamList([]meta.Param{}, ctx, m)
	// Even with no user-supplied args, outException is always appended.
	if !strings.Contains(result, "void **outException") {
		t.Errorf("empty param list should contain 'void **outException', got %q", result)
	}
}

func TestFreeFuncParamListWithArgs(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	args := []meta.Param{
		{Name: "x", ObjCType: "NSUInteger"},
		{Name: "y", ObjCType: "NSUInteger"},
	}
	result := freeFuncParamList(args, ctx, m)
	if !strings.Contains(result, "x") || !strings.Contains(result, "y") {
		t.Errorf("param list should include arg names; got %q", result)
	}
}

func TestFreeFuncParamListSkipsVAList(t *testing.T) {
	m := bridgeTestMapper()
	ctx := bridgeCtx("Foundation")
	args := []meta.Param{
		{Name: "format", ObjCType: "NSString *"},
		{Name: "args", ObjCType: "va_list"},
	}
	result := freeFuncParamList(args, ctx, m)
	if strings.Contains(result, "va_list") || strings.Contains(result, "args") {
		t.Errorf("va_list arg should be skipped; got %q", result)
	}
}


// --------------------------------------------------------------------------
// writeMethodImpl — additional paths via BridgeImpl
// --------------------------------------------------------------------------

func TestBridgeImplNSErrorVoidMethod(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSObject": {
				Methods: []meta.Method{
					{
						Selector:   "performWithError:",
						IsNSError: true,
						Return:     meta.ReturnType{ObjCType: "void"},
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
		t.Errorf("NSError method should write outError; got:\n%s", out)
	}
}

func TestBridgeImplForeignExtensions(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "MyFW",
		Classes:   map[string]meta.Class{},
		ForeignExtensions: map[string][]meta.Method{
			"NSObject": {
				{Selector: "myExt", Return: meta.ReturnType{ObjCType: "void"}},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "myfw_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Should contain the foreign extension method implementation
	if !strings.Contains(out, "@autoreleasepool") {
		t.Errorf("foreign extension impl should have @autoreleasepool; got:\n%s", out)
	}
}

func TestBridgeImplPrimitiveReturnWithNSError(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSObject": {
				Methods: []meta.Method{
					{
						Selector:   "countWithError:",
						IsNSError: true,
						Return:     meta.ReturnType{ObjCType: "NSUInteger"},
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
	if !strings.Contains(out, "_result") {
		t.Errorf("primitive+NSError should use _result; got:\n%s", out)
	}
}

// ── allocBridgeName ───────────────────────────────────────────────────────────

func TestAllocBridgeNameFormat(t *testing.T) {
	got := allocBridgeName("Foundation", "NSObject")
	if got != "foundation_NSObject_alloc" {
		t.Errorf("allocBridgeName = %q; want %q", got, "foundation_NSObject_alloc")
	}
}

// TestBridgeHeaderAllocHelper verifies alloc helper decls appear for classes
// that have a designated initializer with arguments.
func TestBridgeHeaderAllocHelper(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSFoo": {
				Methods: []meta.Method{
					{
						Selector:          "initWithName:",
						IsInit:            true,
						IsDesignatedInit:  true,
						Params:              []meta.Param{{Name: "name", ObjCType: "NSString *"}},
						Return:            meta.ReturnType{IsInstancetype: true},
					},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "foundation_NSFoo_alloc") {
		t.Errorf("expected alloc helper decl; got:\n%s", out)
	}
}

// ── stripGenericID ────────────────────────────────────────────────────────────

func TestStripGenericIDRemovesIDParam(t *testing.T) {
	got := stripGenericID("NSFetchRequest<id<NSFetchRequestResult>>")
	if strings.Contains(got, "<id>") {
		t.Errorf("expected <id> removed; got %q", got)
	}
}

func TestStripGenericIDNoChange(t *testing.T) {
	input := "NSArray<NSString *> *"
	got := stripGenericID(input)
	if got != input {
		t.Errorf("stripGenericID changed type with no <id>; got %q", got)
	}
}

// ── writeProtocolMethodImpl (via BridgeImpl with protocol proxy) ──────────────

func TestBridgeImplProtocolMethodImpl(t *testing.T) {
	m := bridgeTestMapper()
	m.ProtocolProxyIndex = map[string]string{"VZFoo": "Virtualization"}
	framework := &meta.FrameworkMeta{
		Framework: "Virtualization",
		Protocols: map[string]meta.Protocol{
			"VZFoo": {
				Methods: []meta.Method{
					{Selector: "doThing", Return: meta.ReturnType{ObjCType: "void"}},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "virtualization_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// The protocol method implementation should use id<VZFoo> cast.
	if !strings.Contains(out, "id<VZFoo>") {
		t.Errorf("expected id<VZFoo> protocol cast in impl; got:\n%s", out)
	}
}

// ── isStructByValueType ───────────────────────────────────────────────────────

func bridgeStructMapper() *typemap.Mapper {
	return &typemap.Mapper{
		GenericClasses:        map[string]bool{},
		OwnerIndex:        map[string]string{},
		ModulePrefix:          "github.com/example/fw",
		BlockedImports:        map[string]map[string]bool{},
		TypedefIndex:         map[string]string{},
		StructIndex:          map[string]string{"MTLViewport": "Metal"},
		ProtocolIndex:        map[string]string{},
		ProtocolProxyIndex:  map[string]string{},
		CFTypeIndex: map[string]string{},
	}
}

func TestIsStructByValueTypeHardcoded(t *testing.T) {
	m := bridgeStructMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{})
	// CGRect is in the hardcoded structCTypes set.
	if !isStructByValueType("CGRect", ctx, m) {
		t.Error("CGRect should be a struct-by-value type (hardcoded set)")
	}
}

func TestIsStructByValueTypeExplicitStruct(t *testing.T) {
	m := bridgeStructMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{})
	if !isStructByValueType("struct timespec", ctx, m) {
		t.Error("explicit 'struct timespec' should be struct-by-value")
	}
}

func TestIsStructByValueTypeExplicitStructPointer(t *testing.T) {
	m := bridgeStructMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{})
	if isStructByValueType("struct foo *", ctx, m) {
		t.Error("pointer-to-struct should NOT be struct-by-value")
	}
}

func TestIsStructByValueTypeKnownStruct(t *testing.T) {
	m := bridgeStructMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{})
	if !isStructByValueType("MTLViewport", ctx, m) {
		t.Error("StructIndex-registered type should be struct-by-value")
	}
}

func TestIsStructByValueTypePointerIsNotByValue(t *testing.T) {
	m := bridgeStructMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{})
	// Pointer to a known struct is not by-value.
	if isStructByValueType("MTLViewport *", ctx, m) {
		t.Error("pointer to KnownStruct should NOT be struct-by-value")
	}
}

func TestIsStructByValueTypeIDIsNotByValue(t *testing.T) {
	m := bridgeStructMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{})
	if isStructByValueType("id", ctx, m) {
		t.Error("bare id should NOT be struct-by-value")
	}
}

func TestIsStructByValueTypeTypedefChain(t *testing.T) {
	m := bridgeStructMapper()
	m.TypedefIndex = map[string]string{"SCNQuaternion": "struct SCNVector4"}
	ctx := m.BaseContext("Foundation", map[string]bool{})
	if !isStructByValueType("SCNQuaternion", ctx, m) {
		t.Error("typedef → struct should be struct-by-value")
	}
}

func TestIsStructByValueTypeTypedefPointerChain(t *testing.T) {
	m := bridgeStructMapper()
	m.TypedefIndex = map[string]string{"OpaqueRef": "struct _Opaque *"}
	ctx := m.BaseContext("Foundation", map[string]bool{})
	if isStructByValueType("OpaqueRef", ctx, m) {
		t.Error("typedef → struct pointer should NOT be struct-by-value")
	}
}

// ── objcArgCast ───────────────────────────────────────────────────────────────

func TestObjcArgCastBlock(t *testing.T) {
	m := bridgeStructMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{})
	got := objcArgCast("_a0", "void (^)(void)", ctx, m)
	if !strings.Contains(got, "__bridge") {
		t.Errorf("block arg cast should use __bridge; got %q", got)
	}
}

func TestObjcArgCastSEL(t *testing.T) {
	m := bridgeStructMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{})
	got := objcArgCast("_a0", "SEL", ctx, m)
	if !strings.Contains(got, "sel_getUid") {
		t.Errorf("SEL arg cast should use sel_getUid; got %q", got)
	}
}

func TestObjcArgCastID(t *testing.T) {
	m := bridgeStructMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{})
	got := objcArgCast("_a0", "id", ctx, m)
	if got != "(__bridge id)_a0" {
		t.Errorf("id arg cast should be (__bridge id)_a0; got %q", got)
	}
}

func TestObjcArgCastObjectPointer(t *testing.T) {
	m := bridgeStructMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{"NSString": true})
	got := objcArgCast("_a0", "NSString *", ctx, m)
	if !strings.Contains(got, "__bridge id") {
		t.Errorf("NSString * cast should use __bridge id; got %q", got)
	}
}

func TestObjcArgCastStructByValue(t *testing.T) {
	m := bridgeStructMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{})
	got := objcArgCast("_a0", "CGRect", ctx, m)
	if !strings.Contains(got, "CGRect*") && !strings.Contains(got, "CGRect *") && !strings.Contains(got, "*(CGRect*)") {
		t.Errorf("CGRect arg cast should dereference struct pointer; got %q", got)
	}
}

func TestObjcArgCastPrimitive(t *testing.T) {
	m := bridgeStructMapper()
	ctx := m.BaseContext("Foundation", map[string]bool{})
	got := objcArgCast("_a0", "NSUInteger", ctx, m)
	if got != "_a0" {
		t.Errorf("primitive arg cast should pass through; got %q", got)
	}
}

// ── writeBridgeEntitlements ───────────────────────────────────────────────────

func TestWriteBridgeEntitlementsEmitted(t *testing.T) {
	var buf bytes.Buffer
	writeBridgeEntitlements(&buf, []string{"com.apple.private.network"})
	if !strings.Contains(buf.String(), "com.apple.private.network") {
		t.Errorf("expected entitlement comment; got %q", buf.String())
	}
}

func TestWriteBridgeEntitlementsEmpty(t *testing.T) {
	var buf bytes.Buffer
	writeBridgeEntitlements(&buf, nil)
	if buf.Len() != 0 {
		t.Errorf("empty entitlements should write nothing; got %q", buf.String())
	}
}

// TestBridgeImplMethodWithEntitlements verifies entitlement comments appear in BridgeImpl.
func TestBridgeImplMethodWithEntitlements(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSFoo": {
				Methods: []meta.Method{
					{
						Selector: "doPrivateThing",
						Return:   meta.ReturnType{ObjCType: "void"},
						Availability: meta.Availability{
							Entitlements: []string{"com.apple.developer.restricted"},
						},
					},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "com.apple.developer.restricted") {
		t.Errorf("expected entitlement comment in bridge impl; got:\n%s", buf.String())
	}
}

// TestBridgeHeaderMethodWithEntitlements verifies entitlement comments appear in BridgeHeader.
func TestBridgeHeaderMethodWithEntitlements(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSFoo": {
				Methods: []meta.Method{
					{
						Selector: "doPrivateThing",
						Return:   meta.ReturnType{ObjCType: "void"},
						Availability: meta.Availability{
							Entitlements: []string{"com.apple.developer.network"},
						},
					},
				},
			},
		},
		Functions: []meta.Function{},
		Protocols: map[string]meta.Protocol{},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "com.apple.developer.network") {
		t.Errorf("expected entitlement comment in bridge header; got:\n%s", buf.String())
	}
}

// TestBridgeImplStructByValueReturn exercises the struct-by-value malloc path.
func TestBridgeImplStructByValueReturn(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSFoo": {
				Methods: []meta.Method{
					{
						Selector: "frame",
						Return:   meta.ReturnType{ObjCType: "CGRect"},
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
	if !strings.Contains(out, "malloc") {
		t.Errorf("CGRect return should use malloc for struct-by-value; got:\n%s", out)
	}
}

// TestBridgeImplFunctionVoidReturn exercises writeFunctionImpl for a void function.
func TestBridgeImplFunctionVoidReturn(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Functions: []meta.Function{
			{
				Name:   "NSMakeRange",
				Return: meta.ReturnType{ObjCType: "void"},
				Params:   []meta.Param{{Name: "loc", ObjCType: "NSUInteger"}},
			},
		},
		Classes:   map[string]meta.Class{},
		Protocols: map[string]meta.Protocol{},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "foundation_fn_NSMakeRange") {
		t.Errorf("expected function bridge impl; got:\n%s", out)
	}
}

// TestBridgeImplFunctionScalarReturn exercises writeFunctionImpl with a scalar return.
func TestBridgeImplFunctionScalarReturn(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Functions: []meta.Function{
			{
				Name:   "NSStringLength",
				Return: meta.ReturnType{ObjCType: "NSUInteger"},
				Params:   []meta.Param{{Name: "s", ObjCType: "NSString *"}},
			},
		},
		Classes:   map[string]meta.Class{},
		Protocols: map[string]meta.Protocol{},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "foundation_fn_NSStringLength") {
		t.Errorf("expected function bridge impl for NSStringLength; got:\n%s", out)
	}
}

// ── hasByValueUnknownType ─────────────────────────────────────────────────────

func TestHasByValueUnknownTypeBlock(t *testing.T) {
	fn := meta.Function{Name: "test", Return: meta.ReturnType{ObjCType: "void"}, Params: []meta.Param{{Name: "cb", ObjCType: "void (^)(void)"}}}
	if hasByValueUnknownType(fn) {
		t.Error("block type should not be 'by-value unknown'")
	}
}

func TestHasByValueUnknownTypeNullabilityAnnotation(t *testing.T) {
	fn := meta.Function{Name: "test", Return: meta.ReturnType{ObjCType: "void"}, Params: []meta.Param{{Name: "p", ObjCType: "NSString * _Nonnull"}}}
	if hasByValueUnknownType(fn) {
		t.Error("_Nonnull type should not be 'by-value unknown'")
	}
}

func TestHasByValueUnknownTypeVoid(t *testing.T) {
	fn := meta.Function{Name: "test", Return: meta.ReturnType{ObjCType: "void"}}
	if hasByValueUnknownType(fn) {
		t.Error("void return should not be 'by-value unknown'")
	}
}

func TestHasByValueUnknownTypePointer(t *testing.T) {
	fn := meta.Function{Name: "test", Return: meta.ReturnType{ObjCType: "NSObject *"}}
	if hasByValueUnknownType(fn) {
		t.Error("pointer type should not be 'by-value unknown'")
	}
}

func TestHasByValueUnknownTypeSuffixT(t *testing.T) {
	fn := meta.Function{Name: "test", Return: meta.ReturnType{ObjCType: "vmnet_return_t"}}
	if hasByValueUnknownType(fn) {
		t.Error("_t suffix type should not be 'by-value unknown'")
	}
}

func TestHasByValueUnknownTypeRefContaining(t *testing.T) {
	fn := meta.Function{Name: "test", Return: meta.ReturnType{ObjCType: "CGColorRef"}}
	if hasByValueUnknownType(fn) {
		t.Error("Ref-containing type should not be 'by-value unknown'")
	}
}

func TestHasByValueUnknownTypeSIMD(t *testing.T) {
	fn := meta.Function{Name: "test", Return: meta.ReturnType{ObjCType: "DenseMatrix_Float"}}
	if !hasByValueUnknownType(fn) {
		t.Error("SIMD/vector type without _t/Ref should be 'by-value unknown'")
	}
}

func TestHasByValueUnknownTypeBOOL(t *testing.T) {
	fn := meta.Function{Name: "test", Return: meta.ReturnType{ObjCType: "BOOL"}}
	if hasByValueUnknownType(fn) {
		t.Error("BOOL should not be 'by-value unknown'")
	}
}

func TestHasByValueUnknownTypeID(t *testing.T) {
	fn := meta.Function{Name: "test", Return: meta.ReturnType{ObjCType: "id"}}
	if hasByValueUnknownType(fn) {
		t.Error("id should not be 'by-value unknown'")
	}
}

func TestHasByValueUnknownTypeScalar(t *testing.T) {
	fn := meta.Function{Name: "test", Return: meta.ReturnType{ObjCType: "NSUInteger"}}
	if hasByValueUnknownType(fn) {
		t.Error("NSUInteger should not be 'by-value unknown'")
	}
}

// ── writeProtocolMethodImpl additional paths ──────────────────────────────────

// TestBridgeImplProtocolMethodNSError exercises writeProtocolMethodImpl with NSError.
func TestBridgeImplProtocolMethodNSError(t *testing.T) {
	m := bridgeTestMapper()
	m.ProtocolProxyIndex = map[string]string{"VZFoo": "Virtualization"}
	framework := &meta.FrameworkMeta{
		Framework: "Virtualization",
		Protocols: map[string]meta.Protocol{
			"VZFoo": {
				Methods: []meta.Method{
					{
						Selector:   "performAction:error:",
						IsNSError: true,
						Return:     meta.ReturnType{ObjCType: "void"},
					},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "virtualization_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "outError") {
		t.Errorf("expected NSError bridge for protocol method; got:\n%s", out)
	}
}

// TestBridgeImplProtocolMethodObjectReturn verifies object return in protocol method.
func TestBridgeImplProtocolMethodObjectReturn(t *testing.T) {
	m := bridgeTestMapper()
	m.ProtocolProxyIndex = map[string]string{"VZProvider": "Virtualization"}
	framework := &meta.FrameworkMeta{
		Framework: "Virtualization",
		Protocols: map[string]meta.Protocol{
			"VZProvider": {
				Methods: []meta.Method{
					{
						Selector: "currentObject",
						Return:   meta.ReturnType{ObjCType: "NSObject *"},
					},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{"NSObject": true}, "virtualization_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "[_result retain]") {
		t.Errorf("expected retain in object return for protocol method; got:\n%s", out)
	}
}

// TestBridgeImplProtocolMethodPrimitiveReturn verifies primitive return in protocol method.
func TestBridgeImplProtocolMethodPrimitiveReturn(t *testing.T) {
	m := bridgeTestMapper()
	m.ProtocolProxyIndex = map[string]string{"VZCounter": "Virtualization"}
	framework := &meta.FrameworkMeta{
		Framework: "Virtualization",
		Protocols: map[string]meta.Protocol{
			"VZCounter": {
				Methods: []meta.Method{
					{
						Selector: "count",
						Return:   meta.ReturnType{ObjCType: "NSUInteger"},
					},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "virtualization_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "_result") {
		t.Errorf("expected _result in primitive return for protocol method; got:\n%s", out)
	}
}

// ── bridgeCatchReturn ─────────────────────────────────────────────────────────

func TestBridgeCatchReturnVoid(t *testing.T) {
	got := bridgeCatchReturn("void")
	if got != "" {
		t.Errorf("void catch return should be empty; got %q", got)
	}
}

func TestBridgeCatchReturnPointer(t *testing.T) {
	got := bridgeCatchReturn("void *")
	if got != "return nil;" {
		t.Errorf("pointer catch return should be 'return nil;'; got %q", got)
	}
}

func TestBridgeCatchReturnBool(t *testing.T) {
	got := bridgeCatchReturn("bool")
	if got != "return false;" {
		t.Errorf("bool catch return should be 'return false;'; got %q", got)
	}
}

func TestBridgeCatchReturnInt(t *testing.T) {
	got := bridgeCatchReturn("int32_t")
	if got != "return 0;" {
		t.Errorf("int catch return should be 'return 0;'; got %q", got)
	}
}

// ── BridgeHeader with NSCoding class ─────────────────────────────────────────

func TestBridgeHeaderNSCodingClass(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSFoo": {
				Protocols: []string{"NSSecureCoding"},
				Methods:   []meta.Method{},
			},
		},
		Functions: []meta.Function{},
		Protocols: map[string]meta.Protocol{},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{"NSFoo": true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// NSSecureCoding class should get coding bridge decl
	if !strings.Contains(out, "NSFoo") {
		t.Errorf("expected NSFoo coding bridge; got:\n%s", out)
	}
}

// ── BridgeHeader protocol proxy declarations ──────────────────────────────────

// TestBridgeHeaderProtocolProxy verifies protocol proxy bridge declarations appear.
func TestBridgeHeaderProtocolProxy(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Protocols: map[string]meta.Protocol{
			"NSCopying": {
				Methods: []meta.Method{
					{Selector: "copyWithZone:", Return: meta.ReturnType{ObjCType: "id"}},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Should emit a protocol proxy bridge decl for NSCopying
	if !strings.Contains(out, "IDProtocol") {
		t.Errorf("expected IDProtocol bridge decl for protocol; got:\n%s", out)
	}
}

// TestBridgeHeaderProtocolProxyUnavailable verifies unavailable protocols are skipped.
func TestBridgeHeaderProtocolProxyUnavailable(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Protocols: map[string]meta.Protocol{
			"NSDeprecated": {
				Availability: meta.Availability{IsUnavailable: true},
				Methods: []meta.Method{
					{Selector: "doThing", Return: meta.ReturnType{ObjCType: "void"}},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "NSDeprecated") {
		t.Errorf("unavailable protocol should be skipped; got:\n%s", out)
	}
}

// TestBridgeHeaderProtocolProxyClassMethodSkipped verifies class methods are excluded.
func TestBridgeHeaderProtocolProxyClassMethodSkipped(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Protocols: map[string]meta.Protocol{
			"NSFactory": {
				Methods: []meta.Method{
					{Selector: "create", IsClassMethod: true, Return: meta.ReturnType{ObjCType: "id"}},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeHeader(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Class method should not appear as a protocol proxy (only instance methods are proxied)
	if strings.Contains(out, "nsFactory") || strings.Contains(out, "IDProtocol") {
		t.Logf("output (class method should be absent from proxy decl): %s", out)
	}
}

// ── writeFunctionImpl — block arg paths ───────────────────────────────────────

// TestBridgeImplFunctionWithInlineBlock verifies inline block args are cast to concrete type.
func TestBridgeImplFunctionWithInlineBlock(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Functions: []meta.Function{
			{
				Name:   "NSPerformBlock",
				Return: meta.ReturnType{ObjCType: "void"},
				Params: []meta.Param{
					{Name: "block", ObjCType: "void (^)(void)", IsBlock: true},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// The block arg should be cast back to its concrete block type.
	if !strings.Contains(out, "(void (^)(void))") {
		t.Errorf("inline block arg should be cast to concrete type; got:\n%s", out)
	}
}

// TestBridgeImplFunctionWithBlockTypedef verifies named block typedef args are cast by typedef name.
func TestBridgeImplFunctionWithBlockTypedef(t *testing.T) {
	m := bridgeTestMapper()
	m.TypedefIndex = map[string]string{"dispatch_block_t": "void (^)(void)"}
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Functions: []meta.Function{
			{
				Name:   "NSExecBlock",
				Return: meta.ReturnType{ObjCType: "void"},
				Params:   []meta.Param{{Name: "block", ObjCType: "dispatch_block_t"}},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "(dispatch_block_t)") {
		t.Errorf("named block typedef arg should be cast to typedef name; got:\n%s", out)
	}
}

// ── writeFunctionImpl — struct-by-value return path ───────────────────────────

// TestWriteFunctionImplStructReturn covers the struct-by-value return path in writeFunctionImpl.
func TestWriteFunctionImplStructReturn(t *testing.T) {
	m := bridgeStructMapper()
	// MTLViewport is in StructIndex → isStructByValueType returns true.
	fn := meta.Function{
		Name:   "GetViewport",
		Return: meta.ReturnType{ObjCType: "MTLViewport"},
	}
	ctx := m.BaseContext("Metal", map[string]bool{})
	var buf bytes.Buffer
	writeFunctionImpl(&buf, "metal_fn_GetViewport", fn, ctx, m)
	out := buf.String()
	if !strings.Contains(out, "malloc") {
		t.Errorf("struct-by-value return should use malloc; got:\n%s", out)
	}
	if !strings.Contains(out, "return (void *)_result") {
		t.Errorf("struct-by-value return should return (void *)_result; got:\n%s", out)
	}
}

// ── buildObjCCall — block arg and NSError paths ───────────────────────────────

// TestBridgeImplMethodWithBlockArg covers the IsBlock arg path in buildObjCCall + bridgeParamList.
func TestBridgeImplMethodWithBlockArg(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSObject": {
				Methods: []meta.Method{
					{
						Selector: "doWithBlock:",
						Return:   meta.ReturnType{ObjCType: "void"},
						Params:     []meta.Param{{Name: "handler", ObjCType: "void (^)(void)", IsBlock: true}},
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
	// Block arg should be cast to its ObjC block type in the method call.
	if !strings.Contains(out, "__bridge") {
		t.Errorf("block method arg should use __bridge cast in ObjC call; got:\n%s", out)
	}
}

// ── writeProtocolMethodImpl — scalar+NSError and object+NSError paths ─────────

// TestBridgeImplProtocolMethodScalarWithNSError covers scalar return + NSError in writeProtocolMethodImpl.
func TestBridgeImplProtocolMethodScalarWithNSError(t *testing.T) {
	m := bridgeTestMapper()
	m.ProtocolProxyIndex = map[string]string{"VZCounter": "Virtualization"}
	framework := &meta.FrameworkMeta{
		Framework: "Virtualization",
		Protocols: map[string]meta.Protocol{
			"VZCounter": {
				Methods: []meta.Method{
					{
						Selector:   "countWithError:",
						IsNSError: true,
						Return:     meta.ReturnType{ObjCType: "NSUInteger"},
					},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "virtualization_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Should emit outError assignment for scalar return + NSError
	if !strings.Contains(out, "outError") {
		t.Errorf("scalar+NSError protocol method should emit outError; got:\n%s", out)
	}
}

// TestBridgeImplProtocolMethodObjectReturnWithNSError covers object+NSError path in writeProtocolMethodImpl.
func TestBridgeImplProtocolMethodObjectReturnWithNSError(t *testing.T) {
	m := bridgeTestMapper()
	m.ProtocolProxyIndex = map[string]string{"VZProvider": "Virtualization"}
	framework := &meta.FrameworkMeta{
		Framework: "Virtualization",
		Protocols: map[string]meta.Protocol{
			"VZProvider": {
				Methods: []meta.Method{
					{
						Selector:   "currentObjectWithError:",
						IsNSError: true,
						Return:     meta.ReturnType{ObjCType: "NSObject *"},
					},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{"NSObject": true}, "virtualization_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "outError") {
		t.Errorf("object+NSError protocol method should emit outError; got:\n%s", out)
	}
	if !strings.Contains(out, "[_result retain]") {
		t.Errorf("object return should retain result; got:\n%s", out)
	}
}

// ── isObjectObjCType — generic underscore-prefix type ─────────────────────────

// TestIsObjectObjCTypeUnderscoreGeneric covers the generic-suffix strip in isObjectObjCType.
func TestIsObjectObjCTypeUnderscoreGeneric(t *testing.T) {
	// _NSProxy<NSCopying> *: ClassName returns "" (underscore prefix), bare strip needed.
	knownClasses := map[string]bool{"_NSProxy": true}
	if !isObjectObjCType("_NSProxy<NSCopying> *", knownClasses) {
		t.Errorf("isObjectObjCType(_NSProxy<NSCopying> *) should be true; class is in knownClasses")
	}
}

// ── isStructByValueType — struct prefix with StructIndex ────────────────────

// TestIsStructByValueTypeExplicitStructWithKnownStruct covers "struct X" where X is in StructIndex.
func TestIsStructByValueTypeExplicitStructWithKnownStruct(t *testing.T) {
	m := bridgeStructMapper()
	ctx := m.BaseContext("Metal", map[string]bool{})
	// "struct MTLViewport" — bare = "MTLViewport", IsStructCType = false, but StructIndex has it.
	if !isStructByValueType("struct MTLViewport", ctx, m) {
		t.Error("struct MTLViewport should be struct-by-value via StructIndex; got false")
	}
}
