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

// FunctionRegistration pairs a C symbol with its purego.RegisterLibFunc call
// so the runtime emitter can record per-symbol registration failures.
type FunctionRegistration struct {
	Symbol string
	Line   string
}

// EmitFunctions writes Go var declarations and wrapper functions for free C
// functions in the framework. Registration lines (purego.RegisterLibFunc calls)
// are returned as regLines so they can be embedded in the runtime.go init()
// AFTER a successful Dlopen — preventing the init-order bug where a separate
// _functions.go init() would run before _runtime.go's init() and see a zero
// library handle.
// ownerIndex maps ObjC class names to their owning framework — used to
// distinguish ObjC object pointers (need .Ptr()) from C struct pointers.
// Returns the cross-framework imports discovered and the registration lines.
func EmitFunctions(
	w io.Writer,
	framework *meta.FrameworkMeta,
	mapper *typemap.Mapper,
	dylibVarName string,
	ownerIndex map[string]string,
) (imports typemap.ImportSet, regLines []FunctionRegistration, err error) {
	imports = make(typemap.ImportSet)

	fns := EmittableFunctions(framework, mapper.AppendDiagnostic)
	if len(fns) == 0 {
		return imports, nil, nil
	}

	ctx := typemap.Context{Framework: framework.Framework}

	// Emit var declarations for each function binding.
	// ObjC object params and returns use objc.ID in the var (matching the raw
	// C ABI via purego), while wrapper functions use the high-level Go types.
	fmt.Fprintf(w, "var (\n")
	for _, fn := range fns {
		retType := ""
		if _, retIsBlock := mapper.ResolveBlockSignature(fn.Return.ObjCType); retIsBlock {
			retType = "objc.Block"
		} else if fn.Return.ObjCType != "void" && fn.Return.ObjCType != "" {
			retType = mapper.GoReturnType(fn.Return.ObjCType, ctx, imports)
			if retType == "" || isUnexportedXPkg(retType) {
				retType = "unsafe.Pointer"
			}
		}

		var params []string
		for _, param := range fn.Params {
			// Block params cross the C ABI as real block objects.
			if _, isBlock := mapper.ResolveBlockSignature(param.ObjCType); isBlock {
				params = append(params, "objc.Block")
				continue
			}
			goType := mapper.GoType(param.ObjCType, ctx, imports)
			if goType == "" || isUnexportedXPkg(goType) {
				goType = "unsafe.Pointer"
			}
			// ObjC objects are passed as raw objc.ID at the C level.
			if isObjCObjectType(goType) && isObjCClass(goType, ownerIndex) {
				goType = "objc.ID"
			}
			params = append(params, goType)
		}

		// ObjC object returns are also raw objc.ID at the C level.
		varRetType := retType
		if isObjCObjectType(retType) && isObjCClass(retType, ownerIndex) {
			varRetType = "objc.ID"
		}

		funcType := "func(" + strings.Join(params, ", ") + ")"
		if varRetType != "" {
			funcType += " " + varRetType
		}

		if fn.Doc != "" {
			fmt.Fprintf(w, "\t// %s\n", fn.Doc)
		}
		emitDeprecatedComment(w, fn.Availability)
		funcVarName := naming.LowerFirst(fn.Name)
		if funcVarName == fn.Name {
			// Already lowercase — prefix with underscore to make it unexported.
			funcVarName = "_" + funcVarName
		} else {
			funcVarName = "_fn" + fn.Name
		}
		fmt.Fprintf(w, "\t%s %s\n", funcVarName, funcType)
	}
	fmt.Fprintf(w, ")\n\n")

	// Collect registration lines to be embedded in the runtime.go init()
	// after a successful Dlopen — callers must not generate a separate init().
	for _, fn := range fns {
		funcVarName := "_fn" + fn.Name
		if naming.LowerFirst(fn.Name) == fn.Name {
			funcVarName = "_" + fn.Name
		}
		regLines = append(regLines, FunctionRegistration{
			Symbol: fn.Name,
			Line: fmt.Sprintf(
				"purego.RegisterLibFunc(&%s, %s, %q)",
				funcVarName,
				dylibVarName,
				fn.Name,
			),
		})
	}

	// Emit exported wrapper functions.
	for _, fn := range fns {
		ctx2 := typemap.Context{Framework: framework.Framework}
		if err := emitFunction(w, fn, ctx2, mapper, imports, ownerIndex); err != nil {
			return nil, nil, err
		}
	}

	return imports, regLines, nil
}

