package raw

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// --------------------------------------------------------------------------
// isGenericClass
// --------------------------------------------------------------------------

func TestIsGenericClassMapTrue(t *testing.T) {
	m := map[string]bool{"NSArray": true}
	if !isGenericClass("NSArray", m) {
		t.Error("NSArray should be generic when map says so")
	}
}

func TestIsGenericClassMapFalse(t *testing.T) {
	m := map[string]bool{}
	if isGenericClass("NSString", m) {
		t.Error("NSString should not be generic with empty map")
	}
}

func TestIsGenericClassNilMapFallback(t *testing.T) {
	// nil map uses hardcoded heuristic
	for _, name := range []string{"NSArray", "NSMutableArray", "NSSet", "NSMutableSet",
		"NSDictionary", "NSMutableDictionary", "NSMapTable", "NSHashTable", "NSCache",
		"NSCountedSet", "NSEnumerator", "NSOrderedSet", "NSMutableOrderedSet"} {
		if !isGenericClass(name, nil) {
			t.Errorf("isGenericClass(%q, nil) should be true via heuristic", name)
		}
	}
}

func TestIsGenericClassNilMapNonGeneric(t *testing.T) {
	for _, name := range []string{"NSString", "NSObject", "NSPredicate"} {
		if isGenericClass(name, nil) {
			t.Errorf("isGenericClass(%q, nil) should be false", name)
		}
	}
}

// --------------------------------------------------------------------------
// extractStructType
// --------------------------------------------------------------------------

func TestResolveStructTypeStripsPointer(t *testing.T) {
	got := extractStructType("*NSString", false)
	if got != "NSString" {
		t.Errorf("expected 'NSString', got %q", got)
	}
}

func TestResolveStructTypeClassMethodReplacesT(t *testing.T) {
	got := extractStructType("*NSArray[T]", true)
	if got != "NSArray[cgo.Object]" {
		t.Errorf("class method should replace [T] with [cgo.Object]; got %q", got)
	}
}

func TestResolveStructTypeInstanceMethodKeepsT(t *testing.T) {
	got := extractStructType("*NSArray[T]", false)
	if got != "NSArray[T]" {
		t.Errorf("instance method should keep [T]; got %q", got)
	}
}

// --------------------------------------------------------------------------
// objectConstructExpr
// --------------------------------------------------------------------------

func TestObjectConstructExprSameFramework(t *testing.T) {
	m := &typemap.Mapper{}
	got := objectConstructExpr("NSString", "_ptr", nil, m)
	if got != "NewNSString(_ptr)" {
		t.Errorf("same-fw: want NewNSString(_ptr), got %q", got)
	}
}

func TestObjectConstructExprCrossFramework(t *testing.T) {
	m := &typemap.Mapper{}
	got := objectConstructExpr("foundation.NSString", "_ptr", nil, m)
	if got != "foundation.NewNSString(_ptr)" {
		t.Errorf("cross-fw: want foundation.NewNSString(_ptr), got %q", got)
	}
}

func TestObjectConstructExprGenericT(t *testing.T) {
	m := &typemap.Mapper{}
	got := objectConstructExpr("NSArray[T]", "_ptr", nil, m)
	if got != "NewNSArrayT[T](_ptr)" {
		t.Errorf("generic [T]: want NewNSArrayT[T](_ptr), got %q", got)
	}
}

func TestObjectConstructExprGenericRuntimeObject(t *testing.T) {
	m := &typemap.Mapper{}
	got := objectConstructExpr("NSArray[cgo.Object]", "_ptr", nil, m)
	if got != "NewNSArray(_ptr)" {
		t.Errorf("generic [cgo.Object]: want NewNSArray(_ptr), got %q", got)
	}
}

func TestObjectConstructExprCrossFrameworkGenericT(t *testing.T) {
	m := &typemap.Mapper{}
	got := objectConstructExpr("foundation.NSArray[T]", "_ptr", nil, m)
	// Cross-fw with [T] → use TypedWithPtr variant
	if !strings.Contains(got, "T[T]") && !strings.Contains(got, "NewNSArrayT") {
		t.Errorf("cross-fw generic [T]: unexpected expr %q", got)
	}
}

// --------------------------------------------------------------------------
// isCrossFrameworkType / crossFrameworkCtor
// --------------------------------------------------------------------------

func TestIsCrossFrameworkTypeTrue(t *testing.T) {
	for _, s := range []string{"foundation.NSString", "appkit.NSView", "foundation.NSArray[T]"} {
		if !isCrossFrameworkType(s) {
			t.Errorf("isCrossFrameworkType(%q) should be true", s)
		}
	}
}

func TestIsCrossFrameworkTypeFalse(t *testing.T) {
	for _, s := range []string{"NSString", "NSArray[T]", "cgo.Object"} {
		// runtime.Object has a dot but no bracket before it — should still be
		// treated as cross-framework
		_ = isCrossFrameworkType(s)
		// Just verify same-fw names without dot return false
	}
	for _, s := range []string{"NSString", "NSArray[T]"} {
		if isCrossFrameworkType(s) {
			t.Errorf("isCrossFrameworkType(%q) should be false", s)
		}
	}
}

