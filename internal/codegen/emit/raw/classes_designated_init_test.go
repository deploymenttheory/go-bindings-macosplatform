package raw

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// ============================================================
// writeDesignatedInitConstructors — uncovered switch cases
// ============================================================

// TestWriteDesignatedInitCtorIDReturn covers the `case initReturnsID` path
// (Return.ObjCType="id" → GoType returns "objc.Object", HasNSError=false).
func TestWriteDesignatedInitCtorIDReturn(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{
		Methods: []meta.Method{
			{
				Selector:         "initWith:",
				IsDesignatedInit: true,
				IsInit:           true,
				Params:             []meta.Param{{Name: "obj", ObjCType: "NSUInteger"}},
				Return:           meta.ReturnType{ObjCType: "id"}, // → "objc.Object" → initReturnsID
			},
		},
	}
	ctx := typemap.Context{ClassName: "NSObject"}
	allClasses := map[string]meta.Class{"NSObject": cls}

	var buf bytes.Buffer
	writeDesignatedInitConstructors(&buf, "NSObject", cls, false, framework, ctx, m, allClasses, make(typemap.ImportSet))
	out := buf.String()

	if !strings.Contains(out, "_idResult :=") {
		t.Errorf("initReturnsID path should use _idResult; got:\n%s", out)
	}
	if !strings.Contains(out, "_idResult == nil") {
		t.Errorf("initReturnsID path should nil-check _idResult; got:\n%s", out)
	}
	if !strings.Contains(out, "NewNSObject(_idResult.Ptr())") {
		t.Errorf("initReturnsID path should call NewNSObject(_idResult.Ptr()); got:\n%s", out)
	}
}

// TestWriteDesignatedInitCtorIDReturnWithNSError covers the
// `case initReturnsID && method.IsNSError` path.
func TestWriteDesignatedInitCtorIDReturnWithNSError(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{
		Methods: []meta.Method{
			{
				Selector:         "initWithError:",
				IsDesignatedInit: true,
				IsInit:           true,
				IsNSError:       true,
				Params:             []meta.Param{{Name: "obj", ObjCType: "NSUInteger"}},
				Return:           meta.ReturnType{ObjCType: "id"}, // → "objc.Object" → initReturnsID
			},
		},
	}
	ctx := typemap.Context{ClassName: "NSObject"}
	allClasses := map[string]meta.Class{"NSObject": cls}

	var buf bytes.Buffer
	writeDesignatedInitConstructors(&buf, "NSObject", cls, false, framework, ctx, m, allClasses, make(typemap.ImportSet))
	out := buf.String()

	if !strings.Contains(out, "_idResult, _err :=") {
		t.Errorf("initReturnsID+NSError should use _idResult, _err :=; got:\n%s", out)
	}
	if !strings.Contains(out, "return nil, _err") {
		t.Errorf("initReturnsID+NSError should return (nil, _err) on nil result; got:\n%s", out)
	}
	if !strings.Contains(out, "(*NSObject, error)") {
		t.Errorf("NSError ctor should have (*NSObject, error) return; got:\n%s", out)
	}
}

// TestWriteDesignatedInitCtorUnsafeReturn covers the `case initReturnsPtr` path
// (Return.ObjCType="void *" → "unsafe.Pointer" → initReturnsUnsafe, no NSError).
func TestWriteDesignatedInitCtorUnsafeReturn(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{
		Methods: []meta.Method{
			{
				Selector:         "initWithPtr:",
				IsDesignatedInit: true,
				IsInit:           true,
				Params:             []meta.Param{{Name: "raw", ObjCType: "NSUInteger"}},
				Return:           meta.ReturnType{ObjCType: "void *"}, // → "unsafe.Pointer" → initReturnsPtr
			},
		},
	}
	ctx := typemap.Context{ClassName: "NSObject"}
	allClasses := map[string]meta.Class{"NSObject": cls}

	var buf bytes.Buffer
	writeDesignatedInitConstructors(&buf, "NSObject", cls, false, framework, ctx, m, allClasses, make(typemap.ImportSet))
	out := buf.String()

	if !strings.Contains(out, "return NewNSObject(") {
		t.Errorf("initReturnsPtr (no NSError) should call return NewNSObject(callExpr); got:\n%s", out)
	}
}

