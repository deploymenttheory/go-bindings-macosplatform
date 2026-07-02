package rawlib

import (
	"bytes"
	"fmt"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/view"
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
	return render.Execute(w, "functions_file", model)
}

// buildFunctionsModel filters eligible functions, maps their types and arguments,
// and collects all imports. The complex return-path dispatch is resolved here
// (in buildFunctionCallBody) so the template stays a structural description.
func buildFunctionsModel(pkgName, packageName string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) view.FunctionsFileModel {
	usedImports := make(typemap.ImportSet)
	ctx := m.BaseContext(framework.Framework, knownClasses)

	pkgTypeNames := packageTypeNames(framework)
	functions := make([]view.FunctionModel, 0, len(framework.Functions))
	for _, fn := range EmittableFunctions(framework) {
		functions = append(functions, buildFunctionModel(fn, functionGoName(pkgTypeNames, fn), framework.Framework, packageName, ctx, m, usedImports))
	}

	imports := buildFunctionsImports(functions, usedImports)
	return view.FunctionsFileModel{PkgName: pkgName, FwLower: packageName, Imports: imports, Functions: functions}
}

// EmittableFunctions returns the plain C functions that EmitFunctions emits as Go
// wrappers for framework, in declaration order, after applying the same skip and
// de-duplication rules (inline/variadic/unavailable/builtin/UPP/va_list/by-value
// unknown filters, plus collision with package-level type names). The idiomatic
// library layer uses this so it only ever wraps raw functions that actually exist.
func EmittableFunctions(framework *macosplatformmetadata.FrameworkMeta) []macosplatformmetadata.Function {
	pkgTypeNames := packageTypeNames(framework)

	seen := make(map[string]bool)
	out := make([]macosplatformmetadata.Function, 0, len(framework.Functions))
	for _, fn := range framework.Functions {
		if fn.IsInline || fn.IsVariadic || fn.Availability.IsUnavailable ||
			strings.HasPrefix(fn.Name, "__builtin") || isUPPFunction(fn.Name) {
			continue
		}
		if hasVAListArgFn(fn) || hasByValueUnknownType(fn) {
			continue
		}
		goName := functionGoName(pkgTypeNames, fn)
		if seen[goName] {
			continue
		}
		seen[goName] = true
		out = append(out, fn)
	}
	return out
}

// packageTypeNames returns the package-level Go type names (structs,
// struct-typedefs, enums) that function wrapper names must not collide with.
func packageTypeNames(framework *macosplatformmetadata.FrameworkMeta) map[string]bool {
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
	return pkgTypeNames
}

// FunctionGoName returns the Go wrapper name EmitFunctions gives fn. C
// permits a struct and a function to share a name (e.g. mach_time.h declares
// both a mach_timebase_info struct and function); the natural Go name then
// collides with the emitted type, so the wrapper gains an "Fn" suffix
// (Mach_timebase_infoFn) instead of being silently dropped. The idiomatic
// layer resolves its raw call targets through this same rule.
func FunctionGoName(framework *macosplatformmetadata.FrameworkMeta, fn macosplatformmetadata.Function) string {
	return functionGoName(packageTypeNames(framework), fn)
}

func functionGoName(pkgTypeNames map[string]bool, fn macosplatformmetadata.Function) string {
	goName := naming.GoTypeName(fn.Name)
	if pkgTypeNames[goName] {
		goName += "Fn"
	}
	return goName
}