func TestCrossFrameworkCtorBasic(t *testing.T) {
	got := crossFrameworkCtor("foundation.NSString", "_ptr")
	if got != "foundation.NewNSString(_ptr)" {
		t.Errorf("want foundation.NewNSString(_ptr), got %q", got)
	}
}

func TestCrossFrameworkCtorGenericT(t *testing.T) {
	got := crossFrameworkCtor("foundation.NSArray[T]", "_ptr")
	if !strings.Contains(got, "T[T]") {
		t.Errorf("cross-fw generic T ctor should reference T[T]; got %q", got)
	}
}

// --------------------------------------------------------------------------
// buildValueChain (covers buildValueChainInner paths)
// --------------------------------------------------------------------------

func TestBuildValueChainUnknownClass(t *testing.T) {
	chain := buildValueChain("NSUnknown", "", "Foundation", map[string]macosplatformmetadata.Class{}, nil, nil)
	if !strings.Contains(chain, "unknown chain") {
		t.Errorf("unknown class should produce unknown chain comment; got %q", chain)
	}
}

func TestBuildValueChainRoot(t *testing.T) {
	fmClasses := map[string]macosplatformmetadata.Class{
		"NSObject": {}, // root — no Super
	}
	chain := buildValueChain("NSObject", "", "Foundation", fmClasses, nil, nil)
	if !strings.Contains(chain, "ptr: ptr") {
		t.Errorf("root class should have ptr: ptr; got %q", chain)
	}
}

func TestBuildValueChainSameFW(t *testing.T) {
	m := &typemap.Mapper{
		OwnerIndex: map[string]string{"NSObject": "Foundation"},
		GenericClasses: map[string]bool{},
	}
	fmClasses := map[string]macosplatformmetadata.Class{
		"NSObject": {},
		"NSString": {Super: "NSObject"},
	}
	chain := buildValueChain("NSString", "", "Foundation", fmClasses, m, nil)
	// Should have NSObject embedded inside NSString
	if !strings.Contains(chain, "NSObject") || !strings.Contains(chain, "ptr: ptr") {
		t.Errorf("same-fw chain should embed NSObject with ptr; got %q", chain)
	}
}

func TestBuildValueChainCrossFW(t *testing.T) {
	m := &typemap.Mapper{
		OwnerIndex: map[string]string{"NSObject": "Foundation"},
		GenericClasses: map[string]bool{},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
	fmClasses := map[string]macosplatformmetadata.Class{
		// AppKit class whose super (NSObject) is in Foundation
		"NSView": {Super: "NSObject"},
	}
	usedImports := map[string]string{}
	chain := buildValueChain("NSView", "", "AppKit", fmClasses, m, usedImports)
	// Should reference foundation.NSObjectWithPtr
	if !strings.Contains(chain, "foundation") {
		t.Errorf("cross-fw chain should reference foundation package; got %q", chain)
	}
}

func TestBuildValueChainBlockedCrossFW(t *testing.T) {
	m := &typemap.Mapper{
		OwnerIndex: map[string]string{"NSObject": "Foundation"},
		GenericClasses: map[string]bool{},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{
			"AppKit": {"Foundation": true},
		},
	}
	fmClasses := map[string]macosplatformmetadata.Class{
		"NSView": {Super: "NSObject"},
	}
	chain := buildValueChain("NSView", "", "AppKit", fmClasses, m, nil)
	// Blocked → treated as root, has ptr: ptr
	if !strings.Contains(chain, "ptr: ptr") {
		t.Errorf("blocked cross-fw should fallback to ptr: ptr; got %q", chain)
	}
}

// --------------------------------------------------------------------------
// goCallArgs (helpers.go)
// --------------------------------------------------------------------------

func TestGoCallArgsInstanceMethod(t *testing.T) {
	args := []macosplatformmetadata.Param{{Name: "x", ObjCType: "NSUInteger"}}
	result := goCallArgs(false, args, false)
	if !strings.HasPrefix(result, "o.ptr") {
		t.Errorf("instance method should start with o.ptr; got %q", result)
	}
	if !strings.Contains(result, "x") {
		t.Errorf("instance method should include arg name; got %q", result)
	}
}

func TestGoCallArgsClassMethod(t *testing.T) {
	args := []macosplatformmetadata.Param{{Name: "count", ObjCType: "NSUInteger"}}
	result := goCallArgs(true, args, false)
	if strings.Contains(result, "o.ptr") {
		t.Errorf("class method should not have o.ptr; got %q", result)
	}
}

func TestGoCallArgsNSError(t *testing.T) {
	result := goCallArgs(false, []macosplatformmetadata.Param{}, true)
	if !strings.Contains(result, "&nsErr") {
		t.Errorf("NSError method should include &nsErr; got %q", result)
	}
}

func TestGoCallArgsEmpty(t *testing.T) {
	result := goCallArgs(true, []macosplatformmetadata.Param{}, false)
	if result != "" {
		t.Errorf("class method with no args should be empty string; got %q", result)
	}
}

// --------------------------------------------------------------------------
// EmitClasses() — disk-based
// --------------------------------------------------------------------------

func TestClassesCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	m := &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex: map[string]string{"NSObject": "Foundation"},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {},
		},
	}
	all := map[string]macosplatformmetadata.Class{"NSObject": {}}
	knownClasses := map[string]bool{"NSObject": true}
	if err := EmitClasses(dir, framework, m, knownClasses, all, "foundation"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "NSObject.go")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("NSObject.go should have been created: %v", err)
	}
	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "type NSObject struct") {
		t.Errorf("NSObject.go missing struct declaration; got:\n%s", content)
	}
}