// TestWriteDesignatedInitCtorUnsafeReturnWithNSError covers the
// `case initReturnsPtr && method.IsNSError` path.
func TestWriteDesignatedInitCtorUnsafeReturnWithNSError(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{
		Methods: []meta.Method{
			{
				Selector:         "initWithPtrAndError:",
				IsDesignatedInit: true,
				IsInit:           true,
				IsNSError:       true,
				Params:             []meta.Param{{Name: "raw", ObjCType: "NSUInteger"}},
				Return:           meta.ReturnType{ObjCType: "void *"}, // → "unsafe.Pointer" → initReturnsPtr
			},
		},
	}
	ctx := typemap.Context{ClassName: "NSObject"}
	allClasses := map[string]meta.Class{"NSObject": cls}

	var buf bytes.Buffer
	writeDesignatedInitConstructors(&buf, "NSObject", cls, false, framework, ctx, m, allClasses, make(typemap.ImportSet))
	out := buf.String()

	if !strings.Contains(out, "_result, _err :=") {
		t.Errorf("initReturnsPtr+NSError should use _result, _err :=; got:\n%s", out)
	}
	if !strings.Contains(out, "return NewNSObject(_result), _err") {
		t.Errorf("initReturnsPtr+NSError should return (NewNSObject(_result), _err); got:\n%s", out)
	}
}

// TestWriteDesignatedInitCtorEmptySuffix covers the `ctorSuffix == ""` path
// (selector "init:" has MethodName="Init", TrimPrefix("Init","Init")="").
func TestWriteDesignatedInitCtorEmptySuffix(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{
		Methods: []meta.Method{
			{
				Selector:         "init:", // MethodName → "Init" → ctorSuffix = ""
				IsDesignatedInit: true,
				IsInit:           true,
				Params:             []meta.Param{{Name: "v", ObjCType: "NSUInteger"}},
				Return:           meta.ReturnType{IsInstancetype: true},
			},
		},
	}
	ctx := typemap.Context{ClassName: "NSObject"}
	allClasses := map[string]meta.Class{"NSObject": cls}

	var buf bytes.Buffer
	writeDesignatedInitConstructors(&buf, "NSObject", cls, false, framework, ctx, m, allClasses, make(typemap.ImportSet))
	out := buf.String()

	// ctorSuffix="" → ctorSuffix = goMethodName = "Init" → ctorName = "NewNSObjectInit"
	if !strings.Contains(out, "func NewNSObjectInit(") {
		t.Errorf("empty ctorSuffix should produce NewNSObjectInit; got:\n%s", out)
	}
}

// TestWriteDesignatedInitCtorDuplicateSkipped verifies the seenCtorNames guard
// covers the `continue` for duplicate ctor names.
func TestWriteDesignatedInitCtorDuplicateSkipped(t *testing.T) {
	m := classTestMapper()
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	cls := meta.Class{
		Methods: []meta.Method{
			{
				Selector:         "initWithFoo:",
				IsDesignatedInit: true,
				IsInit:           true,
				Params:             []meta.Param{{Name: "foo", ObjCType: "NSUInteger"}},
				Return:           meta.ReturnType{IsInstancetype: true},
			},
			{
				Selector:         "initWithFoo:", // duplicate selector → same ctorName
				IsDesignatedInit: true,
				IsInit:           true,
				Params:             []meta.Param{{Name: "foo", ObjCType: "NSUInteger"}},
				Return:           meta.ReturnType{IsInstancetype: true},
			},
		},
	}
	ctx := typemap.Context{ClassName: "NSObject"}
	allClasses := map[string]meta.Class{"NSObject": cls}

	var buf bytes.Buffer
	writeDesignatedInitConstructors(&buf, "NSObject", cls, false, framework, ctx, m, allClasses, make(typemap.ImportSet))
	out := buf.String()

	// Should emit NewNSObjectWithFoo exactly once.
	first := strings.Index(out, "func NewNSObjectWithFoo(")
	if first < 0 {
		t.Fatalf("NewNSObjectWithFoo should be emitted; got:\n%s", out)
	}
	if strings.Contains(out[first+1:], "func NewNSObjectWithFoo(") {
		t.Errorf("duplicate ctor should be skipped; got:\n%s", out)
	}
}

