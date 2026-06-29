package rawlib

import (
	"fmt"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/view"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// EmitBlocks writes named Go func types for each unique ObjC block signature
// collected by the scanner.
func EmitBlocks(w io.Writer, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) error {
	if len(framework.BlockTypes) == 0 {
		return nil
	}

	ctx := typemap.Context{
		Framework:      framework.Framework,
		ClassNameIndex: knownClasses,
	}

	names := make([]string, 0, len(framework.BlockTypes))
	for k := range framework.BlockTypes {
		names = append(names, k)
	}
	sort.Strings(names)

	for _, name := range names {
		bt := framework.BlockTypes[name]
		model, err := buildBlockTypeModel(name, bt, ctx, m)
		if err != nil {
			return err
		}
		if err := render.Execute(w, "block_type_item", model); err != nil {
			return err
		}
	}
	return nil
}

func buildBlockTypeModel(name string, bt macosplatformmetadata.BlockType, ctx typemap.Context, m *typemap.Mapper) (view.BlockTypeModel, error) {
	imports := make(typemap.ImportSet)
	var goArgs []string
	for i, arg := range bt.Params {
		goType := m.GoType(arg.ObjCType, ctx, imports)
		if goType == "" {
			goType = "unsafe.Pointer"
		}
		argName := arg.Name
		if argName == "" {
			argName = fmt.Sprintf("arg%d", i)
		}
		goArgs = append(goArgs, fmt.Sprintf("%s %s", argName, goType))
	}

	var retType string
	if bt.Return.ObjCType != "" {
		retType = m.GoReturnType(bt.Return.ObjCType, ctx, imports)
	}

	var sb strings.Builder
	sb.WriteString("func(")
	sb.WriteString(strings.Join(goArgs, ", "))
	sb.WriteString(")")
	if retType != "" {
		sb.WriteString(" ")
		sb.WriteString(retType)
	}

	return view.BlockTypeModel{
		GoName:    name,
		Framework: ctx.Framework,
		Sig:       sb.String(),
	}, nil
}