func emitFunction(
	w io.Writer,
	fn meta.Function,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	imports typemap.ImportSet,
	ownerIndex map[string]string,
) error {
	retType := ""
	retIsBlock := false
	if _, retIsBlock = mapper.ResolveBlockSignature(fn.Return.ObjCType); retIsBlock {
		// Reverse-bridging a returned block into a Go func is out of scope;
		// surface the raw block object.
		retType = "objc.Block"
	} else if fn.Return.ObjCType != "void" && fn.Return.ObjCType != "" {
		retType = mapper.GoReturnType(fn.Return.ObjCType, ctx, imports)
		if retType == "" || isUnexportedXPkg(retType) {
			retType = "unsafe.Pointer"
		}
	}

	var goParams []string
	var callArgs []string
	var adapters []blockAdapterModel
	usedNames := make(map[string]int)
	for _, param := range fn.Params {
		paramName := naming.ParamName(param.Name)
		// Deduplicate param names within this function.
		usedNames[paramName]++
		if usedNames[paramName] > 1 {
			paramName = fmt.Sprintf("%s%d", paramName, usedNames[paramName])
		}

		if _, isBlock := mapper.ResolveBlockSignature(param.ObjCType); isBlock {
			adapter := buildBlockAdapter(
				paramName,
				param.ObjCType,
				ctx,
				mapper,
				imports,
				ownerIndex,
			)
			goParams = append(goParams, paramName+" "+adapter.PublicGoType)
			if adapter.Degraded {
				mapper.AppendDiagnostic(
					"%s: %s param %s → objc.Block (%s)",
					ctx.Framework, fn.Name, paramName, adapter.DegradeReason,
				)
				callArgs = append(callArgs, paramName)
			} else {
				adapters = append(adapters, adapter)
				callArgs = append(callArgs, "__block_"+paramName)
			}
			continue
		}

		goType := mapper.GoType(param.ObjCType, ctx, imports)
		if goType == "" || isUnexportedXPkg(goType) {
			goType = "unsafe.Pointer"
		}
		goParams = append(goParams, paramName+" "+goType)
		// Only call .Ptr() for ObjC objects (in OwnerIndex), not C struct pointers.
		if isObjCObjectType(goType) && isObjCClass(goType, ownerIndex) {
			callArgs = append(callArgs, paramName+".Ptr()")
		} else {
			callArgs = append(callArgs, paramName)
		}
	}

	funcVarName := "_fn" + fn.Name
	if naming.LowerFirst(fn.Name) == fn.Name {
		funcVarName = "_" + fn.Name
	}

	paramStr := strings.Join(goParams, ", ")
	callStr := strings.Join(callArgs, ", ")

	retSig := ""
	if retType != "" {
		retSig = " " + retType
	}

	if fn.Doc != "" {
		fmt.Fprintf(w, "// %s\n", fn.Doc)
	}
	goName := naming.ExportedFunctionName(fn.Name)
	if goName != fn.Name {
		fmt.Fprintf(w, "// C function: %s\n", fn.Name)
	}
	emitDeprecatedComment(w, fn.Availability)
	fmt.Fprintf(w, "func %s(%s)%s {\n", goName, paramStr, retSig)
	for _, adapter := range adapters {
		writeBlockAdapter(w, adapter, "\t")
	}
	if retType != "" {
		if !retIsBlock && isObjCObjectType(retType) && isObjCClass(retType, ownerIndex) {
			fmt.Fprintf(w, "\t_ret := %s(%s)\n", funcVarName, callStr)
			fmt.Fprintf(w, "\tif _ret != 0 { _ret.Send(objc.RegisterName(\"retain\")) }\n")
			wrapExpr := buildWrapExprFromType(retType)
			fmt.Fprintf(w, "\treturn %s\n", wrapExpr)
		} else {
			fmt.Fprintf(w, "\treturn %s(%s)\n", funcVarName, callStr)
		}
	} else {
		fmt.Fprintf(w, "\t%s(%s)\n", funcVarName, callStr)
	}
	fmt.Fprintf(w, "}\n\n")
	return nil
}

// EmittableFunctions returns the free C functions the raw emitter emits for
// framework, sorted by name. It applies the skip rules (unavailable, inline,
// non-format variadic, duplicates) and the exported-name collision rules
// (naming.ExportedFunctionName vs other package-level identifiers). The
// optional diag callback receives a message for each collision skip; pass nil
// to filter silently. Downstream emitters (idiomatic layer, genacceptance)
// must use this so they cannot drift from the raw emission set.
func EmittableFunctions(
	framework *meta.FrameworkMeta,
	diag func(format string, args ...any),
) []meta.Function {
	seen := make(map[string]bool)
	usedGoNames := seedReservedGoNames(framework)
	var fns []meta.Function

	passesFilters := func(fn meta.Function) bool {
		if fn.Availability.IsUnavailable || fn.IsInline {
			return false
		}
		if fn.IsVariadic && !isFuncFormatVariadic(fn.Name) {
			return false
		}
		return true
	}

	// Pass 1: functions whose C name is already a valid exported Go name keep
	// it byte-identical — they must win every collision so existing consumers
	// never see a rename (e.g. CGSizeEqualToSize must not lose its name to
	// the transformed __CGSizeEqualToSize).
	for _, fn := range framework.Functions {
		if !passesFilters(fn) || seen[fn.Name] {
			continue
		}
		goName := naming.ExportedFunctionName(fn.Name)
		if goName != fn.Name {
			continue
		}
		seen[fn.Name] = true
		if usedGoNames[goName] {
			if diag != nil {
				diag(
					"%s: function %s skipped (exported name %s collides with an existing identifier)",
					framework.Framework,
					fn.Name,
					goName,
				)
			}
			continue
		}
		usedGoNames[goName] = true
		fns = append(fns, fn)
	}

	// Pass 2: snake_case (and other previously-unexported) names, transformed
	// to PascalCase. Collisions with pass-1 names or other identifiers skip
	// the function with a diagnostic.
	for _, fn := range framework.Functions {
		if !passesFilters(fn) || seen[fn.Name] {
			continue
		}
		seen[fn.Name] = true
		goName := naming.ExportedFunctionName(fn.Name)
		if goName == "" {
			continue
		}
		if usedGoNames[goName] {
			if diag != nil {
				diag(
					"%s: function %s skipped (exported name %s collides with an existing identifier)",
					framework.Framework,
					fn.Name,
					goName,
				)
			}
			continue
		}
		usedGoNames[goName] = true
		fns = append(fns, fn)
	}

	sort.Slice(fns, func(i, j int) bool { return fns[i].Name < fns[j].Name })
	return fns
}