// ============================================================
// primaryReturnType — IsGeneric path + GenericClasses path
// ============================================================

// TestPrimaryReturnTypeIsGenericWithParams covers `ret.IsGeneric && len(ctx.GenericParams) > 0`.
func TestPrimaryReturnTypeIsGenericWithParams(t *testing.T) {
	m := &typemap.Mapper{GenericClasses: map[string]bool{"NSArray": true}}
	ctx := typemap.Context{
		ClassName:     "NSArray",
		GenericParams: []string{"ObjectType"},
	}
	method := meta.Method{
		Return: meta.ReturnType{IsGeneric: true}, // generic element return
	}
	got := primaryReturnType(method, ctx, m, nil)
	if got != "objc.Object" {
		t.Errorf("IsGeneric+GenericParams: want objc.Object, got %q", got)
	}
}

// TestPrimaryReturnTypeGenericClassNoParams covers `m.GenericClasses[ctx.ClassName]`
// when ctx.GenericParams is empty (the [objc.Object] substitution for class methods).
func TestPrimaryReturnTypeGenericClassNoParams(t *testing.T) {
	m := &typemap.Mapper{GenericClasses: map[string]bool{"NSArray": true}}
	ctx := typemap.Context{
		ClassName:     "NSArray",
		GenericParams: nil, // empty — no generic params in context
	}
	method := meta.Method{
		Return: meta.ReturnType{IsInstancetype: true},
	}
	got := primaryReturnType(method, ctx, m, nil)
	// m.GenericClasses["NSArray"]=true, no GenericParams → *NSArray[objc.Object]
	if got != "*NSArray[objc.Object]" {
		t.Errorf("GenericClasses (no params): want *NSArray[objc.Object], got %q", got)
	}
}

// ============================================================
// isKnownStruct — lowercase first-letter key path
// ============================================================

// TestIsKnownStructLowercaseFirstLetter covers the `m.StructIndex[lower] != ""` path
// where the bare name starts with a capital that maps to a lowercase key.
func TestIsKnownStructLowercaseFirstLetter(t *testing.T) {
	m := &typemap.Mapper{
		StructIndex: map[string]string{
			"decform": "SomeFramework", // stored with lowercase first letter
		},
	}
	// bare = "Decform" (capital D): StructIndex["Decform"] = "" but StructIndex["decform"] != ""
	if !isKnownStruct("Decform", m) {
		t.Error("isKnownStruct('Decform') should return true via lowercase key 'decform'")
	}
}

// ============================================================
// methodHasBlockArgs — typedef block path
// ============================================================

// TestMethodHasBlockArgsTypedefBlock covers the TypedefIndex lookup path
// where arg.IsBlock=false but the typedef resolves to a block type.
func TestMethodHasBlockArgsTypedefBlock(t *testing.T) {
	m := &typemap.Mapper{
		TypedefIndex: map[string]string{
			"dispatch_block_t": "void (^)(void)", // typedef → block type
		},
	}
	args := []meta.Param{
		{Name: "handler", ObjCType: "dispatch_block_t", IsBlock: false},
	}
	if !methodHasBlockArgs(args, m) {
		t.Error("methodHasBlockArgs should return true for typedef block arg")
	}
}

// TestMethodHasBlockArgsNoBlock covers the false path when no block args exist.
func TestMethodHasBlockArgsNoBlock(t *testing.T) {
	m := &typemap.Mapper{
		TypedefIndex: map[string]string{},
	}
	args := []meta.Param{
		{Name: "count", ObjCType: "NSUInteger"},
	}
	if methodHasBlockArgs(args, m) {
		t.Error("methodHasBlockArgs should return false when no block args")
	}
}

