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

func implTestMapper() *typemap.Mapper {
	return &typemap.Mapper{
		GenericClasses:     map[string]bool{},
		OwnerIndex:         map[string]string{"NSObject": "Foundation"},
		ModulePrefix:       "github.com/example/fw",
		BlockedImports:     map[string]map[string]bool{},
		TypedefIndex:       map[string]string{},
		StructIndex:        map[string]string{},
		ProtocolIndex:      map[string]string{},
		ProtocolProxyIndex: map[string]string{},
	}
}

// ── collectProtocolMethods ────────────────────────────────────────────────────

func TestCollectProtocolMethodsEmpty(t *testing.T) {
	m := implTestMapper()
	proto := macosplatformmetadata.Protocol{}
	result := collectProtocolMethods(proto, m)
	if len(result) != 0 {
		t.Errorf("expected 0 methods; got %d", len(result))
	}
}

func TestCollectProtocolMethodsIMPSafe(t *testing.T) {
	m := implTestMapper()
	proto := macosplatformmetadata.Protocol{
		Methods: []macosplatformmetadata.Method{
			{Selector: "doThing", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			{Selector: "initWith:", IsInit: true, Return: macosplatformmetadata.ReturnType{IsInstancetype: true}}, // excluded
			{Selector: "bar:", IsClassMethod: true, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},   // excluded
		},
	}
	result := collectProtocolMethods(proto, m)
	if len(result) != 1 {
		t.Errorf("expected 1 IMP-safe method; got %d", len(result))
	}
	if result[0].GoName != "DoThing" {
		t.Errorf("expected GoName=DoThing; got %q", result[0].GoName)
	}
}

func TestCollectProtocolMethodsDeduplicates(t *testing.T) {
	m := implTestMapper()
	// Two methods with same Go name (same selector duplicated) → only one.
	proto := macosplatformmetadata.Protocol{
		Methods: []macosplatformmetadata.Method{
			{Selector: "doThing", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			{Selector: "doThing", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
		},
	}
	result := collectProtocolMethods(proto, m)
	if len(result) != 1 {
		t.Errorf("expected 1 method after dedup; got %d", len(result))
	}
}

func TestCollectProtocolMethodsWithArg(t *testing.T) {
	m := implTestMapper()
	proto := macosplatformmetadata.Protocol{
		Methods: []macosplatformmetadata.Method{
			{
				Selector: "setFlag:",
				Params:   []macosplatformmetadata.Param{{Name: "flag", ObjCType: "BOOL"}},
				Return:   macosplatformmetadata.ReturnType{ObjCType: "void"},
			},
		},
	}
	result := collectProtocolMethods(proto, m)
	if len(result) != 1 {
		t.Errorf("expected 1 method; got %d", len(result))
	}
	if len(result[0].Sig.Params) == 0 {
		t.Error("expected at least one sig arg")
	}
}

// ── emitProtocolImplGo ────────────────────────────────────────────────────────

func TestEmitProtocolImplGoBasic(t *testing.T) {
	m := implTestMapper()
	methods := []overrideMethod{
		{GoName: "DoThing", Selector: "doThing", Sig: MethodSigModel{Name: "void", IsVoidRet: true, CReturnType: "void", ObjCEnc: "v@:"}},
	}
	var buf bytes.Buffer
	if err := emitProtocolImplGo(&buf, "mypkg", "MyPkg", "NSFooDelegate", methods, m); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "NSFooDelegateCallbacks") {
		t.Errorf("expected NSFooDelegateCallbacks; got:\n%s", out)
	}
	if !strings.Contains(out, "NewNSFooDelegateProtocolCallback") {
		t.Errorf("expected NewNSFooDelegateProtocolCallback; got:\n%s", out)
	}
	if !strings.Contains(out, "DoThing") {
		t.Errorf("expected DoThing field; got:\n%s", out)
	}
	if !strings.Contains(out, "//go:build darwin") {
		t.Errorf("expected darwin build tag; got:\n%s", out)
	}
}

func TestEmitProtocolImplGoWithProxy(t *testing.T) {
	m := implTestMapper()
	m.ProtocolProxyIndex["VZFoo"] = "Virtualization"
	methods := []overrideMethod{
		{GoName: "DoThing", Selector: "doThing", Sig: MethodSigModel{Name: "void", IsVoidRet: true, CReturnType: "void", ObjCEnc: "v@:"}},
	}
	var buf bytes.Buffer
	if err := emitProtocolImplGo(&buf, "virtualization", "Virtualization", "VZFoo", methods, m); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// With proxy: return type is *VZFooIDProtocol
	if !strings.Contains(out, "IDProtocol") {
		t.Errorf("with proxy, should return IDProtocol type; got:\n%s", out)
	}
}

func TestEmitProtocolImplGoWithoutProxy(t *testing.T) {
	m := implTestMapper()
	// No proxy → returns cgo.Object
	methods := []overrideMethod{
		{GoName: "DoThing", Selector: "doThing", Sig: MethodSigModel{Name: "void", IsVoidRet: true, CReturnType: "void", ObjCEnc: "v@:"}},
	}
	var buf bytes.Buffer
	if err := emitProtocolImplGo(&buf, "mypkg", "MyPkg", "NSFooDelegate", methods, m); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "cgo.Object") {
		t.Errorf("without proxy, return type should be cgo.Object; got:\n%s", out)
	}
}

// ── emitProtocolImplHeader ────────────────────────────────────────────────────

func TestEmitProtocolImplHeaderBasic(t *testing.T) {
	methods := []overrideMethod{
		{GoName: "DoThing", Selector: "doThing", Sig: MethodSigModel{Name: "void", IsVoidRet: true, CReturnType: "void", ObjCEnc: "v@:"}},
	}
	var buf bytes.Buffer
	emitProtocolImplHeader(&buf, "NSFooDelegate", methods)
	out := buf.String()
	if !strings.Contains(out, "goBridge_Proto_NSFooDelegate") {
		t.Errorf("expected goBridge_Proto_NSFooDelegate; got:\n%s", out)
	}
	if !strings.Contains(out, "#pragma once") {
		t.Errorf("expected #pragma once; got:\n%s", out)
	}
	if !strings.Contains(out, "goBridge_Trampoline_MethodFn_void") {
		t.Errorf("expected goBridge_Trampoline_MethodFn_void; got:\n%s", out)
	}
}

// ── emitProtocolImplImpl ──────────────────────────────────────────────────────

func TestEmitProtocolImplImplBasic(t *testing.T) {
	methods := []overrideMethod{
		{GoName: "DoThing", Selector: "doThing", Sig: MethodSigModel{Name: "void", IsVoidRet: true, CReturnType: "void", ObjCEnc: "v@:"}},
	}
	var buf bytes.Buffer
	emitProtocolImplImpl(&buf, "Foundation", "NSFooDelegate", methods)
	out := buf.String()
	if !strings.Contains(out, "#import <Foundation/Foundation.h>") {
		t.Errorf("expected framework import; got:\n%s", out)
	}
	if !strings.Contains(out, "NSFooDelegate_protocol_callback.h") {
		t.Errorf("expected generated header include; got:\n%s", out)
	}
	if !strings.Contains(out, "goBridge_Proto_NSFooDelegate_createClass") {
		t.Errorf("expected alloc function; got:\n%s", out)
	}
}

// ── EmitProtocolImpls (filesystem) ────────────────────────────────────────────

func TestEmitProtocolImplsEmpty(t *testing.T) {
	m := implTestMapper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bridge"), 0o755); err != nil {
		t.Fatal(err)
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Protocols: map[string]macosplatformmetadata.Protocol{},
	}
	if err := EmitProtocolImpls(dir, framework, m); err != nil {
		t.Fatal(err)
	}
	// No files should be written.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "_protocol_callback.go") {
			t.Errorf("unexpected generated file: %s", e.Name())
		}
	}
}

