package rawlib

import (
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/view"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// externInitExpr returns the Go expression that populates an extern var from
// its bridge address getter, or "" when the extern's shape is unsupported
// (strings, blocks, function-pointer typedefs) and the var stays zero-valued.
//
// Three shapes are supported:
//   - struct-valued globals (e.g. dispatch source types): the useful Go value
//     is the global's address itself;
//   - pointer-typed globals (e.g. xpc constant objects): dereference to load
//     the stored pointer;
//   - integral globals (e.g. mach_task_self_): dereference at the Go type.
func externInitExpr(pkgName string, e macosplatformmetadata.Extern, goType string) string {
	getter := "C." + externGetterName(pkgName, e.Name) + "()"
	t := strings.TrimSpace(e.ObjCType)
	t = strings.TrimPrefix(t, "const ")
	if strings.Contains(t, "(") { // function-pointer / block typedefs
		return ""
	}
	isStructValue := strings.HasPrefix(t, "struct ") && !strings.Contains(t, "*")
	switch {
	case isStructValue && goType == "unsafe.Pointer":
		return getter
	case isStructValue:
		return ""
	case goType == "unsafe.Pointer":
		return "*(*unsafe.Pointer)(" + getter + ")"
	case goType == "uint32", goType == "int32", goType == "uint64", goType == "int64",
		goType == "uint", goType == "int", goType == "uintptr":
		return "*(*" + goType + ")(" + getter + ")"
	default:
		return ""
	}
}

// externGetterName returns the bridge C function that exposes the address of
// the extern global (e.g. "machinit_extern_mach_task_self_").
func externGetterName(pkgName, symbol string) string {
	return pkgName + "_extern_" + symbol
}

// buildExternGetters returns the bridge getter declarations for every extern
// this package initialises. It mirrors buildExternsModel's dedup and shape
// rules so the bridge and the Go init() always agree.
// buildExternsModel resolves types and collects imports, then returns a model
// ready for template execution. All sorting, deduplication, and import decisions
// are made here; the template itself is a pure structural description.
func buildExternsModel(pkgName string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) view.ExternsFileModel {
	usedImports := make(typemap.ImportSet)
	ctx := m.BaseContext(framework.Framework, knownClasses)

	// Sort for deterministic output, then deduplicate by Go name.
	externs := make([]macosplatformmetadata.Extern, len(framework.Externs))
	copy(externs, framework.Externs)
	sort.Slice(externs, func(i, j int) bool { return externs[i].Name < externs[j].Name })

	seen := make(map[string]bool)
	items := make([]view.ExternItemModel, 0, len(externs))
	for _, e := range externs {
		goName := naming.GoTypeName(e.Name)
		if seen[goName] {
			continue
		}
		seen[goName] = true

		goType := e.GoType
		if goType == "" {
			goType = m.GoType(e.ObjCType, ctx, usedImports)
		}
		if goType == "" {
			goType = "unsafe.Pointer"
		}
		items = append(items, view.ExternItemModel{
			GoName:       goName,
			GoType:       goType,
			CommentBlock: renderCommentBlock(e.Doc, e.SDKFile, e.SDKLine, e.Availability, "\t"),
			InitExpr:     externInitExpr(pkgName, e, goType),
			SymbolName:   e.Name,
		})
	}

	imports := buildExternsImports(items, usedImports)
	model := view.ExternsFileModel{PkgName: pkgName, Imports: imports, Items: items}
	for _, it := range items {
		if it.InitExpr != "" {
			model.HasInit = true
			model.BridgeInclude = "bridge/" + pkgName + "_bridge.h"
			break
		}
	}
	return model
}

// buildExternsImports collects and deduplicates the imports needed by an externs file.
func buildExternsImports(items []view.ExternItemModel, usedImports typemap.ImportSet) []string {
	set := make(map[string]bool)
	for _, it := range items {
		if strings.Contains(it.GoType, "unsafe.Pointer") {
			set["unsafe"] = true
		}
	}
	for _, path := range usedImports {
		set[path] = true
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
