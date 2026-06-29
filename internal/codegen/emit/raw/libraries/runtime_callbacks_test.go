package rawlib

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

func methodTrampolineMapper() *typemap.Mapper {
	return &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex:     map[string]string{},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
		TypedefIndex:   map[string]string{},
		StructIndex:    map[string]string{},
	}
}

// ── isKnownCPrimitive ─────────────────────────────────────────────────────────

func TestIsKnownCPrimitiveKnown(t *testing.T) {
	known := []string{"", "void", "void *", "bool", "int8_t", "int16_t", "int32_t",
		"int64_t", "uint8_t", "uint16_t", "uint32_t", "uint64_t",
		"float", "double", "const char *", "NSInteger_t"}
	for _, c := range known {
		if !isKnownCPrimitive(c) {
			t.Errorf("isKnownCPrimitive(%q) = false; want true", c)
		}
	}
}

func TestIsKnownCPrimitiveUnknown(t *testing.T) {
	unknown := []string{"CGRect", "NSPoint", "struct Foo"}
	for _, c := range unknown {
		if isKnownCPrimitive(c) {
			t.Errorf("isKnownCPrimitive(%q) = true; want false", c)
		}
	}
}

// ── isMethodIMPSafe ───────────────────────────────────────────────────────────

func TestIsMethodIMPSafeVoidNoArgs(t *testing.T) {
	m := methodTrampolineMapper()
	method := macosplatformmetadata.Method{
		Selector: "doThing",
		Return:   macosplatformmetadata.ReturnType{ObjCType: "void"},
	}
	if !isMethodIMPSafe(method, m) {
		t.Error("simple void method should be IMP-safe")
	}
}

func TestIsMethodIMPSafeClassMethodExcluded(t *testing.T) {
	m := methodTrampolineMapper()
	method := macosplatformmetadata.Method{Selector: "shared", IsClassMethod: true, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}}
	if isMethodIMPSafe(method, m) {
		t.Error("class method should not be IMP-safe")
	}
}

func TestIsMethodIMPSafeInitExcluded(t *testing.T) {
	m := methodTrampolineMapper()
	method := macosplatformmetadata.Method{Selector: "init", IsInit: true, Return: macosplatformmetadata.ReturnType{ObjCType: "instancetype"}}
	if isMethodIMPSafe(method, m) {
		t.Error("init method should not be IMP-safe")
	}
}

func TestIsMethodIMPSafeInitSelectorExcluded(t *testing.T) {
	m := methodTrampolineMapper()
	method := macosplatformmetadata.Method{Selector: "initWithFrame:", Return: macosplatformmetadata.ReturnType{ObjCType: "instancetype"}}
	if isMethodIMPSafe(method, m) {
		t.Error("initWithFrame: method should not be IMP-safe (init prefix)")
	}
}

func TestIsMethodIMPSafeNSErrorExcluded(t *testing.T) {
	m := methodTrampolineMapper()
	method := macosplatformmetadata.Method{Selector: "doThing", IsNSError: true, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}}
	if isMethodIMPSafe(method, m) {
		t.Error("NSError method should not be IMP-safe")
	}
}

func TestIsMethodIMPSafeVariadicExcluded(t *testing.T) {
	m := methodTrampolineMapper()
	method := macosplatformmetadata.Method{Selector: "logFormat:", IsVariadic: true, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}}
	if isMethodIMPSafe(method, m) {
		t.Error("variadic method should not be IMP-safe")
	}
}

func TestIsMethodIMPSafeStructReturnTreatedAsPointer(t *testing.T) {
	m := methodTrampolineMapper()
	// Struct return types map to void* via CType and are therefore IMP-safe.
	method := macosplatformmetadata.Method{Selector: "frame", Return: macosplatformmetadata.ReturnType{ObjCType: "CGRect"}}
	if !isMethodIMPSafe(method, m) {
		t.Error("struct return type maps to void* and should be IMP-safe")
	}
}

func TestIsMethodIMPSafeWithPrimitiveArgs(t *testing.T) {
	m := methodTrampolineMapper()
	method := macosplatformmetadata.Method{
		Selector: "setFlag:",
		Params:   []macosplatformmetadata.Param{{Name: "flag", ObjCType: "BOOL"}},
		Return:   macosplatformmetadata.ReturnType{ObjCType: "void"},
	}
	if !isMethodIMPSafe(method, m) {
		t.Error("method with BOOL arg should be IMP-safe")
	}
}