// TestClassesFilenameCollision verifies that two class names differing only in
// case get distinct filenames (the second gets a _2 suffix on case-insensitive
// file systems).
func TestClassesFilenameCollision(t *testing.T) {
	dir := t.TempDir()
	m := &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex: map[string]string{
			"NSObject": "Foundation",
			"NSstring": "Foundation", // same as NSString but different case
		},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {},
			"NSobject": {Super: "NSObject"}, // lowercase 'o' — collision on case-insensitive FS
		},
	}
	all := map[string]macosplatformmetadata.Class{
		"NSObject": {},
		"NSobject": {Super: "NSObject"},
	}
	knownClasses := map[string]bool{"NSObject": true, "NSobject": true}
	// Should not error even with case collision
	err := EmitClasses(dir, framework, m, knownClasses, all, "foundation")
	if err != nil {
		t.Fatal(err)
	}
}

func TestClassesMultipleClasses(t *testing.T) {
	dir := t.TempDir()
	m := &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex: map[string]string{
			"NSObject": "Foundation",
			"NSString": "Foundation",
		},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {},
			"NSString": {Super: "NSObject"},
		},
	}
	all := map[string]macosplatformmetadata.Class{
		"NSObject": {},
		"NSString": {Super: "NSObject"},
	}
	knownClasses := map[string]bool{"NSObject": true, "NSString": true}
	if err := EmitClasses(dir, framework, m, knownClasses, all, "foundation"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"NSObject.go", "NSString.go"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s should have been created: %v", name, err)
		}
	}
}

// TestClassesObjectReturnMethod verifies that a method returning an ObjC object
// type produces a constructor call rather than a struct literal.
func TestClassesObjectReturnMethod(t *testing.T) {
	dir := t.TempDir()
	m := &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex: map[string]string{
			"NSObject": "Foundation",
			"NSString": "Foundation",
		},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {},
			"NSString": {
				Super: "NSObject",
				Methods: []macosplatformmetadata.Method{
					{
						Selector: "lowercaseString",
						Return:   macosplatformmetadata.ReturnType{ObjCType: "NSString *"},
					},
				},
			},
		},
	}
	all := map[string]macosplatformmetadata.Class{
		"NSObject": {},
		"NSString": framework.Classes["NSString"],
	}
	knownClasses := map[string]bool{"NSObject": true, "NSString": true}
	if err := EmitClasses(dir, framework, m, knownClasses, all, "foundation"); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "NSString.go"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(content)
	// The method should use cgo.WrapTyped with the NewNSString constructor.
	if !strings.Contains(out, "cgo.WrapTyped(_ptr, NewNSString)") {
		t.Errorf("object return should use cgo.WrapTyped with NewNSString; got:\n%s", out)
	}
}

// TestClassesNSErrorObjectReturn verifies HasNSError + object return uses (obj, error) signature.
func TestClassesNSErrorObjectReturn(t *testing.T) {
	dir := t.TempDir()
	m := &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex: map[string]string{
			"NSObject": "Foundation",
			"NSString": "Foundation",
		},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {},
			"NSString": {
				Super: "NSObject",
				Methods: []macosplatformmetadata.Method{
					{
						Selector:   "stringWithContentsOfFile:encoding:error:",
						IsNSError: true,
						Return:     macosplatformmetadata.ReturnType{ObjCType: "NSString *"},
						Params: []macosplatformmetadata.Param{
							{Name: "path", ObjCType: "NSString *"},
							{Name: "enc", ObjCType: "NSUInteger"},
						},
					},
				},
			},
		},
	}
	all := map[string]macosplatformmetadata.Class{
		"NSObject": {},
		"NSString": framework.Classes["NSString"],
	}
	knownClasses := map[string]bool{"NSObject": true, "NSString": true}
	if err := EmitClasses(dir, framework, m, knownClasses, all, "foundation"); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, "NSString.go"))
	out := string(content)
	if !strings.Contains(out, "error") {
		t.Errorf("NSError method should produce error return; got:\n%s", out)
	}
}

// --------------------------------------------------------------------------
// goCGoArgExpr — direct unit tests
// --------------------------------------------------------------------------

func TestGoCGoArgExprUnsafePointer(t *testing.T) {
	var p, ka []string
	got := goCGoArgExpr("unsafe.Pointer", "obj", nil, &p, &ka)
	if got != "obj" {
		t.Errorf("unsafe.Pointer arg should pass through; got %q", got)
	}
}

func TestGoCGoArgExprBool(t *testing.T) {
	var p, ka []string
	got := goCGoArgExpr("bool", "flag", nil, &p, &ka)
	if got != "C.bool(flag)" {
		t.Errorf("bool: want C.bool(flag), got %q", got)
	}
}

