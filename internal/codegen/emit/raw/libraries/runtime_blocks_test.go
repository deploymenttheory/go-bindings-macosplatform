package raw

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

func blockTestMapper() *typemap.Mapper {
	return &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex: map[string]string{},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
		TypedefIndex:  map[string]string{},
		StructIndex:   map[string]string{},
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

func TestCollectBlockSignaturesEmpty(t *testing.T) {
	m := blockTestMapper()
	frameworks := []*macosplatformmetadata.FrameworkMeta{{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}}
	sigs := CollectBlockSignaturesFromFrameworks(frameworks, m)
	if len(sigs) != 0 {
		t.Errorf("expected 0 sigs; got %d", len(sigs))
	}
}

func TestCollectBlockSignaturesFromClass(t *testing.T) {
	m := blockTestMapper()
	frameworks := []*macosplatformmetadata.FrameworkMeta{
		{
			Framework: "Foundation",
			Classes: map[string]macosplatformmetadata.Class{
				"NSFoo": {
					Methods: []macosplatformmetadata.Method{
						{
							Selector: "doWith:",
							Params:     []macosplatformmetadata.Param{{Name: "handler", ObjCType: "void (^)(NSError *)", IsBlock: true}},
							Return:   macosplatformmetadata.ReturnType{ObjCType: "void"},
						},
					},
				},
			},
		},
	}
	sigs := CollectBlockSignaturesFromFrameworks(frameworks, m)
	if len(sigs) == 0 {
		t.Error("expected at least one sig from block arg")
	}
}

func TestCollectBlockSignaturesFromProtocol(t *testing.T) {
	m := blockTestMapper()
	frameworks := []*macosplatformmetadata.FrameworkMeta{
		{
			Framework: "Foundation",
			Classes:   map[string]macosplatformmetadata.Class{},
			Protocols: map[string]macosplatformmetadata.Protocol{
				"NSFooDelegate": {
					Methods: []macosplatformmetadata.Method{
						{
							Selector: "foo:handler:",
							Params:     []macosplatformmetadata.Param{{Name: "handler", ObjCType: "void (^)(void)", IsBlock: true}},
							Return:   macosplatformmetadata.ReturnType{ObjCType: "void"},
						},
					},
				},
			},
		},
	}
	sigs := CollectBlockSignaturesFromFrameworks(frameworks, m)
	if len(sigs) == 0 {
		t.Error("expected at least one sig from protocol block arg")
	}
}

func TestCollectBlockSignaturesDeduplicates(t *testing.T) {
	m := blockTestMapper()
	// Two frameworks with identical block types should deduplicate.
	fm1 := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSFoo": {Methods: []macosplatformmetadata.Method{
				{Selector: "a:", Params: []macosplatformmetadata.Param{{Name: "h", ObjCType: "void (^)(NSError *)", IsBlock: true}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			}},
		},
	}
	fm2 := &macosplatformmetadata.FrameworkMeta{
		Framework: "AppKit",
		Classes: map[string]macosplatformmetadata.Class{
			"NSBar": {Methods: []macosplatformmetadata.Method{
				{Selector: "b:", Params: []macosplatformmetadata.Param{{Name: "h", ObjCType: "void (^)(NSError *)", IsBlock: true}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
			}},
		},
	}
	sigs := CollectBlockSignaturesFromFrameworks([]*macosplatformmetadata.FrameworkMeta{fm1, fm2}, m)
	// Should be 1, not 2.
	for i, s := range sigs {
		for j, s2 := range sigs {
			if i != j && s.Name == s2.Name {
				t.Errorf("duplicate sig %q", s.Name)
			}
		}
	}
}

func TestCollectBlockSignaturesFromTypedef(t *testing.T) {
	m := blockTestMapper()
	frameworks := []*macosplatformmetadata.FrameworkMeta{
		{
			Framework: "Foundation",
			Typedefs:  map[string]string{"myHandler_t": "void (^)(void)"},
			Classes: map[string]macosplatformmetadata.Class{
				"NSFoo": {Methods: []macosplatformmetadata.Method{
					{
						Selector: "doThing:",
						Params:     []macosplatformmetadata.Param{{Name: "handler", ObjCType: "myHandler_t"}},
						Return:   macosplatformmetadata.ReturnType{ObjCType: "void"},
					},
				}},
			},
		},
	}
	sigs := CollectBlockSignaturesFromFrameworks(frameworks, m)
	if len(sigs) == 0 {
		t.Error("expected sig from typedef block arg")
	}
}

// ── EmitRuntimeBlocksGo ───────────────────────────────────────────────────────

func TestEmitRuntimeBlocksGoEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := EmitRuntimeBlocksGo(&buf, nil, "blocks"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "package blocks") {
		t.Errorf("expected package declaration; got:\n%s", out)
	}
}