// ============================================================
// writeNSStringClassOverload — void return path
// ============================================================

// TestWriteNSStringClassOverloadVoidReturnPath covers the void-return branch
// in writeNSStringClassOverload (`goRet == ""` → emit call without "return").
func TestWriteNSStringClassOverloadVoidReturnPath(t *testing.T) {
	m := classTestMapper()
	m.IsNSStringOverloads = true
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{}}
	// A class method with NSString arg but void return
	cls := meta.Class{
		Methods: []meta.Method{
			{
				Selector:      "logString:",
				IsClassMethod: true,
				Params:          []meta.Param{{Name: "msg", ObjCType: "NSString *"}},
				Return:        meta.ReturnType{ObjCType: "void"},
			},
		},
	}
	all := map[string]meta.Class{"NSObject": cls}
	known := map[string]bool{"NSObject": true}
	var buf bytes.Buffer
	if err := EmitClass(&buf, "NSObject", cls, framework, m, known, all, "foundation"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// The overload function should call NSObjectLogString without "return".
	if strings.Contains(out, "return NSObjectLogStringGo") {
		t.Errorf("void return class overload should not have return statement; got:\n%s", out)
	}
	// Should contain the overload func definition
	if !strings.Contains(out, "NSObjectLogStringGo") {
		t.Errorf("expected NSObjectLogStringGo overload to be emitted; got:\n%s", out)
	}
}

// ============================================================
// writeNSStringInstanceOverload — void return path
// ============================================================

// TestWriteNSStringInstanceOverloadVoidReturnPath covers the void-return branch
// in writeNSStringInstanceOverload (goRet == "" → emit call without "return").
func TestWriteNSStringInstanceOverloadVoidReturnPath(t *testing.T) {
	m := classTestMapper()
	m.IsNSStringOverloads = true
	framework := &meta.FrameworkMeta{Framework: "Foundation", Classes: map[string]meta.Class{"NSObject": {}}}
	// Instance method with NSString arg and void return.
	cls := meta.Class{
		Methods: []meta.Method{
			{
				Selector: "setName:",
				Params:     []meta.Param{{Name: "name", ObjCType: "NSString *"}},
				Return:   meta.ReturnType{ObjCType: "void"},
			},
		},
	}
	all := map[string]meta.Class{"NSObject": cls}
	known := map[string]bool{"NSObject": true}
	var buf bytes.Buffer
	if err := EmitClass(&buf, "NSObject", cls, framework, m, known, all, "foundation"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// The overload func should call SetName without "return".
	if strings.Contains(out, "return o.SetNameGo") {
		t.Errorf("void instance overload should not use return; got:\n%s", out)
	}
}

// ============================================================
// buildValueChainInner — superInFW override by OwnerIndex
// ============================================================

// TestBuildValueChainSuperInFWOverriddenByOwner covers the `superInFW = false` path
// where the super IS in fmClasses but OwnerIndex says it belongs to another framework.
func TestBuildValueChainSuperInFWOverriddenByOwner(t *testing.T) {
	m := &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex: map[string]string{
			"NSObject": "Foundation", // NSObject owned by Foundation, not AppKit
		},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
	// NSView is in AppKit; its super NSObject IS in fmClasses but OwnerIndex→Foundation.
	fmClasses := map[string]meta.Class{
		"NSObject": {},                  // super IS present in fmClasses
		"NSView":   {Super: "NSObject"}, // child class
	}
	// currentFW = "AppKit" != "Foundation" → superInFW overridden to false
	result := buildValueChainInner("NSView", "", "AppKit", fmClasses, m, make(map[string]string))
	// Should use cross-fw constructor for NSObject (since OwnerIndex overrides superInFW).
	if !strings.Contains(result, "foundation") {
		t.Errorf("superInFW override should use cross-fw reference to foundation; got %q", result)
	}
}

// ============================================================
// CollectBlockSignaturesFromFrameworks — functions loop + nil typedefs
// ============================================================

// TestCollectBlockSigsFromFunctions covers the framework.Functions loop in CollectBlockSignaturesFromFrameworks.
func TestCollectBlockSigsFromFunctions(t *testing.T) {
	m := blockTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Functions: []meta.Function{
			{
				Name: "NSDoWith",
				Params: []meta.Param{
					{Name: "handler", ObjCType: "void (^)(NSError *)", IsBlock: true},
				},
				Return: meta.ReturnType{ObjCType: "void"},
			},
		},
	}
	sigs := CollectBlockSignaturesFromFrameworks([]*meta.FrameworkMeta{framework}, m)
	if len(sigs) == 0 {
		t.Error("expected block signature from Functions loop; got none")
	}
}