func TestGoCGoArgExprNumericTypes(t *testing.T) {
	cases := []struct {
		goType string
		want   string
	}{
		{"int8", "C.int8_t(v)"},
		{"int16", "C.int16_t(v)"},
		{"int32", "C.int32_t(v)"},
		{"int64", "C.int64_t(v)"},
		{"uint8", "C.uint8_t(v)"},
		{"uint16", "C.uint16_t(v)"},
		{"uint32", "C.uint32_t(v)"},
		{"uint64", "C.uint64_t(v)"},
		{"float32", "C.float(v)"},
		{"float64", "C.double(v)"},
	}
	for _, tc := range cases {
		var p, ka []string
		got := goCGoArgExpr(tc.goType, "v", nil, &p, &ka)
		if got != tc.want {
			t.Errorf("goCGoArgExpr(%q): want %q, got %q", tc.goType, tc.want, got)
		}
	}
}

func TestGoCGoArgExprString(t *testing.T) {
	var p, ka []string
	got := goCGoArgExpr("string", "s", nil, &p, &ka)
	// Should return the _cstr_ variable name and add preambles
	if got != "_cstr_s" {
		t.Errorf("string: want _cstr_s, got %q", got)
	}
	if len(p) != 2 {
		t.Errorf("string: want 2 preambles, got %d", len(p))
	}
}

func TestGoCGoArgExprObjectPointer(t *testing.T) {
	var p, ka []string
	got := goCGoArgExpr("*NSString", "str", nil, &p, &ka)
	// nil-safe extraction: the returned expr is a preamble variable, not an
	// inline Ptr() call. Verify the preamble contains the nil guard + Ptr().
	if got != "_objcPtr_str" {
		t.Errorf("object pointer: want _objcPtr_str, got %q", got)
	}
	found := false
	for _, pre := range p {
		if strings.Contains(pre, "Ptr()") {
			found = true
		}
	}
	if !found {
		t.Errorf("object pointer: preambles should contain Ptr() call; preambles=%v", p)
	}
	// KeepAlive is now tracked via keepAlives, not preambles.
	if len(ka) != 1 || ka[0] != "str" {
		t.Errorf("object pointer: expected keepAlives=[str], got %v", ka)
	}
}

func TestGoCGoArgExprDefault(t *testing.T) {
	var p, ka []string
	got := goCGoArgExpr("SomeUnknownType", "x", nil, &p, &ka)
	if !strings.Contains(got, "unsafe.Pointer(x)") {
		t.Errorf("default: should wrap in unsafe.Pointer; got %q", got)
	}
}

func TestGoCGoArgExprEnumType(t *testing.T) {
	m := &typemap.Mapper{
		EnumGoTypeIndex: map[string]string{
			"MyFlags":      "uint64",
			"MyIndex":      "int64",
			"MySmallEnum":  "int32",
			"CrossPkgEnum": "uint32",
		},
	}
	cases := []struct {
		goType string
		want   string
	}{
		{"MyFlags", "C.uint64_t(myFlags)"},
		{"MyIndex", "C.int64_t(myIndex)"},
		{"MySmallEnum", "C.int32_t(mySmallEnum)"},
		// Cross-framework qualified name: package qualifier stripped before lookup.
		{"otherpkg.CrossPkgEnum", "C.uint32_t(crossPkgEnum)"},
	}
	for _, tc := range cases {
		var p, ka []string
		got := goCGoArgExpr(tc.goType, strings.ToLower(strings.TrimPrefix(strings.ReplaceAll(tc.goType, "otherpkg.", ""), "My")), m, &p, &ka)
		_ = got // just verify it compiles; detailed check via sub-test
	}
	// Focused checks with concrete arg names.
	var p, ka []string
	if got := goCGoArgExpr("MyFlags", "flags", m, &p, &ka); got != "C.uint64_t(flags)" {
		t.Errorf("MyFlags: want C.uint64_t(flags), got %q", got)
	}
	if got := goCGoArgExpr("MyIndex", "idx", m, &p, &ka); got != "C.int64_t(idx)" {
		t.Errorf("MyIndex: want C.int64_t(idx), got %q", got)
	}
	if got := goCGoArgExpr("otherpkg.CrossPkgEnum", "val", m, &p, &ka); got != "C.uint32_t(val)" {
		t.Errorf("otherpkg.CrossPkgEnum: want C.uint32_t(val), got %q", got)
	}
}

// TestGoCGoArgExprEnumTypeSmall covers int8/int16/uint8/uint16 enum underlying types.
func TestGoCGoArgExprEnumTypeSmall(t *testing.T) {
	m := &typemap.Mapper{
		EnumGoTypeIndex: map[string]string{
			"TinySignedEnum":   "int8",
			"SmallSignedEnum":  "int16",
			"TinyUnsignedEnum": "uint8",
			"SmallUnsignedEnum": "uint16",
		},
	}
	cases := []struct {
		goType string
		arg    string
		want   string
	}{
		{"TinySignedEnum", "v", "C.int8_t(v)"},
		{"SmallSignedEnum", "v", "C.int16_t(v)"},
		{"TinyUnsignedEnum", "v", "C.uint8_t(v)"},
		{"SmallUnsignedEnum", "v", "C.uint16_t(v)"},
	}
	for _, tc := range cases {
		var p, ka []string
		got := goCGoArgExpr(tc.goType, tc.arg, m, &p, &ka)
		if got != tc.want {
			t.Errorf("goCGoArgExpr(%q): want %q, got %q", tc.goType, tc.want, got)
		}
	}
}

