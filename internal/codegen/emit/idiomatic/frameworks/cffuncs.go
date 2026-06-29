//go:build darwin

package idiofw

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/view"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// isCFErrorOutParam returns true when the ObjC type is a CFErrorRef out-parameter.
func isCFErrorOutParam(objcType string) bool {
	t := normaliseObjC(objcType)
	// A CFErrorRef out-parameter is CFErrorRef * (the common case) or
	// CFErrorRef ** — possibly with inner nullability qualifiers, which
	// normaliseObjC only trims at the ends (e.g. "CFErrorRef _Nullable * _Nullable").
	if !strings.HasPrefix(t, "CFErrorRef") {
		return false
	}
	switch strings.Count(t, "*") {
	case 1, 2:
		return true
	default:
		return false
	}
}

// emitFunctionWrappers writes <pkgname>_functions_generated.go for any functions
// in framework whose last parameter is a CFErrorRef * out-parameter. Each wrapper:
//   - Omits the error out-parameter from its Go signature
//   - Passes unsafe.Pointer(&_cfErr) internally to the raw function
//   - Returns error on failure, reading the CFErrorRef as an NSError
//
// bool-returning functions become func(...) error.
// Pointer-returning functions become func(...) (unsafe.Pointer, error).
//
// Returns the set of exported wrapper names it emitted so the generic
// C-function pass can avoid redeclaring them.
func emitFunctionWrappers(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	fc *frameworkContext,
	mapper *typemap.Mapper,
	trialNames trialNameMap,
) (map[string]bool, error) {
	_ = rawPkgAlias
	_ = rawPkgPath
	framework := fc.framework
	ctx := typemap.Context{Framework: framework.Framework}

	emittedNames := make(map[string]bool)
	// Every CFError wrapper binds the symbol (ebipurego), holds the CFErrorRef
	// out-parameter (unsafe), and converts it to a Go error (errkit). Parameter
	// type imports come from the type mapper; a pointer-returning wrapper adds
	// obj and objc for its obj.Wrap(objc.ID) result.
	imports := map[string]string{
		"ebipurego": ebipuregoImportPath,
		"unsafe":    "unsafe",
		"errkit":    errkitImportPath,
	}

	var funcs []view.Func
	for _, fn := range rawfw.EmittableFunctions(framework, nil) {
		if len(fn.Params) == 0 || !isCFErrorOutParam(fn.Params[len(fn.Params)-1].ObjCType) {
			continue
		}

		// Classify every parameter except the trailing error out-parameter.
		var sigParts, abiParts, callArgs []string
		fnImports := map[string]string{}
		ok := true
		usedNames := make(map[string]int)
		for _, param := range fn.Params[:len(fn.Params)-1] {
			pName := safeParamName(naming.ParamName(param.Name))
			if pName == "" {
				pName = fmt.Sprintf("arg%d", len(sigParts))
			}
			usedNames[pName]++
			if usedNames[pName] > 1 {
				pName = fmt.Sprintf("%s%d", pName, usedNames[pName])
			}
			sig, argExpr, imports, pok := idiomaticArg(
				pName,
				param.ObjCType,
				ctx,
				mapper,
				fc,
				rawPkgAlias,
				trialNames,
			)
			if !pok {
				ok = false
				break
			}
			maps.Copy(fnImports, imports)
			sigParts = append(sigParts, pName+" "+sig)
			abiParts = append(abiParts, cfuncABIType(sig, argExpr))
			callArgs = append(callArgs, argExpr)
		}
		if !ok {
			mapper.AppendDiagnostic(
				"%s: idiomatic wrapper for %s left out (a parameter type is not yet expressible)",
				framework.Framework,
				fn.Name,
			)
			continue
		}

		goName := naming.ExportedFunctionName(fn.Name)
		if goName == "" {
			continue
		}
		emittedNames[goName] = true
		// The wrapper is committed: merge its parameter type imports.
		maps.Copy(imports, fnImports)
		varName := "_fn" + goName
		// The bound C function takes the same arguments plus the error
		// out-parameter pointer.
		argsWithErr := strings.Join(
			append(append([]string{}, callArgs...), "unsafe.Pointer(&_cfErr)"),
			", ",
		)

		retObjC := strings.TrimSpace(fn.Return.ObjCType)
		goRet := mapper.GoType(fn.Return.ObjCType, ctx, make(typemap.ImportSet))
		retIsBool := retObjC == "bool" || retObjC == "BOOL" || retObjC == "_Bool" ||
			retObjC == "Boolean" || goRet == "bool"

		// A reference parameter crosses the ABI as an objc.ID, which names objc.
		if slices.Contains(abiParts, "objc.ID") {
			imports["objc"] = objcImportPath
		}
		comment := fmt.Sprintf(
			"// %s reports an error if the %s framework function %s fails.\n",
			goName,
			framework.Framework,
			fn.Name,
		)
		if retIsBool {
			abiRet, fail := "bool", "!_ok"
			if goRet != "bool" {
				abiRet, fail = "uint8", "_ok == 0"
			}
			funcs = append(funcs, view.Func{
				GoName:       goName,
				CName:        fn.Name,
				VarName:      varName,
				CommentLine:  comment,
				CommentFirst: true,
				ABIParams:    append(append([]string{}, abiParts...), "unsafe.Pointer"),
				ABIRet:       abiRet,
				SigParams:    sigParts,
				RetSig:       " error",
				Kind:         view.FuncCFErrorBool,
				PreLines:     []string{"var _cfErr unsafe.Pointer"},
				Call:         fmt.Sprintf("%s(%s)", varName, argsWithErr),
				Fail:         fail,
			})
		} else {
			// A pointer-returning function: the result is a toll-free-bridged
			// object, surfaced together with the error.
			imports["obj"] = objImportPath
			imports["objc"] = objcImportPath
			funcs = append(funcs, view.Func{
				GoName:       goName,
				CName:        fn.Name,
				VarName:      varName,
				CommentLine:  comment,
				CommentFirst: true,
				ABIParams:    append(append([]string{}, abiParts...), "unsafe.Pointer"),
				ABIRet:       "objc.ID",
				SigParams:    sigParts,
				RetSig:       " (obj.Object, error)",
				Kind:         view.FuncCFErrorPtr,
				PreLines:     []string{"var _cfErr unsafe.Pointer"},
				Call:         fmt.Sprintf("%s(%s)", varName, argsWithErr),
			})
		}
	}

	if len(funcs) == 0 {
		return emittedNames, nil
	}

	body, err := render.Funcs(funcs)
	if err != nil {
		return nil, err
	}
	fname := pkgName + "_functions_generated.go"
	file := assembleFile(pkgName, imports, body)
	if err := rawfw.WriteGoFile(filepath.Join(outDir, fname), file); err != nil {
		return nil, err
	}
	return emittedNames, nil
}