func TestEmitProtocolImplsUnavailableSkipped(t *testing.T) {
	m := implTestMapper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bridge"), 0o755); err != nil {
		t.Fatal(err)
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Protocols: map[string]macosplatformmetadata.Protocol{
			"NSFoo": {Availability: macosplatformmetadata.Availability{IsUnavailable: true}},
		},
	}
	if err := EmitProtocolImpls(dir, framework, m); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "_protocol_callback.go") {
			t.Errorf("unavailable protocol should be skipped; got %s", e.Name())
		}
	}
}

func TestEmitProtocolImplsWritesFiles(t *testing.T) {
	m := implTestMapper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bridge"), 0o755); err != nil {
		t.Fatal(err)
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Protocols: map[string]macosplatformmetadata.Protocol{
			"NSFooDelegate": {
				Methods: []macosplatformmetadata.Method{
					{Selector: "didFoo", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				},
			},
		},
	}
	if err := EmitProtocolImpls(dir, framework, m); err != nil {
		t.Fatal(err)
	}
	goPath := filepath.Join(dir, "NSFooDelegate_protocol_callback.go")
	if _, err := os.Stat(goPath); err != nil {
		t.Errorf("expected .go file written; got %v", err)
	}
	hPath := filepath.Join(dir, "bridge", "NSFooDelegate_protocol_callback.h")
	if _, err := os.Stat(hPath); err != nil {
		t.Errorf("expected .h file written; got %v", err)
	}
	mPath := filepath.Join(dir, "bridge", "NSFooDelegate_protocol_callback.m")
	if _, err := os.Stat(mPath); err != nil {
		t.Errorf("expected .m file written; got %v", err)
	}
}

func TestEmitProtocolImplsNoIMPSafeMethodsSkipped(t *testing.T) {
	m := implTestMapper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bridge"), 0o755); err != nil {
		t.Fatal(err)
	}
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Protocols: map[string]macosplatformmetadata.Protocol{
			"NSFooDelegate": {
				Methods: []macosplatformmetadata.Method{
					// class method → not IMP-safe
					{Selector: "shared", IsClassMethod: true, Return: macosplatformmetadata.ReturnType{ObjCType: "instancetype"}},
				},
			},
		},
	}
	if err := EmitProtocolImpls(dir, framework, m); err != nil {
		t.Fatal(err)
	}
	// No IMP-safe methods → no files written.
	if _, err := os.Stat(filepath.Join(dir, "NSFooDelegate_protocol_callback.go")); err == nil {
		t.Error("no IMP-safe methods should result in no file written")
	}
}