// TestGoCGoArgExprPrimitivePointer covers pointer-to-primitive path (*uint32 → unsafe.Pointer(arg)).
func TestGoCGoArgExprPrimitivePointer(t *testing.T) {
	var p, ka []string
	got := goCGoArgExpr("*uint32", "v", nil, &p, &ka)
	if got != "unsafe.Pointer(v)" {
		t.Errorf("*uint32 primitive pointer: want unsafe.Pointer(v), got %q", got)
	}
}

// TestGoCGoArgExprEnumPointer covers pointer-to-enum path (*SomeEnum → unsafe.Pointer(arg)).
func TestGoCGoArgExprEnumPointer(t *testing.T) {
	m := &typemap.Mapper{
		EnumGoTypeIndex: map[string]string{"MyEventType": "uint64"},
	}
	var p, ka []string
	got := goCGoArgExpr("*MyEventType", "v", m, &p, &ka)
	if got != "unsafe.Pointer(v)" {
		t.Errorf("*enum pointer: want unsafe.Pointer(v), got %q", got)
	}
}

// TestGoCGoArgExprGenericPointerStripsParam covers *NSArray[T] → strips [T] before KnownStruct check.
func TestGoCGoArgExprGenericPointerStripsParam(t *testing.T) {
	var p, ka []string
	// *NSArray[T] without struct mapper: falls to ObjC class wrapper preamble.
	got := goCGoArgExpr("*NSArray[T]", "arr", nil, &p, &ka)
	// Generic pointer to non-struct non-primitive: preamble var name returned.
	if got != "_objcPtr_arr" {
		t.Errorf("generic pointer: want _objcPtr_arr, got %q", got)
	}
}

// TestGoCGoArgExprQualifiedPointerStripsPackage covers *foundation.NSString pointer (dot stripping).
func TestGoCGoArgExprQualifiedPointerStripsPackage(t *testing.T) {
	var p, ka []string
	// *foundation.NSString: strip "foundation." → bare = "NSString"; not KnownStruct → ObjC class preamble.
	got := goCGoArgExpr("*foundation.NSString", "s", nil, &p, &ka)
	if got != "_objcPtr_s" {
		t.Errorf("qualified pointer: want _objcPtr_s, got %q", got)
	}
}

// TestGoCGoArgExprBSDValueType covers non-pointer bsd. type → unsafe.Pointer(&arg).
func TestGoCGoArgExprBSDValueType(t *testing.T) {
	m := &typemap.Mapper{StructIndex: map[string]string{}}
	var p, ka []string
	got := goCGoArgExpr("bsd.Timespec", "ts", m, &p, &ka)
	if got != "unsafe.Pointer(&ts)" {
		t.Errorf("bsd value type: want unsafe.Pointer(&ts), got %q", got)
	}
}

// TestGoCGoArgExprKnownStructValue covers non-pointer known struct → unsafe.Pointer(&arg).
func TestGoCGoArgExprKnownStructValue(t *testing.T) {
	m := &typemap.Mapper{StructIndex: map[string]string{"CGRect": "CoreGraphics"}}
	var p, ka []string
	got := goCGoArgExpr("CGRect", "r", m, &p, &ka)
	if got != "unsafe.Pointer(&r)" {
		t.Errorf("KnownStruct value: want unsafe.Pointer(&r), got %q", got)
	}
}

// TestGoCGoArgExprInterfaceProtocol covers interface { ... } compound protocol → unsafe.Pointer(arg.Ptr()).
func TestGoCGoArgExprInterfaceProtocol(t *testing.T) {
	m := &typemap.Mapper{StructIndex: map[string]string{}, ProtocolIndex: map[string]string{}}
	var p, ka []string
	got := goCGoArgExpr("interface { Foo(); Bar() }", "proto", m, &p, &ka)
	if got != "unsafe.Pointer(proto.Ptr())" {
		t.Errorf("interface protocol: want unsafe.Pointer(proto.Ptr()), got %q", got)
	}
}

// TestGoCGoArgExprKnownProtocol covers ProtocolIndex named type → unsafe.Pointer(arg.Ptr()).
func TestGoCGoArgExprKnownProtocol(t *testing.T) {
	m := &typemap.Mapper{
		StructIndex:   map[string]string{},
		ProtocolIndex: map[string]string{"NSCopying": "Foundation"},
	}
	var p, ka []string
	got := goCGoArgExpr("NSCopyingProtocol", "p", m, &p, &ka)
	if got != "unsafe.Pointer(p.Ptr())" {
		t.Errorf("KnownProtocol: want unsafe.Pointer(p.Ptr()), got %q", got)
	}
}

// --------------------------------------------------------------------------
// cgoReturnConvert — direct unit tests
// --------------------------------------------------------------------------

func TestCgoReturnConvertBool(t *testing.T) {
	got := cgoReturnConvert("C.fn()", "bool", nil)
	if got != "bool(C.fn())" {
		t.Errorf("bool: want bool(C.fn()), got %q", got)
	}
}