// ── methodSigFromMethod ───────────────────────────────────────────────────────

func TestMethodSigFromMethodVoid(t *testing.T) {
	m := methodTrampolineMapper()
	method := macosplatformmetadata.Method{Selector: "doThing", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}}
	sig, ok := methodSigFromMethod(method, m)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !sig.IsVoidRet {
		t.Error("expected IsVoidRet=true")
	}
	if sig.Name != "void" {
		t.Errorf("sig.Name = %q; want \"void\"", sig.Name)
	}
}

func TestMethodSigFromMethodWithArg(t *testing.T) {
	m := methodTrampolineMapper()
	method := macosplatformmetadata.Method{
		Selector: "setFlag:",
		Params:   []macosplatformmetadata.Param{{Name: "flag", ObjCType: "BOOL"}},
		Return:   macosplatformmetadata.ReturnType{ObjCType: "void"},
	}
	sig, ok := methodSigFromMethod(method, m)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if len(sig.Params) == 0 {
		t.Error("expected at least one arg")
	}
	if !strings.Contains(sig.Name, "void") {
		t.Errorf("sig.Name should contain 'void'; got %q", sig.Name)
	}
}

func TestMethodSigFromMethodNonVoidReturn(t *testing.T) {
	m := methodTrampolineMapper()
	method := macosplatformmetadata.Method{
		Selector: "count",
		Return:   macosplatformmetadata.ReturnType{ObjCType: "NSUInteger"},
	}
	sig, ok := methodSigFromMethod(method, m)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if sig.IsVoidRet {
		t.Error("expected IsVoidRet=false for non-void return")
	}
}

func TestMethodSigFromMethodObjCEncoding(t *testing.T) {
	m := methodTrampolineMapper()
	method := macosplatformmetadata.Method{Selector: "doThing", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}}
	sig, ok := methodSigFromMethod(method, m)
	if !ok {
		t.Fatal("expected ok=true")
	}
	// ObjC encoding for void method: "v@:"
	if !strings.HasPrefix(sig.ObjCEnc, "v") {
		t.Errorf("void method encoding should start with 'v'; got %q", sig.ObjCEnc)
	}
}

// ── goCallbackFuncType ────────────────────────────────────────────────────────

func TestGoCallbackFuncTypeVoid(t *testing.T) {
	sig := MethodSigModel{
		Name:      "void",
		IsVoidRet: true,
	}
	got := sig.goCallbackFuncType()
	if got != "func(unsafe.Pointer)" {
		t.Errorf("goCallbackFuncType(void) = %q; want \"func(unsafe.Pointer)\"", got)
	}
}

func TestGoCallbackFuncTypeNonVoid(t *testing.T) {
	sig := MethodSigModel{
		Name:         "int64",
		IsVoidRet:    false,
		GoReturnType: "int64",
		Params: []MethodSigArg{
			{GoType: "unsafe.Pointer"},
		},
	}
	got := sig.goCallbackFuncType()
	if !strings.Contains(got, "int64") {
		t.Errorf("goCallbackFuncType should contain return type; got %q", got)
	}
}

func TestGoCallbackFuncTypeBool(t *testing.T) {
	sig := MethodSigModel{
		Name:         "bool",
		IsVoidRet:    false,
		GoReturnType: "bool",
	}
	got := sig.goCallbackFuncType()
	if !strings.Contains(got, "bool") {
		t.Errorf("goCallbackFuncType should contain bool; got %q", got)
	}
}

// ── CollectMethodSigsFromFrameworks ──────────────────────────────────────────

func TestCollectMethodSigsEmpty(t *testing.T) {
	m := methodTrampolineMapper()
	frameworks := []*macosplatformmetadata.FrameworkMeta{{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}}
	sigs := CollectMethodSigsFromFrameworks(frameworks, m)
	if len(sigs) != 0 {
		t.Errorf("expected 0 sigs; got %d", len(sigs))
	}
}

