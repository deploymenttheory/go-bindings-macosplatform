package raw

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// Functions writes a complete _functions.go file for the framework's plain C functions.
func EmitFunctions(w io.Writer, pkgName, packageName string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) error {
	model := buildFunctionsModel(pkgName, packageName, framework, m, knownClasses)
	return executeTemplate(w, "functions_file", model)
}

// buildFunctionsModel filters eligible functions, maps their types and arguments,
// and collects all imports. The complex return-path dispatch is resolved here
// (in buildFunctionCallBody) so the template stays a structural description.
func buildFunctionsModel(pkgName, packageName string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) functionsFileModel {
	usedImports := make(typemap.ImportSet)
	ctx := m.BaseContext(framework.Framework, knownClasses)

	// Build a set of Go type names to detect name collisions with package-level types.
	pkgTypeNames := make(map[string]bool)
	for n := range framework.Structs {
		pkgTypeNames[naming.GoTypeName(n)] = true
	}
	for n, target := range framework.Typedefs {
		if strings.HasPrefix(target, "struct ") {
			pkgTypeNames[naming.GoTypeName(n)] = true
		}
	}
	for n := range framework.Enums {
		pkgTypeNames[naming.GoTypeName(n)] = true
	}

	seen := make(map[string]bool)
	functions := make([]functionModel, 0, len(framework.Functions))

	for _, fn := range framework.Functions {
		if fn.IsInline || fn.IsVariadic || fn.Availability.IsUnavailable ||
			strings.HasPrefix(fn.Name, "__builtin") || isUPPFunction(fn.Name) {
			continue
		}
		if hasVAListArgFn(fn) || hasByValueUnknownType(fn) {
			continue
		}
		goName := naming.GoTypeName(fn.Name)
		if seen[goName] || pkgTypeNames[goName] {
			continue
		}
		seen[goName] = true
		functions = append(functions, buildFunctionModel(fn, framework.Framework, packageName, ctx, m, usedImports))
	}

	imports := buildFunctionsImports(functions, usedImports)
	return functionsFileModel{PkgName: pkgName, FwLower: packageName, Imports: imports, Functions: functions}
}

// buildFunctionModel builds the model for a single C function → Go wrapper.
func buildFunctionModel(fn macosplatformmetadata.Function, framework, _ string, ctx typemap.Context, m *typemap.Mapper, imports typemap.ImportSet) functionModel {
	cFunc := strings.ToLower(framework) + "_fn_" + fn.Name
	goName := naming.GoTypeName(fn.Name)

	// Resolve return type.
	var retType string
	if fn.Return.ObjCType != "" {
		retType = m.GoReturnType(fn.Return.ObjCType, ctx, imports)
	}

	// Build parameter list (ctx first, then mapped ObjC args).
	goArgs := append([]string{"ctx context.Context"}, buildGoArgs(fn.Params, false, ctx, m, imports)...)
	params := strings.Join(goArgs, ", ")

	// Build CGo call argument list, collecting preambles and keep-alives.
	var preambles, keepAlives []string
	cgoArgs := buildCGOCallArgs(fn.Params, true, false, true, ctx, m, &preambles, &keepAlives, imports)

	callExpr := fmt.Sprintf("C.%s(%s)", cFunc, cgoArgs)
	callBody := buildFunctionCallBody(callExpr, retType, m)

	return functionModel{
		CommentBlock: renderCommentBlock(fn.Doc, fn.SDKFile, fn.SDKLine, fn.Availability, ""),
		BridgeID:     naming.FunctionBridgeID(framework, fn.Name),
		IsWarnUnused:   fn.IsWarnUnused,
		GoName:       goName,
		Params:       params,
		Ret:          retType,
		SpanName:     fmt.Sprintf("%s/%s", framework, fn.Name),
		TelExtra:     buildTelCallExtra(keepAlives),
		Preambles:    preambles,
		CallBody:     callBody,
	}
}