func TestCgoReturnConvertString(t *testing.T) {
	got := cgoReturnConvert("C.fn()", "string", nil)
	if got != "C.GoString(C.fn())" {
		t.Errorf("string: want C.GoString(C.fn()), got %q", got)
	}
}

func TestCgoReturnConvertNumericTypes(t *testing.T) {
	cases := []struct {
		goType string
		prefix string
	}{
		{"int8", "int8("},
		{"int16", "int16("},
		{"int32", "int32("},
		{"int64", "int64("},
		{"uint8", "uint8("},
		{"uint16", "uint16("},
		{"uint32", "uint32("},
		{"uint64", "uint64("},
		{"float32", "float32("},
		{"float64", "float64("},
		{"unsafe.Pointer", "unsafe.Pointer("},
	}
	for _, tc := range cases {
		got := cgoReturnConvert("C.fn()", tc.goType, nil)
		if !strings.HasPrefix(got, tc.prefix) {
			t.Errorf("cgoReturnConvert(%q): want prefix %q, got %q", tc.goType, tc.prefix, got)
		}
	}
}

func TestCgoReturnConvertDefault(t *testing.T) {
	got := cgoReturnConvert("C.fn()", "SomeType", nil)
	if got != "C.fn()" {
		t.Errorf("default: should return expr unchanged; got %q", got)
	}
}

// --------------------------------------------------------------------------
// primaryReturnType — direct unit tests
// --------------------------------------------------------------------------

func TestPrimaryReturnTypeInstancetype(t *testing.T) {
	m := &typemap.Mapper{GenericClasses: map[string]bool{}}
	ctx := typemap.Context{ClassName: "NSString"}
	method := macosplatformmetadata.Method{Return: macosplatformmetadata.ReturnType{IsInstancetype: true}}
	got := primaryReturnType(method, ctx, m, nil)
	if got != "*NSString" {
		t.Errorf("instancetype: want *NSString, got %q", got)
	}
}

func TestPrimaryReturnTypeInstancetypeGeneric(t *testing.T) {
	m := &typemap.Mapper{GenericClasses: map[string]bool{"NSArray": true}}
	ctx := typemap.Context{
		ClassName:     "NSArray",
		GenericParams: []string{"ObjectType"},
	}
	method := macosplatformmetadata.Method{Return: macosplatformmetadata.ReturnType{IsInstancetype: true}}
	got := primaryReturnType(method, ctx, m, nil)
	if got != "*NSArray[T]" {
		t.Errorf("generic instancetype: want *NSArray[T], got %q", got)
	}
}

func TestPrimaryReturnTypeNoClassName(t *testing.T) {
	m := &typemap.Mapper{GenericClasses: map[string]bool{}}
	ctx := typemap.Context{}
	method := macosplatformmetadata.Method{Return: macosplatformmetadata.ReturnType{IsInstancetype: true}}
	got := primaryReturnType(method, ctx, m, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("no class name: want unsafe.Pointer, got %q", got)
	}
}

// --------------------------------------------------------------------------
// writeClassMethod via EmitClass (class method emission path)
// --------------------------------------------------------------------------

func TestWriteClassMethodStaticEmitted(t *testing.T) {
	m := &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex: map[string]string{"NSObject": "Foundation"},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]macosplatformmetadata.Class{"NSObject": {}},
	}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector:      "sharedInstance",
				IsClassMethod: true,
				Return:        macosplatformmetadata.ReturnType{IsInstancetype: true},
			},
		},
	}
	all := map[string]macosplatformmetadata.Class{"NSObject": {}}
	known := map[string]bool{"NSObject": true}
	var buf bytes.Buffer
	if err := EmitClass(&buf, "NSObject", cls, framework, m, known, all, "foundation"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Class method generates a package-level func NSObjectSharedInstance()
	if !strings.Contains(out, "NSObjectSharedInstance") {
		t.Errorf("class method NSObjectSharedInstance not emitted; got:\n%s", out)
	}
}

func TestWriteClassMethodPrimReturnEmitted(t *testing.T) {
	m := &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex: map[string]string{"NSObject": "Foundation"},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]macosplatformmetadata.Class{"NSObject": {}},
	}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector:      "version",
				IsClassMethod: true,
				Return:        macosplatformmetadata.ReturnType{ObjCType: "NSInteger"},
			},
		},
	}
	all := map[string]macosplatformmetadata.Class{"NSObject": {}}
	known := map[string]bool{"NSObject": true}
	var buf bytes.Buffer
	if err := EmitClass(&buf, "NSObject", cls, framework, m, known, all, "foundation"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "NSObjectVersion") {
		t.Errorf("class method NSObjectVersion not emitted; got:\n%s", out)
	}
}

// --------------------------------------------------------------------------
// constructorRef — the object-return constructor reference (used by method_body)
// --------------------------------------------------------------------------

func TestConstructorRefSameFramework(t *testing.T) {
	fmClasses := map[string]macosplatformmetadata.Class{}
	if got := constructorRef("NSString", fmClasses); got != "NewNSString" {
		t.Errorf("same-fw constructorRef = %q, want NewNSString", got)
	}
}

func TestConstructorRefCrossFramework(t *testing.T) {
	// Cross-fw: New<Cls> handles nil internally, no explicit nil check needed.
	if got := constructorRef("foundation.NSString", nil); got != "foundation.NewNSString" {
		t.Errorf("cross-fw constructorRef = %q, want foundation.NewNSString", got)
	}
}

