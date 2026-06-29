package rawfw

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks/view"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// EmitExterns writes accessor functions for extern global symbols.
// Each extern is accessed via purego.Dlsym at runtime.
func EmitExterns(
	w io.Writer,
	framework *meta.FrameworkMeta,
	mapper *typemap.Mapper,
	dylibVarName string,
) (typemap.ImportSet, error) {
	imports := make(typemap.ImportSet)

	// Build set of function names to avoid declaring an extern accessor with
	// the same name as an existing exported function (causes redeclaration).
	// Both the raw C names and the exported Go names are reserved.
	funcNames := make(map[string]bool, len(framework.Functions)*2)
	for _, fn := range framework.Functions {
		funcNames[fn.Name] = true
		if goName := naming.ExportedFunctionName(fn.Name); goName != "" {
			funcNames[goName] = true
		}
	}

	seen := make(map[string]bool)
	seenGoNames := make(map[string]bool)
	externs := make([]meta.Extern, 0, len(framework.Externs))
	for _, ext := range framework.Externs {
		if ext.Availability.IsUnavailable || seen[ext.Name] || funcNames[ext.Name] {
			continue
		}
		goName := exportedExternName(ext.Name)
		if goName == "" || seenGoNames[goName] || funcNames[goName] {
			continue
		}
		seen[ext.Name] = true
		seenGoNames[goName] = true
		externs = append(externs, ext)
	}
	if len(externs) == 0 {
		return imports, nil
	}

	sort.Slice(externs, func(i, j int) bool { return externs[i].Name < externs[j].Name })

	ctx := typemap.Context{Framework: framework.Framework}

	views := make([]view.Extern, 0, len(externs))
	for _, ext := range externs {
		goType := ext.GoType
		if goType == "" {
			goType = mapper.GoType(ext.ObjCType, ctx, imports)
		}
		if goType == "" || goType == "unsafe.Pointer" || isUnexportedXPkg(goType) {
			views = append(views, buildExternRawView(ext, dylibVarName))
			continue
		}

		fromIDCall, isClassPtr := mapper.ObjCClassFromID(goType)
		if !isClassPtr {
			fromIDCall = ""
		}
		views = append(views, buildExternTypedView(ext, goType, dylibVarName, mapper.IsEnumType(goType), fromIDCall))
	}

	out, err := render.Externs(views)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(out); err != nil {
		return nil, err
	}
	return imports, nil
}

// externCommentBlock renders an extern's doc + deprecation comment (column 0).
func externCommentBlock(ext meta.Extern) string {
	var sb strings.Builder
	if ext.Doc != "" {
		fmt.Fprintf(&sb, "// %s\n", ext.Doc)
	}
	sb.WriteString(deprecatedComment(ext.Availability))
	return sb.String()
}

// buildExternRawView resolves an extern with no usable typed mapping into a raw
// uintptr accessor view.
func buildExternRawView(ext meta.Extern, dylibVarName string) view.Extern {
	return view.Extern{
		CommentBlock: externCommentBlock(ext),
		GoName:       exportedExternName(ext.Name),
		RetType:      "uintptr",
		DylibVar:     dylibVarName,
		Symbol:       ext.Name,
		Form:         "raw",
	}
}

// exportedExternName maps an extern symbol to an exported Go accessor name.
// C symbols may begin with a lowercase letter (kSecClass) or underscores
// (_NSGetEnviron); a verbatim name would be unexported and therefore
// uncallable from outside the generated package.
func exportedExternName(symbol string) string {
	trimmed := strings.TrimLeft(symbol, "_")
	if trimmed == "" {
		return ""
	}
	return strings.ToUpper(trimmed[:1]) + trimmed[1:]
}

// isUnexportedXPkg reports whether a Go type references an unexported identifier
// from another package (which would cause a compile error).
func isUnexportedXPkg(goType string) bool {
	t := strings.TrimPrefix(goType, "*")
	t = strings.TrimPrefix(t, "[]")
	dotIdx := strings.LastIndex(t, ".")
	if dotIdx < 0 {
		return false
	}
	after := t[dotIdx+1:]
	return len(after) > 0 && after[0] >= 'a' && after[0] <= 'z'
}

// buildExternTypedView resolves an extern with a usable Go type into a typed
// accessor view, selecting the body Form by the type's nature (ObjC object,
// char* string, or value type).
func buildExternTypedView(ext meta.Extern, goType, dylibVarName string, isEnum bool, fromIDCall string) view.Extern {
	built := view.Extern{
		CommentBlock: externCommentBlock(ext),
		GoName:       exportedExternName(ext.Name),
		RetType:      goType,
		DylibVar:     dylibVarName,
		Symbol:       ext.Name,
		GoType:       goType,
	}
	switch {
	case fromIDCall != "":
		built.Form = "fromid"
		built.FromIDCall = fromIDCall
	case goType == "string":
		built.Form = "string"
	default:
		zero := zeroValue(goType)
		if isEnum {
			zero = "0" // enum types are integers — TypeName{} is invalid
		}
		built.Form = "value"
		built.Zero = zero
	}
	return built
}