// buildFunctionCallBody pre-renders the CGo call + exception check + optional return
// for a function with the given return type. The multi-path dispatch lives here in Go
// so the template can remain a structural description of the function wrapper.
func buildFunctionCallBody(callExpr, retType string, m *typemap.Mapper) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "\tvar _exc unsafe.Pointer\n")

	switch {
	case retType == "":
		fmt.Fprintf(&sb, "\t%s\n", callExpr)
		fmt.Fprintf(&sb, "\ttel.RaiseIfException(ctx, _exc)\n")

	case retType == "cgo.Object":
		fmt.Fprintf(&sb, "\t_ptr := unsafe.Pointer(%s)\n", callExpr)
		fmt.Fprintf(&sb, "\ttel.RaiseIfException(ctx, _exc)\n")
		fmt.Fprintf(&sb, "\treturn cgo.WrapObject(_ptr)\n")

	case isObjectReturn(retType):
		structType := strings.TrimPrefix(retType, "*")
		fmt.Fprintf(&sb, "\t_ptr := unsafe.Pointer(%s)\n", callExpr)
		fmt.Fprintf(&sb, "\ttel.RaiseIfException(ctx, _exc)\n")
		fmt.Fprintf(&sb, "\treturn %s\n", objectConstructExpr(structType, "_ptr", nil, m))

	case isValueStructReturn(retType, m):
		fmt.Fprintf(&sb, "\t_ptr := unsafe.Pointer(%s)\n", callExpr)
		fmt.Fprintf(&sb, "\ttel.RaiseIfException(ctx, _exc)\n")
		fmt.Fprintf(&sb, "\tif _ptr == nil {\n\t\treturn %s{}\n\t}\n", retType)
		fmt.Fprintf(&sb, "\t_result := *(*%s)(unsafe.Pointer(_ptr))\n", retType)
		fmt.Fprintf(&sb, "\tcgo.FreePtr(_ptr)\n")
		fmt.Fprintf(&sb, "\treturn _result\n")

	default:
		fmt.Fprintf(&sb, "\t_result := %s\n", cgoReturnConvert(callExpr, retType, m))
		fmt.Fprintf(&sb, "\ttel.RaiseIfException(ctx, _exc)\n")
		fmt.Fprintf(&sb, "\treturn _result\n")
	}

	return sb.String()
}

// buildFunctionsImports assembles the import list for a _functions.go file.
// It scans the pre-rendered function bodies AND signatures for tell-tale strings
// rather than re-running the type mapper a second time.
func buildFunctionsImports(functions []functionModel, usedImports typemap.ImportSet) []string {
	// Scan call bodies, preambles, and signatures (return types can reference cgo.Object).
	var combined bytes.Buffer
	for _, fn := range functions {
		combined.WriteString(fn.CallBody)
		combined.WriteString(fn.Params)
		combined.WriteString(fn.Ret)
		for _, p := range fn.Preambles {
			combined.WriteString(p)
		}
	}
	body := combined.Bytes()

	set := map[string]bool{"unsafe": true}
	if len(functions) > 0 {
		set["context"] = true
		set["github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/tel"] = true
		if bytes.Contains(body, []byte("blocks.MakeBlock_")) {
			set["github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/blocks"] = true
		}
		if bytes.Contains(body, []byte("cgo.")) {
			set["github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/cgo"] = true
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

// writeFunction is a thin adapter used by tests to exercise individual function
// generation without going through the full file model.
func writeFunction(w io.Writer, fn macosplatformmetadata.Function, framework string, ctx typemap.Context, m *typemap.Mapper) error {
	packageName := strings.ToLower(framework)
	imports := make(typemap.ImportSet)
	model := buildFunctionModel(fn, framework, packageName, ctx, m, imports)
	// Emit just the function body — tests don't need the file header.
	return executeTemplate(w, "functions_file", functionsFileModel{
		PkgName:   strings.ToLower(framework),
		FwLower:   packageName,
		Imports:   []string{"context", "unsafe", "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/tel"},
		Functions: []functionModel{model},
	})
}

func hasVAListArgFn(fn macosplatformmetadata.Function) bool {
	for _, arg := range fn.Params {
		n := strings.ToLower(typemap.Normalise(arg.ObjCType))
		if strings.Contains(n, "va_list") {
			return true
		}
	}
	return false
}