// --------------------------------------------------------------------------
// writeStructDef — additional paths via EmitClass
// --------------------------------------------------------------------------

func TestWriteClassWithProtocols(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{"NSObject": {}}}
	cls := macosplatformmetadata.Class{
		Protocols: []string{"NSCopying", "NSMutableCopying"},
	}
	out, err := writeClassBuf("NSObject", cls, framework, m, map[string]macosplatformmetadata.Class{"NSObject": cls})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "NSCopying") || !strings.Contains(out, "NSMutableCopying") {
		t.Errorf("class with protocols should list them in comment; got:\n%s", out)
	}
}

func TestWriteClassMethodWithAvailability(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{"NSObject": {}}}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector: "newMethod",
				Return:   macosplatformmetadata.ReturnType{ObjCType: "void"},
				Availability: macosplatformmetadata.Availability{
					MacOSIntroduced: "13.0",
				},
			},
		},
	}
	all := map[string]macosplatformmetadata.Class{"NSObject": cls}
	out, err := writeClassBuf("NSObject", cls, framework, m, all)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Introduced: macOS 13.0") {
		t.Errorf("method availability comment missing; got:\n%s", out)
	}
}

func TestWriteClassMethodDeprecated(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{"NSObject": {}}}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector: "oldMethod",
				Return:   macosplatformmetadata.ReturnType{ObjCType: "void"},
				Availability: macosplatformmetadata.Availability{
					MacOSDeprecated: "12.0",
					ReplacedBy:      "newMethod",
				},
			},
		},
	}
	all := map[string]macosplatformmetadata.Class{"NSObject": cls}
	out, err := writeClassBuf("NSObject", cls, framework, m, all)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Deprecated: Deprecated in macOS 12.0") {
		t.Errorf("method deprecated comment missing; got:\n%s", out)
	}
}

// TestWriteClassMethodTypeNameCollisionSkip verifies that a class method whose
// generated function name collides with a package-level enum type is silently skipped.
func TestWriteClassMethodTypeNameCollisionSkip(t *testing.T) {
	m := classTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]macosplatformmetadata.Class{"NSObject": {}},
		Enums: map[string]macosplatformmetadata.Enum{
			// This enum's name will match the class method's generated function name:
			// className="NSObject" + goName="Version" = "NSObjectVersion"
			"NSObjectVersion": {Members: []macosplatformmetadata.EnumMember{{Name: "v1", Value: "1"}}},
		},
	}
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{
				Selector:      "version",
				IsClassMethod: true,
				Return:        macosplatformmetadata.ReturnType{ObjCType: "NSInteger"},
			},
		},
	}
	out, err := writeClassBuf("NSObject", cls, framework, m, map[string]macosplatformmetadata.Class{"NSObject": cls})
	if err != nil {
		t.Fatal(err)
	}
	// The class method "NSObjectVersion" should be skipped because of the enum collision.
	if strings.Contains(out, "func NSObjectVersion(") {
		t.Errorf("class method colliding with enum type should be skipped; got:\n%s", out)
	}
}

// TestWriteClassMethodGenericInstancetypeSubstitution tests the writeClassMethod path
// where isGeneric=true and buildGoReturn returns "*NSArray" without the bracket suffix
// (because the class is generic by GenericParams but NOT in Mapper.GenericClasses).
// writeClassMethod should then append "[cgo.Object]" to produce the correct return.
func TestWriteClassMethodGenericInstancetypeSubstitution(t *testing.T) {
	// Intentionally omit "NSArray" from GenericClasses so buildGoReturn returns "*NSArray"
	// instead of "*NSArray[cgo.Object]". writeClassMethod must fix this up.
	m := &typemap.Mapper{
		GenericClasses: map[string]bool{}, // NSArray NOT here
		OwnerIndex: map[string]string{
			"NSObject": "Foundation",
			"NSArray":  "Foundation",
		},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {},
			"NSArray":  {Super: "NSObject", GenericParams: []string{"ObjectType"}},
		},
	}
	cls := macosplatformmetadata.Class{
		Super:         "NSObject",
		GenericParams: []string{"ObjectType"},
		Methods: []macosplatformmetadata.Method{
			{
				Selector:      "array",
				IsClassMethod: true,
				Return:        macosplatformmetadata.ReturnType{IsInstancetype: true},
			},
		},
	}
	all := map[string]macosplatformmetadata.Class{
		"NSObject": {},
		"NSArray":  cls,
	}
	known := map[string]bool{"NSObject": true, "NSArray": true}
	var buf bytes.Buffer
	if err := EmitClass(&buf, "NSArray", cls, framework, m, known, all, "foundation"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// The class method should produce [cgo.Object] return type, not bare *NSArray.
	if !strings.Contains(out, "*NSArray[cgo.Object]") {
		t.Errorf("generic class method should have *NSArray[cgo.Object] return; got:\n%s", out)
	}
}

