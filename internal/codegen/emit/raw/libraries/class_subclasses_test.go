package rawlib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

func subclassTestMapper() *typemap.Mapper {
	return &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex:     map[string]string{"NSObject": "Foundation"},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
		TypedefIndex:   map[string]string{},
		StructIndex:    map[string]string{},
		ProtocolIndex:  map[string]string{},
	}
}

// ── uniqueIMPSigsFor ──────────────────────────────────────────────────────────

func TestUniqueIMPSigsForEmpty(t *testing.T) {
	got := uniqueIMPSigsFor(nil)
	if len(got) != 0 {
		t.Errorf("expected empty; got %v", got)
	}
}

func TestUniqueIMPSigsForDeduplicates(t *testing.T) {
	methods := []overrideMethod{
		{Sig: MethodSigModel{Name: "void"}},
		{Sig: MethodSigModel{Name: "void"}},
		{Sig: MethodSigModel{Name: "ptr"}},
	}
	got := uniqueIMPSigsFor(methods)
	if len(got) != 2 {
		t.Errorf("expected 2 unique sigs; got %d: %v", len(got), got)
	}
}

func TestUniqueIMPSigsForSorted(t *testing.T) {
	methods := []overrideMethod{
		{Sig: MethodSigModel{Name: "void_ptr"}},
		{Sig: MethodSigModel{Name: "bool"}},
		{Sig: MethodSigModel{Name: "ptr"}},
	}
	got := uniqueIMPSigsFor(methods)
	for i := 1; i < len(got); i++ {
		if got[i] < got[i-1] {
			t.Errorf("expected sorted output; got %v", got)
		}
	}
}

// ── impGetterName ─────────────────────────────────────────────────────────────