// TestCollectBlockSigsNilTypedefs covers the `if typedefs == nil { return "" }` path
// in resolveBlockObjCType, reached when arg.IsBlock=false and framework.Typedefs is nil.
func TestCollectBlockSigsNilTypedefs(t *testing.T) {
	m := blockTestMapper()
	// framework.Typedefs is nil; arg is NOT a block (IsBlock=false) → hits typedefs==nil early return
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSFoo": {
				Methods: []meta.Method{
					{
						Selector: "doThing:",
						Params:     []meta.Param{{Name: "handler", ObjCType: "NSHandler", IsBlock: false}},
						Return:   meta.ReturnType{ObjCType: "void"},
					},
				},
			},
		},
		// Typedefs is nil — exercises typedefs == nil early return in resolveBlockObjCType
	}
	sigs := CollectBlockSignaturesFromFrameworks([]*meta.FrameworkMeta{framework}, m)
	// Non-block arg with nil typedefs → no sigs.
	if len(sigs) != 0 {
		t.Errorf("non-block arg with nil typedefs should produce no sigs; got %d", len(sigs))
	}
}

// TestCollectBlockSigsInvalidBlockSig covers the `if !ok { continue }` path
// in addArgs, reached when BlockSigFromObjC returns !ok for an invalid block type.
func TestCollectBlockSigsInvalidBlockSig(t *testing.T) {
	m := blockTestMapper()
	// arg.IsBlock=true but ObjCType is not parseable as a block type → BlockSigFromObjC !ok
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSFoo": {
				Methods: []meta.Method{
					{
						Selector: "doThing:",
						Params:     []meta.Param{{Name: "handler", ObjCType: "notABlockType", IsBlock: true}},
						Return:   meta.ReturnType{ObjCType: "void"},
					},
				},
			},
		},
	}
	sigs := CollectBlockSignaturesFromFrameworks([]*meta.FrameworkMeta{framework}, m)
	// Invalid block type → BlockSigFromObjC returns !ok → continue → 0 sigs.
	if len(sigs) != 0 {
		t.Errorf("invalid block type should produce no sigs; got %d", len(sigs))
	}
}

// ============================================================
// blockNeedsWrapper — CFTypeIndex path
// ============================================================

// TestBlockNeedsWrapperCFType covers the `m.CFTypeIndex[n]` path.
func TestBlockNeedsWrapperCFType(t *testing.T) {
	m := &typemap.Mapper{
		CFTypeIndex: map[string]string{
			"CGColorRef": "CoreGraphics", // known CF type
		},
		TypedefIndex: map[string]string{},
	}
	ctx := typemap.Context{ClassNameIndex: map[string]bool{}}
	// Block with CGColorRef arg → hits CFTypeIndex path
	if !blockNeedsWrapper("void (^)(CGColorRef)", ctx, m) {
		t.Error("block with CF type arg should need wrapper (CFTypeIndex path)")
	}
}

// TestBlockNeedsWrapperReturnKnownClassPtr covers the
// `cls != "" && ctx.ClassNameIndex[cls]` on return type path (RETURN class, not arg).
func TestBlockNeedsWrapperReturnKnownClassPtr(t *testing.T) {
	m := &typemap.Mapper{
		TypedefIndex:         map[string]string{},
		CFTypeIndex: map[string]string{},
	}
	ctx := typemap.Context{
		ClassNameIndex: map[string]bool{"NSString": true},
	}
	// Block returning NSString * → ClassName("NSString *")="NSString" is in ClassNameIndex
	if !blockNeedsWrapper("NSString * (^)(void)", ctx, m) {
		t.Error("block with known class return should need wrapper")
	}
}

