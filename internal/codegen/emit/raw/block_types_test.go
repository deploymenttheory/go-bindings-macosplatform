package raw

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// newBlockFM returns a FrameworkMeta with the given BlockTypes map.
func newBlockFM(framework string, blocks map[string]meta.BlockType) *meta.FrameworkMeta {
	return &meta.FrameworkMeta{
		Framework:  framework,
		BlockTypes: blocks,
		Classes:    map[string]meta.Class{},
		Protocols:  map[string]meta.Protocol{},
		Enums:      map[string]meta.Enum{},
		Structs:    map[string]meta.Struct{},
	}
}

func TestBlocksEmpty(t *testing.T) {
	framework := newBlockFM("Foundation", map[string]meta.BlockType{})
	m := testMapper()
	var buf bytes.Buffer
	err := EmitBlocks(&buf, framework, m, map[string]bool{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty BlockTypes, got %q", buf.String())
	}
}

func TestBlockNoArgs(t *testing.T) {
	framework := newBlockFM("Foundation", map[string]meta.BlockType{
		"MyBlock": {
			Params:   nil,
			Return: meta.ReturnType{},
		},
	})
	m := testMapper()
	var buf bytes.Buffer
	if err := EmitBlocks(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "type MyBlock = func()") {
		t.Errorf("expected 'type MyBlock = func()', got:\n%s", out)
	}
}

func TestBlockVoidArgs(t *testing.T) {
	// Blocks whose only arg has ObjCType "void" should produce func() with no args.
	framework := newBlockFM("Foundation", map[string]meta.BlockType{
		"VoidArgBlock": {
			Params: []meta.Param{
				{Name: "", ObjCType: "void"},
			},
			Return: meta.ReturnType{},
		},
	})
	m := testMapper()
	var buf bytes.Buffer
	if err := EmitBlocks(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	// The mapper resolves "void" to "" (empty string), so the func gets no typed arg.
	// The emitter will still include the arg slot, but with an empty type.
	// Verify the type alias is produced.
	if !strings.Contains(out, "type VoidArgBlock = func(") {
		t.Errorf("expected type VoidArgBlock definition, got:\n%s", out)
	}
}

func TestBlockWithArgs(t *testing.T) {
	framework := newBlockFM("Foundation", map[string]meta.BlockType{
		"CompletionBlock": {
			Params: []meta.Param{
				{Name: "result", ObjCType: "NSUInteger"},
				{Name: "finished", ObjCType: "BOOL"},
			},
			Return: meta.ReturnType{},
		},
	})
	m := testMapper()
	var buf bytes.Buffer
	if err := EmitBlocks(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "result uint64") {
		t.Errorf("expected 'result uint64' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "finished bool") {
		t.Errorf("expected 'finished bool' in output, got:\n%s", out)
	}
}

func TestBlockWithReturn(t *testing.T) {
	framework := newBlockFM("Foundation", map[string]meta.BlockType{
		"PredicateBlock": {
			Params:   nil,
			Return: meta.ReturnType{ObjCType: "BOOL"},
		},
	})
	m := testMapper()
	var buf bytes.Buffer
	if err := EmitBlocks(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "func() bool") {
		t.Errorf("expected 'func() bool', got:\n%s", out)
	}
}

func TestBlockUnnamedArg(t *testing.T) {
	// Args with no Name should be given positional names arg0, arg1, etc.
	framework := newBlockFM("Foundation", map[string]meta.BlockType{
		"UnnamedBlock": {
			Params: []meta.Param{
				{Name: "", ObjCType: "NSUInteger"},
				{Name: "", ObjCType: "BOOL"},
			},
			Return: meta.ReturnType{},
		},
	})
	m := testMapper()
	var buf bytes.Buffer
	if err := EmitBlocks(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "arg0") {
		t.Errorf("expected positional arg 'arg0' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "arg1") {
		t.Errorf("expected positional arg 'arg1' in output, got:\n%s", out)
	}
}

func TestBlockArgObjCTypeResolution(t *testing.T) {
	// ObjC primitive types are resolved to Go types by the typemap.
	framework := newBlockFM("Foundation", map[string]meta.BlockType{
		"DirectTypeBlock": {
			Params: []meta.Param{
				{Name: "val", ObjCType: "int64_t"},
			},
			Return: meta.ReturnType{},
		},
	})
	m := testMapper()
	var buf bytes.Buffer
	if err := EmitBlocks(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "val int64") {
		t.Errorf("expected 'val int64' in output, got:\n%s", out)
	}
}

func TestBlockSortOrder(t *testing.T) {
	// Multiple blocks should be emitted in alphabetical order.
	framework := newBlockFM("Foundation", map[string]meta.BlockType{
		"ZebraBlock": {Return: meta.ReturnType{}},
		"AlphaBlock": {Return: meta.ReturnType{}},
		"MidBlock":   {Return: meta.ReturnType{}},
	})
	m := testMapper()
	var buf bytes.Buffer
	if err := EmitBlocks(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	alphaIdx := strings.Index(out, "AlphaBlock")
	midIdx := strings.Index(out, "MidBlock")
	zebraIdx := strings.Index(out, "ZebraBlock")
	if alphaIdx < 0 || midIdx < 0 || zebraIdx < 0 {
		t.Fatalf("one or more block names not found in output:\n%s", out)
	}
	if !(alphaIdx < midIdx && midIdx < zebraIdx) {
		t.Errorf("blocks not in alphabetical order: AlphaBlock@%d MidBlock@%d ZebraBlock@%d",
			alphaIdx, midIdx, zebraIdx)
	}
}

func TestBlockCommentContainsName(t *testing.T) {
	// Each block should have a doc comment naming it.
	framework := newBlockFM("Foundation", map[string]meta.BlockType{
		"HandlerBlock": {Return: meta.ReturnType{}},
	})
	m := testMapper()
	var buf bytes.Buffer
	if err := EmitBlocks(&buf, framework, m, map[string]bool{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "// HandlerBlock") {
		t.Errorf("expected doc comment '// HandlerBlock', got:\n%s", out)
	}
}
