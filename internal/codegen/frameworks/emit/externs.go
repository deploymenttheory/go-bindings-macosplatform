package emit

import (
	"fmt"
	"io"
	"sort"
	"strings"

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

	for _, ext := range externs {
		goType := ext.GoType
		if goType == "" {
			goType = mapper.GoType(ext.ObjCType, ctx, imports)
		}
		if goType == "" || goType == "unsafe.Pointer" || isUnexportedXPkg(goType) {
			emitExternRaw(w, ext, dylibVarName)
			continue
		}

		fromIDCall, isClassPtr := mapper.ObjCClassFromID(goType)
		if !isClassPtr {
			fromIDCall = ""
		}
		emitExternTyped(w, ext, goType, dylibVarName, mapper.IsEnumType(goType), fromIDCall)
	}

	return imports, nil
}

func emitExternRaw(w io.Writer, ext meta.Extern, dylibVarName string) {
	if ext.Doc != "" {
		fmt.Fprintf(w, "// %s\n", ext.Doc)
	}
	emitDeprecatedComment(w, ext.Availability)
	fmt.Fprintf(w, "func %s() uintptr {\n", exportedExternName(ext.Name))
	fmt.Fprintf(w, "\tptr, _ := purego.Dlsym(%s, %q)\n", dylibVarName, ext.Name)
	fmt.Fprintf(w, "\treturn ptr\n")
	fmt.Fprintf(w, "}\n\n")
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

func emitExternTyped(w io.Writer, ext meta.Extern, goType, dylibVarName string, isEnum bool, fromIDCall string) {
	if ext.Doc != "" {
		fmt.Fprintf(w, "// %s\n", ext.Doc)
	}
	emitDeprecatedComment(w, ext.Availability)
	fmt.Fprintf(w, "func %s() %s {\n", exportedExternName(ext.Name), goType)
	fmt.Fprintf(w, "\tptr, _ := purego.Dlsym(%s, %q)\n", dylibVarName, ext.Name)
	switch {
	case fromIDCall != "":
		// ObjC object extern: the symbol holds a pointer-sized ObjC object reference.
		// Read it as objc.ID, then wrap via the typed FromID constructor.
		fmt.Fprintf(w, "\tif ptr == 0 { return nil }\n")
		fmt.Fprintf(w, "\tid := *(*objc.ID)(unsafe.Pointer(ptr))\n")
		fmt.Fprintf(w, "\tif id == 0 { return nil }\n")
		fmt.Fprintf(w, "\treturn %s\n", fromIDCall)
	case goType == "string":
		// char* extern — return as Go string.
		fmt.Fprintf(w, "\tif ptr == 0 { return \"\" }\n")
		fmt.Fprintf(w, "\treturn unsafe.String((*byte)(unsafe.Pointer(ptr)), strlen(ptr))\n")
	default:
		zero := zeroValue(goType)
		if isEnum {
			zero = "0" // enum types are integers — TypeName{} is invalid
		}
		fmt.Fprintf(w, "\tif ptr == 0 { return %s }\n", zero)
		fmt.Fprintf(w, "\treturn *(*%s)(unsafe.Pointer(ptr))\n", goType)
	}
	fmt.Fprintf(w, "}\n\n")
}
