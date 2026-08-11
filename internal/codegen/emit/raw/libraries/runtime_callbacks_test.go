package rawlib

import (
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

// ── EmitRuntimeCallbacksGo ────────────────────────────────────────────────────

// ── EmitRuntimeCallbacksTrampolineHeader ─────────────────────────────────────

// ── EmitRuntimeCallbacksTrampolineImpl ────────────────────────────────────────

// ── impCSignature ─────────────────────────────────────────────────────────────

// ── goCallIMPCDecl ─────────────────────────────────────────────────────────────