func TestImpGetterName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"void", "goBridge_Trampoline_MethodFn_void"},
		{"void_ptr", "goBridge_Trampoline_MethodFn_void_ptr"},
		{"int64", "goBridge_Trampoline_MethodFn_int64"},
	}
	for _, c := range cases {
		if got := impGetterName(c.in); got != c.want {
			t.Errorf("impGetterName(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

// ── collectOverridableMethods ─────────────────────────────────────────────────

func TestCollectOverridableMethodsEmpty(t *testing.T) {
	m := subclassTestMapper()
	cls := macosplatformmetadata.Class{}
	result := collectOverridableMethods(cls, m, map[string]macosplatformmetadata.Class{})
	if len(result) != 0 {
		t.Errorf("expected 0 methods; got %d", len(result))
	}
}

func TestCollectOverridableMethodsIMPSafe(t *testing.T) {
	m := subclassTestMapper()
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{Selector: "doThing", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			{Selector: "init", IsInit: true, Return: macosplatformmetadata.ReturnType{IsInstancetype: true}},
		},
	}
	result := collectOverridableMethods(cls, m, map[string]macosplatformmetadata.Class{})
	if len(result) != 1 {
		t.Errorf("expected 1 IMP-safe method; got %d", len(result))
	}
}

func TestCollectOverridableMethodsWalksChain(t *testing.T) {
	m := subclassTestMapper()
	// NSResponder sits between NSFoo and NSObject, so its methods should appear.
	// NSObject itself is the root (Super == "") and must not contribute methods.
	allClasses := map[string]macosplatformmetadata.Class{
		"NSObject": {
			// Super == "" marks this as the root class. Its methods must be excluded.
			Methods: []macosplatformmetadata.Method{
				{Selector: "description", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			},
		},
		"NSResponder": {
			Super: "NSObject",
			Methods: []macosplatformmetadata.Method{
				{Selector: "mouseDown:", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			},
		},
		"NSFoo": {
			Super: "NSResponder",
			Methods: []macosplatformmetadata.Method{
				{Selector: "doFoo", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			},
		},
	}
	cls := allClasses["NSFoo"]
	result := collectOverridableMethods(cls, m, allClasses)
	names := make(map[string]bool)
	for _, r := range result {
		names[r.GoName] = true
	}
	if !names["DoFoo"] {
		t.Error("expected DoFoo from NSFoo")
	}
	if !names["MouseDown"] {
		t.Error("expected MouseDown from NSResponder (framework ancestor)")
	}
	if names["Description"] {
		t.Error("NSObject is the root class — its methods must not appear in the overrides struct")
	}
}

func TestCollectOverridableMethodsExcludesRootClass(t *testing.T) {
	m := subclassTestMapper()
	// A class that directly extends NSObject with no own methods should
	// produce an empty result — no subclass factory should be generated.
	allClasses := map[string]macosplatformmetadata.Class{
		"NSObject": {
			Methods: []macosplatformmetadata.Method{
				{Selector: "description", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			},
		},
		"VZConfig": {
			Super: "NSObject",
			// No own methods.
		},
	}
	cls := allClasses["VZConfig"]
	result := collectOverridableMethods(cls, m, allClasses)
	if len(result) != 0 {
		t.Errorf("expected 0 methods for a class with no own methods extending NSObject; got %d", len(result))
	}
}

func TestCollectOverridableMethodsDeduplicates(t *testing.T) {
	m := subclassTestMapper()
	cls := macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{Selector: "doThing", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			{Selector: "doThing", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
		},
	}
	result := collectOverridableMethods(cls, m, map[string]macosplatformmetadata.Class{})
	if len(result) != 1 {
		t.Errorf("expected 1 method after dedup; got %d", len(result))
	}
}

// ── emitSubclassGo ────────────────────────────────────────────────────────────

func TestEmitSubclassGoBasic(t *testing.T) {
	m := subclassTestMapper()
	methods := []overrideMethod{
		{GoName: "MouseDown", Selector: "mouseDown:", Sig: MethodSigModel{Name: "void_ptr", IsVoidRet: true, CReturnType: "void", ObjCEnc: "v@:@"}},
	}
	var buf bytes.Buffer
	if err := emitSubclassGo(&buf, "appkit", "AppKit", "NSView", methods, m); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "NSViewOverrides") {
		t.Errorf("expected NSViewOverrides struct; got:\n%s", out)
	}
	if !strings.Contains(out, "NewNSViewSubclass") {
		t.Errorf("expected NewNSViewSubclass function; got:\n%s", out)
	}
	if !strings.Contains(out, "goBridge_Sub_NSView") {
		t.Errorf("expected goBridge_Sub_NSView; got:\n%s", out)
	}
	if !strings.Contains(out, "MouseDown") {
		t.Errorf("expected MouseDown field; got:\n%s", out)
	}
}

func TestEmitSubclassGoHeader(t *testing.T) {
	m := subclassTestMapper()
	methods := []overrideMethod{
		{GoName: "DoThing", Selector: "doThing", Sig: MethodSigModel{Name: "void", IsVoidRet: true, CReturnType: "void", ObjCEnc: "v@:"}},
	}
	var buf bytes.Buffer
	if err := emitSubclassGo(&buf, "foundation", "Foundation", "NSFoo", methods, m); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Code generated by go-bindings-codegen") {
		t.Errorf("expected generated header; got:\n%s", out)
	}
	if !strings.Contains(out, "//go:build darwin") {
		t.Errorf("expected darwin build tag; got:\n%s", out)
	}
}

// ── emitSubclassHeader ────────────────────────────────────────────────────────

func TestEmitSubclassHeaderBasic(t *testing.T) {
	methods := []overrideMethod{
		{Sig: MethodSigModel{Name: "void"}},
	}
	var buf bytes.Buffer
	emitSubclassHeader(&buf, "NSView", methods)
	out := buf.String()
	if !strings.Contains(out, "goBridge_Sub_NSView_createClass") {
		t.Errorf("expected alloc declaration; got:\n%s", out)
	}
	if !strings.Contains(out, "#pragma once") {
		t.Errorf("expected #pragma once; got:\n%s", out)
	}
	if !strings.Contains(out, "goBridge_Trampoline_MethodFn_void") {
		t.Errorf("expected goBridge_Trampoline_MethodFn_void; got:\n%s", out)
	}
}

// ── emitSubclassImpl ──────────────────────────────────────────────────────────

func TestEmitSubclassImplBasic(t *testing.T) {
	methods := []overrideMethod{
		{Sig: MethodSigModel{Name: "void"}},
	}
	var buf bytes.Buffer
	emitSubclassImpl(&buf, "AppKit", "NSView", methods)
	out := buf.String()
	if !strings.Contains(out, "#import <AppKit/AppKit.h>") {
		t.Errorf("expected framework import; got:\n%s", out)
	}
	if !strings.Contains(out, "NSView_subclass.h") {
		t.Errorf("expected generated header include; got:\n%s", out)
	}
	if !strings.Contains(out, "goBridge_Sub_NSView_createClass") {
		t.Errorf("expected alloc function; got:\n%s", out)
	}
	if !strings.Contains(out, "objc_allocateClassPair") {
		t.Errorf("expected objc_allocateClassPair; got:\n%s", out)
	}
}

// ── EmitSubclassFactories (filesystem) ───────────────────────────────────────

func TestEmitSubclassFactoriesEmpty(t *testing.T) {
	m := subclassTestMapper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bridge"), 0o755); err != nil {
		t.Fatal(err)
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes:   map[string]macosplatformmetadata.Class{},
	}
	if err := EmitSubclassFactories(dir, framework, m, map[string]bool{}, map[string]macosplatformmetadata.Class{}); err != nil {
		t.Fatal(err)
	}
}

func TestEmitSubclassFactoriesNotInSuperIndex(t *testing.T) {
	m := subclassTestMapper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bridge"), 0o755); err != nil {
		t.Fatal(err)
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSFoo": {Methods: []macosplatformmetadata.Method{
				{Selector: "doThing", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			}},
		},
	}
	// NSFoo not in superIndex → skip.
	if err := EmitSubclassFactories(dir, framework, m, map[string]bool{}, map[string]macosplatformmetadata.Class{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "NSFoo_subclass.go")); err == nil {
		t.Error("NSFoo not in superIndex should not generate file")
	}
}