func TestCollectMethodSigsFromClass(t *testing.T) {
	m := methodTrampolineMapper()
	frameworks := []*macosplatformmetadata.FrameworkMeta{
		{
			Framework: "Foundation",
			Classes: map[string]macosplatformmetadata.Class{
				"NSFoo": {Methods: []macosplatformmetadata.Method{
					{Selector: "doThing", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				}},
			},
		},
	}
	sigs := CollectMethodSigsFromFrameworks(frameworks, m)
	if len(sigs) == 0 {
		t.Error("expected at least one sig")
	}
}

func TestCollectMethodSigsDeduplicates(t *testing.T) {
	m := methodTrampolineMapper()
	// Two identical signatures across frameworks → deduplicated.
	fm1 := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSFoo": {Methods: []macosplatformmetadata.Method{
				{Selector: "doThing", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			}},
		},
	}
	fm2 := &macosplatformmetadata.FrameworkMeta{
		Framework: "AppKit",
		Classes: map[string]macosplatformmetadata.Class{
			"NSBar": {Methods: []macosplatformmetadata.Method{
				{Selector: "doOther", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			}},
		},
	}
	sigs := CollectMethodSigsFromFrameworks([]*macosplatformmetadata.FrameworkMeta{fm1, fm2}, m)
	seen := map[string]bool{}
	for _, s := range sigs {
		if seen[s.Name] {
			t.Errorf("duplicate sig name %q", s.Name)
		}
		seen[s.Name] = true
	}
}

func TestCollectMethodSigsFromProtocol(t *testing.T) {
	m := methodTrampolineMapper()
	frameworks := []*macosplatformmetadata.FrameworkMeta{
		{
			Framework: "Foundation",
			Classes:   map[string]macosplatformmetadata.Class{},
			Protocols: map[string]macosplatformmetadata.Protocol{
				"NSFooDelegate": {Methods: []macosplatformmetadata.Method{
					{Selector: "handleEvent", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				}},
			},
		},
	}
	sigs := CollectMethodSigsFromFrameworks(frameworks, m)
	if len(sigs) == 0 {
		t.Error("expected sig from protocol method")
	}
}

// ── EmitRuntimeCallbacksGo ────────────────────────────────────────────────────

func TestEmitRuntimeCallbacksGoEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := EmitRuntimeCallbacksGo(&buf, nil, "callbacks"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "package callbacks") {
		t.Errorf("expected package declaration; got:\n%s", out)
	}
}

func TestEmitRuntimeCallbacksGoVoidSig(t *testing.T) {
	sigs := []MethodSigModel{
		{Name: "void", IsVoidRet: true, CReturnType: "void"},
	}
	var buf bytes.Buffer
	if err := EmitRuntimeCallbacksGo(&buf, sigs, "callbacks"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "goCallIMP_void") {
		t.Errorf("expected goCallIMP_void; got:\n%s", out)
	}
	if !strings.Contains(out, "//export goCallIMP_void") {
		t.Errorf("expected //export directive; got:\n%s", out)
	}
}

func TestEmitRuntimeCallbacksGoNonVoidSig(t *testing.T) {
	sigs := []MethodSigModel{
		{Name: "int64", IsVoidRet: false, CReturnType: "int64_t", CGoReturnType: "C.int64_t", GoReturnType: "int64"},
	}
	var buf bytes.Buffer
	if err := EmitRuntimeCallbacksGo(&buf, sigs, "callbacks"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "goCallIMP_int64") {
		t.Errorf("expected goCallIMP_int64; got:\n%s", out)
	}
	if !strings.Contains(out, "C.int64_t") {
		t.Errorf("expected C.int64_t return type; got:\n%s", out)
	}
}

func TestEmitRuntimeCallbacksGoBoolReturn(t *testing.T) {
	sigs := []MethodSigModel{
		{Name: "bool", IsVoidRet: false, CReturnType: "bool", CGoReturnType: "C.bool", GoReturnType: "bool"},
	}
	var buf bytes.Buffer
	if err := EmitRuntimeCallbacksGo(&buf, sigs, "callbacks"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "C.bool(") {
		t.Errorf("expected C.bool cast for bool return; got:\n%s", out)
	}
}

