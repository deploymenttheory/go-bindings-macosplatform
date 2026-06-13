package raw

import (
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// Externs writes a complete _externs.go file for the framework's extern symbols.
func EmitExterns(w io.Writer, pkgName string, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) error {
	if len(framework.Externs) == 0 {
		return nil
	}
	return executeTemplate(w, "externs_file", buildExternsModel(pkgName, framework, m, knownClasses))
}

// buildExternsModel resolves types and collects imports, then returns a model
// ready for template execution. All sorting, deduplication, and import decisions
// are made here; the template itself is a pure structural description.
func buildExternsModel(pkgName string, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) externsFileModel {
	usedImports := make(typemap.ImportSet)
	ctx := m.BaseContext(framework.Framework, knownClasses)

	// Sort for deterministic output, then deduplicate by Go name.
	externs := make([]meta.Extern, len(framework.Externs))
	copy(externs, framework.Externs)
	sort.Slice(externs, func(i, j int) bool { return externs[i].Name < externs[j].Name })

	seen := make(map[string]bool)
	items := make([]externItemModel, 0, len(externs))
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
		items = append(items, externItemModel{
			GoName:       goName,
			GoType:       goType,
			CommentBlock: renderCommentBlock(e.Doc, e.SDKFile, e.SDKLine, e.Availability, "\t"),
		})
	}

	imports := buildExternsImports(items, usedImports)
	return externsFileModel{PkgName: pkgName, Imports: imports, Items: items}
}

// buildExternsImports collects and deduplicates the imports needed by an externs file.
func buildExternsImports(items []externItemModel, usedImports typemap.ImportSet) []string {
	set := make(map[string]bool)
	for _, it := range items {
		if strings.Contains(it.GoType, "unsafe.Pointer") {
			set["unsafe"] = true
		}
		if strings.Contains(it.GoType, "objc.") {
			set["github.com/deploymenttheory/go-bindings-macosplatform/internal/objc"] = true
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