func TestEmitSubclassFactoriesWritesFiles(t *testing.T) {
	m := subclassTestMapper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bridge"), 0o755); err != nil {
		t.Fatal(err)
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSFoo": {Methods: []macosplatformmetadata.Method{
				{Selector: "doThing", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			}},
		},
	}
	superIndex := map[string]bool{"NSFoo": true}
	allClasses := map[string]macosplatformmetadata.Class{"NSFoo": framework.Classes["NSFoo"]}
	if err := EmitSubclassFactories(dir, framework, m, superIndex, allClasses); err != nil {
		t.Fatal(err)
	}
	goPath := filepath.Join(dir, "NSFoo_subclass.go")
	if _, err := os.Stat(goPath); err != nil {
		t.Errorf("expected .go file; got %v", err)
	}
	hPath := filepath.Join(dir, "bridge", "NSFoo_subclass.h")
	if _, err := os.Stat(hPath); err != nil {
		t.Errorf("expected .h file; got %v", err)
	}
	mPath := filepath.Join(dir, "bridge", "NSFoo_subclass.m")
	if _, err := os.Stat(mPath); err != nil {
		t.Errorf("expected .m file; got %v", err)
	}
}

func TestEmitSubclassFactoriesUnavailableSkipped(t *testing.T) {
	m := subclassTestMapper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bridge"), 0o755); err != nil {
		t.Fatal(err)
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSFoo": {
				Availability: macosplatformmetadata.Availability{IsUnavailable: true},
				Methods:      []macosplatformmetadata.Method{{Selector: "doThing", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}}},
			},
		},
	}
	superIndex := map[string]bool{"NSFoo": true}
	if err := EmitSubclassFactories(dir, framework, m, superIndex, map[string]macosplatformmetadata.Class{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "NSFoo_subclass.go")); err == nil {
		t.Error("unavailable class should be skipped")
	}
}