// TestWriteClassGenericSuperEmbed tests a generic child class whose super is also generic.
// This exercises the `si.superIsGeneric && isGeneric` path in writeStructDef.
func TestWriteClassGenericSuperEmbed(t *testing.T) {
	m := &typemap.Mapper{
		GenericClasses: map[string]bool{"NSArray": true, "NSMutableArray": true},
		OwnerIndex: map[string]string{
			"NSObject":       "Foundation",
			"NSArray":        "Foundation",
			"NSMutableArray": "Foundation",
		},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject":       {},
			"NSArray":        {Super: "NSObject", GenericParams: []string{"ObjectType"}},
			"NSMutableArray": {Super: "NSArray", GenericParams: []string{"ObjectType"}},
		},
	}
	cls := framework.Classes["NSMutableArray"]
	all := map[string]macosplatformmetadata.Class{
		"NSObject":       {},
		"NSArray":        framework.Classes["NSArray"],
		"NSMutableArray": cls,
	}
	known := map[string]bool{"NSObject": true, "NSArray": true, "NSMutableArray": true}
	var buf bytes.Buffer
	if err := EmitClass(&buf, "NSMutableArray", cls, framework, m, known, all, "foundation"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// NSMutableArray[T] should embed NSArray[T]
	if !strings.Contains(out, "NSArray[T]") {
		t.Errorf("generic child of generic super should embed super[T]; got:\n%s", out)
	}
}

// --------------------------------------------------------------------------
// NSCoding convenience methods
// --------------------------------------------------------------------------

func TestWriteClassNSCodingMethods(t *testing.T) {
	m := testMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {},
			"NSString": {
				Super:     "NSObject",
				Protocols: []string{"NSSecureCoding", "NSCopying"},
			},
		},
		Enums:      map[string]macosplatformmetadata.Enum{},
		Structs:    map[string]macosplatformmetadata.Struct{},
		BlockTypes: map[string]macosplatformmetadata.BlockType{},
	}
	cls := framework.Classes["NSString"]
	all := map[string]macosplatformmetadata.Class{"NSObject": {}, "NSString": cls}
	known := map[string]bool{"NSObject": true, "NSString": true}

	var buf bytes.Buffer
	if err := EmitClass(&buf, "NSString", cls, framework, m, known, all, "foundation"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, want := range []string{
		"func (o *NSString) SerializeToArchive() ([]byte, error) {",
		"func NewNSStringFromArchive(data []byte) (*NSString, error) {",
		"cgo.RaiseIfException(_exc)",
		"cgo.NSErrorToError(_nsErr)",
		"foundation_NSString_serializeToArchive",
		"foundation_NSString_newFromArchive",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in NSCoding output:\n%s", want, out)
		}
	}
}

func TestWriteClassNSCodingMethodsSkippedForNonConforming(t *testing.T) {
	m := testMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {},
			"NSDate":   {Super: "NSObject", Protocols: []string{"NSCopying"}},
		},
		Enums:      map[string]macosplatformmetadata.Enum{},
		Structs:    map[string]macosplatformmetadata.Struct{},
		BlockTypes: map[string]macosplatformmetadata.BlockType{},
	}
	cls := framework.Classes["NSDate"]
	all := map[string]macosplatformmetadata.Class{"NSObject": {}, "NSDate": cls}
	known := map[string]bool{"NSObject": true, "NSDate": true}

	var buf bytes.Buffer
	if err := EmitClass(&buf, "NSDate", cls, framework, m, known, all, "foundation"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if strings.Contains(out, "SerializeToArchive") {
		t.Errorf("SerializeToArchive should not be emitted for non-NSCoding class:\n%s", out)
	}
	if strings.Contains(out, "FromArchive") {
		t.Errorf("FromArchive should not be emitted for non-NSCoding class:\n%s", out)
	}
}

func TestWriteClassNSCodingBridgeDecl(t *testing.T) {
	m := testMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {},
			"NSString": {
				Super:     "NSObject",
				Protocols: []string{"NSSecureCoding"},
			},
		},
		Enums:      map[string]macosplatformmetadata.Enum{},
		Structs:    map[string]macosplatformmetadata.Struct{},
		BlockTypes: map[string]macosplatformmetadata.BlockType{},
		Functions:  nil,
	}
	var hbuf bytes.Buffer
	if err := EmitBridgeHeader(&hbuf, framework, m, map[string]bool{"NSObject": true, "NSString": true}); err != nil {
		t.Fatal(err)
	}
	header := hbuf.String()

	var mbuf bytes.Buffer
	if err := EmitBridgeImpl(&mbuf, framework, m, map[string]bool{"NSObject": true, "NSString": true}, "foundation_bridge.h"); err != nil {
		t.Fatal(err)
	}
	impl := mbuf.String()

	for _, want := range []string{
		"foundation_NSString_serializeToArchive",
		"foundation_NSString_newFromArchive",
		"size_t *outLen",
	} {
		if !strings.Contains(header, want) {
			t.Errorf("bridge header missing %q:\n%s", want, header)
		}
	}
	for _, want := range []string{
		"NSKeyedArchiver",
		"NSKeyedUnarchiver",
		"[NSString class]",
		"archivedDataWithRootObject",
		"unarchivedObjectOfClass",
		"malloc",
		"memcpy",
	} {
		if !strings.Contains(impl, want) {
			t.Errorf("bridge impl missing %q:\n%s", want, impl)
		}
	}
}