func TestEmitRuntimeBlocksGoWithSig(t *testing.T) {
	m := blockTestMapper()
	sigs := CollectBlockSignaturesFromFrameworks([]*macosplatformmetadata.FrameworkMeta{
		{
			Framework: "Foundation",
			Classes: map[string]macosplatformmetadata.Class{
				"NSFoo": {Methods: []macosplatformmetadata.Method{
					{Selector: "do:", Params: []macosplatformmetadata.Param{{Name: "h", ObjCType: "void (^)(NSError *)", IsBlock: true}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				}},
			},
		},
	}, m)

	var buf bytes.Buffer
	if err := EmitRuntimeBlocksGo(&buf, sigs, "blocks"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "goCallBlock_") {
		t.Errorf("expected goCallBlock_ in output; got:\n%s", out)
	}
	if !strings.Contains(out, "MakeBlock_") {
		t.Errorf("expected MakeBlock_ in output; got:\n%s", out)
	}
}

// ── EmitRuntimeBlocksTrampolineHeader ────────────────────────────────────────

func TestEmitRuntimeBlocksTrampolineHeaderEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := EmitRuntimeBlocksTrampolineHeader(&buf, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "#pragma once") {
		t.Errorf("expected #pragma once; got:\n%s", out)
	}
}

func TestEmitRuntimeBlocksTrampolineHeaderWithSig(t *testing.T) {
	m := blockTestMapper()
	sigs := CollectBlockSignaturesFromFrameworks([]*macosplatformmetadata.FrameworkMeta{
		{
			Framework: "Foundation",
			Classes: map[string]macosplatformmetadata.Class{
				"NSFoo": {Methods: []macosplatformmetadata.Method{
					{Selector: "do:", Params: []macosplatformmetadata.Param{{Name: "h", ObjCType: "void (^)(NSError *)", IsBlock: true}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				}},
			},
		},
	}, m)

	var buf bytes.Buffer
	if err := EmitRuntimeBlocksTrampolineHeader(&buf, sigs); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "goMakeBlock_") {
		t.Errorf("expected goMakeBlock_ declaration; got:\n%s", out)
	}
}

// ── EmitRuntimeBlocksTrampolineImpl ──────────────────────────────────────────

func TestEmitRuntimeBlocksTrampolineImplEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := EmitRuntimeBlocksTrampolineImpl(&buf, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Code generated") {
		t.Errorf("expected generated header; got:\n%s", out)
	}
}

func TestEmitRuntimeBlocksTrampolineImplWithSig(t *testing.T) {
	m := blockTestMapper()
	sigs := CollectBlockSignaturesFromFrameworks([]*macosplatformmetadata.FrameworkMeta{
		{
			Framework: "Foundation",
			Classes: map[string]macosplatformmetadata.Class{
				"NSFoo": {Methods: []macosplatformmetadata.Method{
					{Selector: "do:", Params: []macosplatformmetadata.Param{{Name: "h", ObjCType: "void (^)(NSError *)", IsBlock: true}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				}},
			},
		},
	}, m)

	var buf bytes.Buffer
	if err := EmitRuntimeBlocksTrampolineImpl(&buf, sigs); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "goCallBlock_") {
		t.Errorf("expected goCallBlock_ in impl; got:\n%s", out)
	}
}

func TestEmitRuntimeBlocksTrampolineImplNonVoidRet(t *testing.T) {
	m := blockTestMapper()
	// int64_t return block
	sigs := CollectBlockSignaturesFromFrameworks([]*macosplatformmetadata.FrameworkMeta{
		{
			Framework: "Foundation",
			Classes: map[string]macosplatformmetadata.Class{
				"NSFoo": {Methods: []macosplatformmetadata.Method{
					{Selector: "count:", Params: []macosplatformmetadata.Param{{Name: "h", ObjCType: "int64_t (^)(void)", IsBlock: true}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				}},
			},
		},
	}, m)

	var buf bytes.Buffer
	if err := EmitRuntimeBlocksTrampolineImpl(&buf, sigs); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "int64_t") {
		t.Errorf("expected int64_t in non-void block impl; got:\n%s", out)
	}
}

// ── emitGoCallBlockExport — non-void return and GoType!=CGOType paths ─────────

// TestEmitRuntimeBlocksGoNonVoidReturn covers the non-void return path in emitGoCallBlockExport.
func TestEmitRuntimeBlocksGoNonVoidReturn(t *testing.T) {
	m := blockTestMapper()
	sigs := CollectBlockSignaturesFromFrameworks([]*macosplatformmetadata.FrameworkMeta{
		{
			Framework: "Foundation",
			Classes: map[string]macosplatformmetadata.Class{
				"NSFoo": {Methods: []macosplatformmetadata.Method{
					// int64_t return block → non-void return in emitGoCallBlockExport
					{Selector: "count:", Params: []macosplatformmetadata.Param{{Name: "h", ObjCType: "int64_t (^)(void)", IsBlock: true}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				}},
			},
		},
	}, m)

	var buf bytes.Buffer
	if err := EmitRuntimeBlocksGo(&buf, sigs, "blocks"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Non-void return block should have named return variable and result assignment.
	if !strings.Contains(out, "(result ") {
		t.Errorf("non-void return block should have named result return; got:\n%s", out)
	}
	if !strings.Contains(out, "result =") {
		t.Errorf("non-void return block should assign result; got:\n%s", out)
	}
}

// TestEmitRuntimeBlocksGoWithInt64Arg covers the GoType != CGOType path in emitGoCallBlockExport.
func TestEmitRuntimeBlocksGoWithInt64Arg(t *testing.T) {
	m := blockTestMapper()
	sigs := CollectBlockSignaturesFromFrameworks([]*macosplatformmetadata.FrameworkMeta{
		{
			Framework: "Foundation",
			Classes: map[string]macosplatformmetadata.Class{
				"NSFoo": {Methods: []macosplatformmetadata.Method{
					// int64_t arg: CGOType = "C.int64_t", GoType = "int64" → GoType != CGOType
					{Selector: "sort:", Params: []macosplatformmetadata.Param{{Name: "h", ObjCType: "void (^)(int64_t)", IsBlock: true}}, Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
				}},
			},
		},
	}, m)

	var buf bytes.Buffer
	if err := EmitRuntimeBlocksGo(&buf, sigs, "blocks"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// GoType != CGOType: arg should be cast to GoType
	if !strings.Contains(out, "int64(arg") {
		t.Errorf("int64 arg should be cast from C.int64_t to int64; got:\n%s", out)
	}
}

// ── CollectBlockSignaturesFromFrameworks — ForeignExtensions path ─────────────

// TestCollectBlockSignaturesFromForeignExtensions covers the ForeignExtensions loop.
func TestCollectBlockSignaturesFromForeignExtensions(t *testing.T) {
	m := blockTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "MyFW",
		ForeignExtensions: map[string][]macosplatformmetadata.Method{
			"NSObject": {
				{
					Selector: "doWithBlock:",
					Params:     []macosplatformmetadata.Param{{Name: "h", ObjCType: "void (^)(NSError *)", IsBlock: true}},
					Return:   macosplatformmetadata.ReturnType{ObjCType: "void"},
				},
			},
		},
	}
	sigs := CollectBlockSignaturesFromFrameworks([]*macosplatformmetadata.FrameworkMeta{framework}, m)
	if len(sigs) == 0 {
		t.Error("expected block signature from ForeignExtensions; got none")
	}
}
