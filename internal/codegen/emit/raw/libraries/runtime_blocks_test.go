package rawlib

import (
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
)

func blockTestMapper() *typemap.Mapper {
	return &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex:     map[string]string{},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
		TypedefIndex:   map[string]string{},
		StructIndex:    map[string]string{},
	}
}

// ── cTypeToken ────────────────────────────────────────────────────────────────

func TestCTypeTokenKnownTypes(t *testing.T) {
	cases := []struct{ in, want string }{
		{"void", "void"},
		{"", "void"},
		{"void *", "ptr"},
		{"bool", "bool"},
		{"int8_t", "int8"},
		{"int16_t", "int16"},
		{"int32_t", "int32"},
		{"int64_t", "int64"},
		{"uint8_t", "uint8"},
		{"uint16_t", "uint16"},
		{"uint32_t", "uint32"},
		{"uint64_t", "uint64"},
		{"float", "float32"},
		{"double", "float64"},
		{"id", "ptr"}, // unknown → "ptr"
	}
	for _, c := range cases {
		if got := cTypeToken(c.in); got != c.want {
			t.Errorf("cTypeToken(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

// ── BlockSigName ──────────────────────────────────────────────────────────────

func TestBlockSigNameVoidNoArgs(t *testing.T) {
	m := blockTestMapper()
	got := BlockSigName("void (^)(void)", m)
	if got != "void" {
		t.Errorf("BlockSigName(void (^)(void)) = %q; want %q", got, "void")
	}
}

func TestBlockSigNameVoidPtrArg(t *testing.T) {
	m := blockTestMapper()
	// void (^)(NSError *) → void_ptr (object args → ptr)
	got := BlockSigName("void (^)(NSError *)", m)
	if got != "void_ptr" {
		t.Errorf("BlockSigName = %q; want \"void_ptr\"", got)
	}
}

func TestBlockSigNameBoolReturn(t *testing.T) {
	m := blockTestMapper()
	got := BlockSigName("BOOL (^)(void)", m)
	// BOOL maps to bool in CType
	if got == "" {
		t.Errorf("BlockSigName(BOOL (^)(void)) returned empty")
	}
}

func TestBlockSigNameInvalidReturnsEmpty(t *testing.T) {
	m := blockTestMapper()
	got := BlockSigName("not a block type", m)
	if got != "" {
		t.Errorf("invalid block type should return empty; got %q", got)
	}
}

func TestBlockSigNameMultiArgs(t *testing.T) {
	m := blockTestMapper()
	// void (^)(id, BOOL) → void_ptr_bool (object→ptr, BOOL→bool)
	got := BlockSigName("void (^)(id, BOOL)", m)
	if !strings.Contains(got, "void") {
		t.Errorf("BlockSigName multi-arg: got %q, want prefix 'void'", got)
	}
}

// ── BlockSigFromObjC ──────────────────────────────────────────────────────────

func TestBlockSigFromObjCVoidNoArgs(t *testing.T) {
	m := blockTestMapper()
	sig, ok := BlockSigFromObjC("void (^)(void)", m)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !sig.IsVoidRet {
		t.Error("expected IsVoidRet=true")
	}
	if len(sig.Args) != 0 {
		t.Errorf("expected 0 args; got %d", len(sig.Args))
	}
}

func TestBlockSigFromObjCInt64Ret(t *testing.T) {
	m := blockTestMapper()
	// int64_t return
	sig, ok := BlockSigFromObjC("int64_t (^)(void)", m)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if sig.IsVoidRet {
		t.Error("expected IsVoidRet=false")
	}
	if sig.RetGo != "int64" {
		t.Errorf("RetGo = %q; want \"int64\"", sig.RetGo)
	}
}

func TestBlockSigFromObjCPtrReturn(t *testing.T) {
	m := blockTestMapper()
	sig, ok := BlockSigFromObjC("id (^)(void)", m)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if sig.RetGo != "unsafe.Pointer" {
		t.Errorf("RetGo = %q; want \"unsafe.Pointer\"", sig.RetGo)
	}
}

func TestBlockSigFromObjCWithArgs(t *testing.T) {
	m := blockTestMapper()
	sig, ok := BlockSigFromObjC("void (^)(id, BOOL)", m)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if len(sig.Args) == 0 {
		t.Error("expected at least one arg")
	}
}

func TestBlockSigFromObjCInvalid(t *testing.T) {
	m := blockTestMapper()
	_, ok := BlockSigFromObjC("not-a-block", m)
	if ok {
		t.Error("expected ok=false for invalid block type")
	}
}

// ── CollectBlockSignaturesFromFrameworks ──────────────────────────────────────

// ── EmitRuntimeBlocksGo ───────────────────────────────────────────────────────

// ── EmitRuntimeBlocksTrampolineHeader ────────────────────────────────────────

// ── EmitRuntimeBlocksTrampolineImpl ──────────────────────────────────────────

// ── emitGoCallBlockExport — non-void return and GoType!=CGOType paths ─────────

// ── CollectBlockSignaturesFromFrameworks — ForeignExtensions path ─────────────