// seedReservedGoNames builds the set of package-level Go identifiers produced
// by the other emitters for this framework (enum types and members, struct
// types, class wrappers and their FromID constructors, protocol interfaces)
// so exported function names cannot collide with them.
func seedReservedGoNames(framework *meta.FrameworkMeta) map[string]bool {
	reserved := make(map[string]bool)
	for enumName, enum := range framework.Enums {
		reserved[naming.GoTypeName(enumName)] = true
		for _, member := range enum.Members {
			reserved[naming.GoTypeName(member.Name)] = true
		}
	}
	for structName := range framework.Structs {
		reserved[naming.ExportedTypeName(structName)] = true
	}
	for className := range framework.Classes {
		reserved[className] = true
		reserved[className+"FromID"] = true
	}
	for protocolName := range framework.Protocols {
		reserved[naming.GoTypeName(protocolName)] = true
		reserved[naming.GoTypeName(protocolName)+"Protocol"] = true
	}
	return reserved
}

// buildWrapExprFromType builds a wrap expression for a return type string.
// Handles generic types like *NSArray[objc.ID] → NSArrayFromID[objc.ID](_ret).
func buildWrapExprFromType(retGoType string) string {
	return buildWrapExprFromTypeExpr(retGoType, "_ret")
}

// buildWrapExprFromTypeExpr builds the FromID wrap expression for an
// arbitrary objc.ID-typed expression (e.g. a block adapter argument).
func buildWrapExprFromTypeExpr(retGoType, expr string) string {
	if !strings.HasPrefix(retGoType, "*") {
		return expr
	}
	inner := retGoType[1:]
	// Split off generics before checking for cross-package dot.
	baseType, typeArgs := splitGenericNameF(inner)
	if dotIdx := strings.Index(baseType, "."); dotIdx >= 0 {
		pkg := baseType[:dotIdx]
		typName := baseType[dotIdx+1:]
		if typeArgs != "" {
			return fmt.Sprintf("%s.%sFromID[%s](%s)", pkg, typName, typeArgs, expr)
		}
		return fmt.Sprintf("%s.%sFromID(%s)", pkg, typName, expr)
	}
	if typeArgs != "" {
		return fmt.Sprintf("%sFromID[%s](%s)", baseType, typeArgs, expr)
	}
	return fmt.Sprintf("%sFromID(%s)", baseType, expr)
}

// splitGenericNameF splits "NSArray[objc.ID]" into ("NSArray", "objc.ID").
func splitGenericNameF(name string) (base, args string) {
	brIdx := strings.Index(name, "[")
	if brIdx < 0 {
		return name, ""
	}
	return name[:brIdx], strings.TrimSuffix(name[brIdx+1:], "]")
}

// isObjCClass reports whether a Go pointer type (e.g. *NSString or
// *foundation.NSString) refers to an ObjC class (has a .Ptr() method)
// rather than a plain C struct pointer.
func isObjCClass(goType string, ownerIndex map[string]string) bool {
	if ownerIndex == nil {
		return false
	}
	if !strings.HasPrefix(goType, "*") {
		return false
	}
	inner := goType[1:]
	// Strip package qualifier: "foundation.NSString" → "NSString"
	if dotIdx := strings.LastIndex(inner, "."); dotIdx >= 0 {
		inner = inner[dotIdx+1:]
	}
	// Strip generic params: "NSArray[T]" → "NSArray"
	if brIdx := strings.Index(inner, "["); brIdx >= 0 {
		inner = inner[:brIdx]
	}
	_, ok := ownerIndex[inner]
	return ok
}

// isFuncFormatVariadic reports whether a function name suggests format-string
// variadic usage (bridgeable via pre-formatted string).
func isFuncFormatVariadic(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "format") || strings.Contains(lower, "printf") ||
		strings.Contains(lower, "sprintf")
}