func TestEmitRuntimeCallbacksGoWithArgs(t *testing.T) {
	sigs := []MethodSigModel{
		{
			Name:        "void_ptr",
			IsVoidRet:   true,
			CReturnType: "void",
			Params: []MethodSigArg{
				{CType: "void *", CGOType: "unsafe.Pointer", GoType: "unsafe.Pointer"},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitRuntimeCallbacksGo(&buf, sigs, "callbacks"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "arg0") {
		t.Errorf("expected arg0 parameter; got:\n%s", out)
	}
}

// ── EmitRuntimeCallbacksTrampolineHeader ─────────────────────────────────────

func TestEmitRuntimeCallbacksTrampolineHeaderEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := EmitRuntimeCallbacksTrampolineHeader(&buf, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "#pragma once") {
		t.Errorf("expected #pragma once; got:\n%s", out)
	}
}

func TestEmitRuntimeCallbacksTrampolineHeaderWithSig(t *testing.T) {
	sigs := []MethodSigModel{
		{Name: "void", IsVoidRet: true, CReturnType: "void"},
	}
	var buf bytes.Buffer
	if err := EmitRuntimeCallbacksTrampolineHeader(&buf, sigs); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "goIMP_void") {
		t.Errorf("expected goIMP_void declaration; got:\n%s", out)
	}
	if !strings.Contains(out, "goBridge_Trampoline_MethodFn_void") {
		t.Errorf("expected goBridge_Trampoline_MethodFn_void declaration; got:\n%s", out)
	}
}

// ── EmitRuntimeCallbacksTrampolineImpl ────────────────────────────────────────

func TestEmitRuntimeCallbacksTrampolineImplEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := EmitRuntimeCallbacksTrampolineImpl(&buf, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Code generated") {
		t.Errorf("expected generated header; got:\n%s", out)
	}
}

func TestEmitRuntimeCallbacksTrampolineImplVoidSig(t *testing.T) {
	sigs := []MethodSigModel{
		{Name: "void", IsVoidRet: true, CReturnType: "void"},
	}
	var buf bytes.Buffer
	if err := EmitRuntimeCallbacksTrampolineImpl(&buf, sigs); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "goIMP_void") {
		t.Errorf("expected goIMP_void; got:\n%s", out)
	}
	if !strings.Contains(out, "goBridge_Trampoline_MethodFn_void") {
		t.Errorf("expected goBridge_Trampoline_MethodFn_void; got:\n%s", out)
	}
}

func TestEmitRuntimeCallbacksTrampolineImplNonVoidSig(t *testing.T) {
	sigs := []MethodSigModel{
		{Name: "ptr", IsVoidRet: false, CReturnType: "void *"},
	}
	var buf bytes.Buffer
	if err := EmitRuntimeCallbacksTrampolineImpl(&buf, sigs); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// non-void: return nil for void* zero value
	if !strings.Contains(out, "nil") {
		t.Errorf("expected nil zero value for void* return; got:\n%s", out)
	}
}

func TestEmitRuntimeCallbacksTrampolineImplBoolSig(t *testing.T) {
	sigs := []MethodSigModel{
		{Name: "bool", IsVoidRet: false, CReturnType: "bool"},
	}
	var buf bytes.Buffer
	if err := EmitRuntimeCallbacksTrampolineImpl(&buf, sigs); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "false") {
		t.Errorf("expected false zero value for bool return; got:\n%s", out)
	}
}

// ── impCSignature ─────────────────────────────────────────────────────────────

func TestImpCSignatureVoid(t *testing.T) {
	sig := MethodSigModel{Name: "void", IsVoidRet: true, CReturnType: "void"}
	got := impCSignature(sig)
	if !strings.Contains(got, "goIMP_void") {
		t.Errorf("impCSignature: got %q; want to contain 'goIMP_void'", got)
	}
	if !strings.Contains(got, "id self") {
		t.Errorf("impCSignature: missing 'id self'; got %q", got)
	}
}

func TestImpCSignatureWithArgs(t *testing.T) {
	sig := MethodSigModel{
		Name:        "void_ptr",
		CReturnType: "void",
		Params:      []MethodSigArg{{CType: "void *"}},
	}
	got := impCSignature(sig)
	if !strings.Contains(got, "void *") {
		t.Errorf("impCSignature: expected void * arg; got %q", got)
	}
}

// ── goCallIMPCDecl ─────────────────────────────────────────────────────────────

func TestGoCallIMPCDeclVoid(t *testing.T) {
	sig := MethodSigModel{Name: "void", IsVoidRet: true, CReturnType: "void"}
	got := goCallIMPCDecl(sig)
	if !strings.Contains(got, "goCallIMP_void") {
		t.Errorf("goCallIMPCDecl: got %q; want to contain 'goCallIMP_void'", got)
	}
	if !strings.HasPrefix(got, "void ") {
		t.Errorf("goCallIMPCDecl: should start with 'void '; got %q", got)
	}
}

func TestGoCallIMPCDeclNonVoid(t *testing.T) {
	sig := MethodSigModel{Name: "int64", IsVoidRet: false, CReturnType: "int64_t"}
	got := goCallIMPCDecl(sig)
	if !strings.HasPrefix(got, "int64_t ") {
		t.Errorf("goCallIMPCDecl: should start with 'int64_t '; got %q", got)
	}
}