func TestEmitSubclassFactoriesGenericSkipped(t *testing.T) {
	m := subclassTestMapper()
	m.GenericClasses = map[string]bool{"NSArray": true}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bridge"), 0o755); err != nil {
		t.Fatal(err)
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSArray": {
				GenericParams: []string{"ObjectType"},
				Methods:       []macosplatformmetadata.Method{{Selector: "count", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}}},
			},
		},
	}
	superIndex := map[string]bool{"NSArray": true}
	if err := EmitSubclassFactories(dir, framework, m, superIndex, map[string]macosplatformmetadata.Class{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "NSArray_subclass.go")); err == nil {
		t.Error("generic class should be skipped")
	}
}

// ── EmitGeneratedBridgesImpl ──────────────────────────────────────────────────

func TestEmitGeneratedBridgesImplNoBridgeDir(t *testing.T) {
	dir := t.TempDir()
	// No bridge/ dir → no error, no file.
	if err := EmitGeneratedBridgesImpl(dir, "foundation"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "foundation_impl.m")); err == nil {
		t.Error("no bridge dir → no generated impl file expected")
	}
}

func TestEmitGeneratedBridgesImplNoMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	bridgeDir := filepath.Join(dir, "bridge")
	if err := os.MkdirAll(bridgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// EmitBridge dir exists but no _generated.m files.
	if err := EmitGeneratedBridgesImpl(dir, "foundation"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "foundation_impl.m")); err == nil {
		t.Error("no generated .m files → no impl file expected")
	}
}

func TestEmitGeneratedBridgesImplWithSubclassFile(t *testing.T) {
	dir := t.TempDir()
	bridgeDir := filepath.Join(dir, "bridge")
	if err := os.MkdirAll(bridgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a fake _subclass.m file.
	if err := os.WriteFile(filepath.Join(bridgeDir, "NSView_subclass.m"), []byte("// stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EmitGeneratedBridgesImpl(dir, "appkit"); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "appkit_impl.m"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(content)
	if !strings.Contains(out, "NSView_subclass.m") {
		t.Errorf("expected NSView_subclass.m in impl; got:\n%s", out)
	}
	if !strings.Contains(out, "#pragma clang diagnostic push") {
		t.Errorf("expected pragma diagnostic; got:\n%s", out)
	}
}

func TestEmitGeneratedBridgesImplWithImplFile(t *testing.T) {
	dir := t.TempDir()
	bridgeDir := filepath.Join(dir, "bridge")
	if err := os.MkdirAll(bridgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a fake _impl.m file.
	if err := os.WriteFile(filepath.Join(bridgeDir, "NSFooDelegate_protocol_callback.m"), []byte("// stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EmitGeneratedBridgesImpl(dir, "foundation"); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "foundation_impl.m"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(content)
	if !strings.Contains(out, "NSFooDelegate_protocol_callback.m") {
		t.Errorf("expected impl file included; got:\n%s", out)
	}
}

// ── emitSubclassGo with multiple IMP sigs ────────────────────────────────────

func TestEmitSubclassGoMultipleMethods(t *testing.T) {
	m := subclassTestMapper()
	methods := []overrideMethod{
		{GoName: "MouseDown", Selector: "mouseDown:", Sig: MethodSigModel{Name: "void_ptr", IsVoidRet: true, CReturnType: "void", ObjCEnc: "v@:@"}},
		{GoName: "MouseUp", Selector: "mouseUp:", Sig: MethodSigModel{Name: "void_ptr", IsVoidRet: true, CReturnType: "void", ObjCEnc: "v@:@"}},
	}
	var buf bytes.Buffer
	if err := emitSubclassGo(&buf, "appkit", "AppKit", "NSView", methods, m); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "MouseDown") {
		t.Errorf("expected MouseDown; got:\n%s", out)
	}
	if !strings.Contains(out, "MouseUp") {
		t.Errorf("expected MouseUp; got:\n%s", out)
	}
	// Both use the same sig name → only one CGo comment declaration (deduped).
	count := strings.Count(out, "// extern void* goBridge_Trampoline_MethodFn_void_ptr")
	if count != 1 {
		t.Errorf("expected 1 goBridge_Trampoline_MethodFn_void_ptr comment decl (deduped); got %d", count)
	}
}