// buildFunctionModel builds the model for a single C function → Go wrapper.
// goName is the collision-resolved wrapper name from functionGoName.
func buildFunctionModel(fn macosplatformmetadata.Function, goName, framework, _ string, ctx typemap.Context, m *typemap.Mapper, imports typemap.ImportSet) view.FunctionModel {
	cFunc := strings.ToLower(framework) + "_fn_" + fn.Name

	// Resolve return type.
	var retType string
	if fn.Return.ObjCType != "" {
		retType = m.GoReturnType(fn.Return.ObjCType, ctx, imports)
	}

	// Build parameter list (mapped ObjC args).
	params := strings.Join(buildGoArgs(fn.Params, false, ctx, m, imports), ", ")

	// Build CGo call argument list, collecting preambles and keep-alives.
	var preambles, keepAlives []string
	cgoArgs := buildCGOCallArgs(fn.Params, true, false, true, ctx, m, &preambles, &keepAlives, imports)

	callExpr := fmt.Sprintf("C.%s(%s)", cFunc, cgoArgs)
	callBody := buildFunctionCallBody(callExpr, retType, m)

	return view.FunctionModel{
		CommentBlock: renderCommentBlock(fn.Doc, fn.SDKFile, fn.SDKLine, fn.Availability, ""),
		BridgeID:     naming.FunctionBridgeID(framework, fn.Name),
		IsWarnUnused: fn.IsWarnUnused,
		GoName:       goName,
		Params:       params,
		Ret:          retType,
		KeepAlives:   keepAlives,
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
		fmt.Fprintf(&sb, "\tcgo.RaiseIfException(_exc)\n")

	case retType == "cgo.Object":
		fmt.Fprintf(&sb, "\t_ptr := unsafe.Pointer(%s)\n", callExpr)
		fmt.Fprintf(&sb, "\tcgo.RaiseIfException(_exc)\n")
		fmt.Fprintf(&sb, "\treturn cgo.WrapObject(_ptr)\n")

	case isObjectReturn(retType):
		structType := strings.TrimPrefix(retType, "*")
		fmt.Fprintf(&sb, "\t_ptr := unsafe.Pointer(%s)\n", callExpr)
		fmt.Fprintf(&sb, "\tcgo.RaiseIfException(_exc)\n")
		fmt.Fprintf(&sb, "\treturn %s\n", objectConstructExpr(structType, "_ptr", nil, m))

	case isValueStructReturn(retType, m):
		fmt.Fprintf(&sb, "\t_ptr := unsafe.Pointer(%s)\n", callExpr)
		fmt.Fprintf(&sb, "\tcgo.RaiseIfException(_exc)\n")
		fmt.Fprintf(&sb, "\tif _ptr == nil {\n\t\treturn %s{}\n\t}\n", retType)
		fmt.Fprintf(&sb, "\t_result := *(*%s)(unsafe.Pointer(_ptr))\n", retType)
		fmt.Fprintf(&sb, "\tcgo.FreePtr(_ptr)\n")
		fmt.Fprintf(&sb, "\treturn _result\n")

	default:
		fmt.Fprintf(&sb, "\t_result := %s\n", cgoReturnConvert(callExpr, retType, m))
		fmt.Fprintf(&sb, "\tcgo.RaiseIfException(_exc)\n")
		fmt.Fprintf(&sb, "\treturn _result\n")
	}

	return sb.String()
}

// buildFunctionsImports assembles the import list for a _functions.go file.
// It scans the pre-rendered function bodies AND signatures for tell-tale strings
// rather than re-running the type mapper a second time.
func buildFunctionsImports(functions []view.FunctionModel, usedImports typemap.ImportSet) []string {
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
		// Every generated wrapper calls cgo.RaiseIfException.
		set["github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/cgo"] = true
		if bytes.Contains(body, []byte("blocks.MakeBlock_")) {
			set["github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/blocks"] = true
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
	model := buildFunctionModel(fn, naming.GoTypeName(fn.Name), framework, packageName, ctx, m, imports)
	// Emit just the function body — tests don't need the file header.
	return render.Execute(w, "functions_file", view.FunctionsFileModel{
		PkgName:   strings.ToLower(framework),
		FwLower:   packageName,
		Imports:   []string{"unsafe", "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/cgo"},
		Functions: []view.FunctionModel{model},
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