// ============================================================
// blockArgCtorTyped — struct pointer path
// ============================================================

// TestBlockArgCtorTypedKnownStructPointer covers the `m.StructIndex[structName] != ""` path
// which returns an unsafe cast rather than a New* constructor.
func TestBlockArgCtorTypedKnownStructPointer(t *testing.T) {
	m := &typemap.Mapper{
		StructIndex: map[string]string{"CGRect": "CoreGraphics"},
	}
	got := blockArgCtorTyped("*CGRect", "p", m)
	if got != "(*CGRect)(p)" {
		t.Errorf("KnownStruct pointer: want (*CGRect)(p), got %q", got)
	}
}

// TestBlockArgCtorTypedCrossFrameworkStructPointer covers stripping a package qualifier
// from a struct name before the StructIndex lookup.
func TestBlockArgCtorTypedCrossFrameworkStructPointer(t *testing.T) {
	m := &typemap.Mapper{
		StructIndex: map[string]string{"CGRect": "CoreGraphics"},
	}
	// *coregraphics.CGRect → strip "coregraphics." → bare "CGRect" → StructIndex match
	got := blockArgCtorTyped("*coregraphics.CGRect", "p", m)
	if got != "(*coregraphics.CGRect)(p)" {
		t.Errorf("cross-fw struct pointer: want (*coregraphics.CGRect)(p), got %q", got)
	}
}

// ============================================================
// BridgeImpl writeProtocolMethodImpl — AlreadyRetained path
// ============================================================

// TestBridgeImplProtocolMethodAlreadyRetained covers the `ret.IsAlreadyRetained=true` path
// in writeProtocolMethodImpl which emits `(__bridge void *)_result` without `retain`.
func TestBridgeImplProtocolMethodAlreadyRetained(t *testing.T) {
	m := bridgeTestMapper()
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Protocols: map[string]meta.Protocol{
			"NSFooDelegate": {
				Methods: []meta.Method{
					{
						Selector: "createFoo",
						Return: meta.ReturnType{
							ObjCType:        "NSObject *",
							IsAlreadyRetained: true, // → no [_result retain]
						},
					},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{"NSObject": true}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// AlreadyRetained path: return (__bridge id)_result (no retain call).
	if strings.Contains(out, "[_result retain]") {
		t.Errorf("AlreadyRetained path should NOT call retain; got:\n%s", out)
	}
	if !strings.Contains(out, "(__bridge id)_result") {
		t.Errorf("AlreadyRetained path should emit (__bridge id)_result; got:\n%s", out)
	}
}

// ============================================================
// hasByValueUnknownType — arg-type check path
// ============================================================

// TestHasByValueUnknownTypeArgFiltered verifies that a function whose arg type
// is an unknown struct-by-value type (no _t suffix, no Ref) is filtered by BridgeImpl.
// This covers the `check(arg.ObjCType) = true` path when return type is void (fine).
func TestHasByValueUnknownTypeArgFiltered(t *testing.T) {
	m := bridgeTestMapper()
	// "vFloat" has no _t suffix, no Ref, no *, not a scalar → check returns true for it.
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]meta.Class{},
		Functions: []meta.Function{
			{
				Name:   "NSProcessFloat",
				Params:   []meta.Param{{Name: "v", ObjCType: "vFloat"}},
				Return: meta.ReturnType{ObjCType: "void"}, // void return is fine
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitBridgeImpl(&buf, framework, m, map[string]bool{}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Function with vFloat arg should be filtered by hasByValueUnknownType.
	if strings.Contains(out, "NSProcessFloat") {
		t.Errorf("vFloat arg function should be filtered by hasByValueUnknownType; got:\n%s", out)
	}
}
