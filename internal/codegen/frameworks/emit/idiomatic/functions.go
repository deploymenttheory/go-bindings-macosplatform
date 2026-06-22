//go:build darwin

package idiomatic

import (
	"bytes"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/idiomatic/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/idiomatic/view"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// emitGenericFunctionWrappers writes <pkgname>_cfunctions_generated.go with
// one exported forwarding wrapper per emittable raw C function, giving
// C-function-only frameworks (e.g. vmnet) a usable idiomatic surface:
//
//   - The framework-name prefix is stripped when present
//     (raw.VmnetStartInterface → vmnet.StartInterface).
//   - Parameter and return types mirror the raw signature exactly (block
//     params surface as the same Go closure types the raw wrappers take).
//   - Functions already wrapped by the CFError pass are skipped.
//
// Name collisions fall back to the full exported name; if that is also taken
// the function is skipped with a diagnostic. Iteration is in sorted order so
// the outcome is deterministic.
// ebipuregoImportPath is the ebitengine purego package, used to bind a C
// function symbol to a Go function variable at runtime.
const ebipuregoImportPath = "github.com/ebitengine/purego"

// cfuncABIType returns the type a parameter has when it is handed to the bound C
// function, given its idiomatic type (sig) and the expression that produces the
// argument (argExpr). When the argument is converted to an Objective-C pointer
// (an object, an NSString, a URL, or an array), the C function receives an
// objc.ID; otherwise the value — a C string, a number, an enum, or a value
// struct — is passed unchanged.
func cfuncABIType(sig, argExpr string) string {
	if strings.HasPrefix(argExpr, "objc.NewBlock(") {
		return "objc.Block"
	}
	for _, p := range []string{"purego.NSString(", "objref.IDOf(", "purego.SliceToNSArray(", "rt.FileURL("} {
		if strings.HasPrefix(argExpr, p) {
			return "objc.ID"
		}
	}
	return sig
}

// cfuncRetABI returns the type the bound C function returns, before it is
// converted back to the idiomatic return type.
func cfuncRetABI(kind objKind, retType string) string {
	switch kind {
	case kindVoid:
		return ""
	case kindString, kindObject, kindArray:
		return "objc.ID"
	default:
		return retType
	}
}

func emitGenericFunctionWrappers(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	fc *frameworkContext,
	m *typemap.Mapper,
	trialNames trialNameMap,
	takenNames map[string]bool,
) error {
	_ = rawPkgAlias
	_ = rawPkgPath
	fw := fc.fw
	ctx := typemap.Context{Framework: fw.Framework}
	prefix := naming.GoTypeName(strings.ToLower(fw.Framework))

	// Imports are accumulated from what each wrapper actually emits: ebipurego is
	// always used to bind the symbol, parameter/return type imports come from the
	// type mapper, and objc is added below when a parameter or the return crosses
	// the ABI as an objc.ID.
	imports := map[string]string{"ebipurego": ebipuregoImportPath}

	var funcs []view.Func
	for _, fn := range emit.EmittableFunctions(fw, nil) {
		// Classify every parameter and the return. If any cannot yet be expressed
		// idiomatically (a function pointer, a varargs, an out-parameter), the
		// whole function is left out and a diagnostic is recorded. Type imports are
		// gathered per function and merged only once the wrapper is committed, so a
		// skipped function never contributes an unused import.
		var sigParts, abiParts, callArgs []string
		fnImports := map[string]string{}
		ok := true
		usedNames := make(map[string]int)
		for _, param := range fn.Params {
			pName := safeParamName(naming.ParamName(param.Name))
			if pName == "" {
				pName = fmt.Sprintf("arg%d", len(sigParts))
			}
			usedNames[pName]++
			if usedNames[pName] > 1 {
				pName = fmt.Sprintf("%s%d", pName, usedNames[pName])
			}
			sig, argExpr, imps, pok := idiomaticArg(
				pName,
				param.ObjCType,
				ctx,
				m,
				fc,
				rawPkgAlias,
				trialNames,
			)
			if !pok {
				ok = false
				break
			}
			maps.Copy(fnImports, imps)
			sigParts = append(sigParts, pName+" "+sig)
			abiParts = append(abiParts, cfuncABIType(sig, argExpr))
			callArgs = append(callArgs, argExpr)
		}
		if !ok {
			m.AppendDiagnostic(
				"%s: idiomatic wrapper for %s left out (a parameter type is not yet expressible)",
				fw.Framework,
				fn.Name,
			)
			continue
		}
		retType, kind, wrap, _, rimps, rok := idiomaticRet(
			fn.Return.ObjCType,
			ctx,
			m,
			fc,
			rawPkgAlias,
			trialNames,
		)
		if !rok {
			m.AppendDiagnostic(
				"%s: idiomatic wrapper for %s left out (the return type is not yet expressible)",
				fw.Framework,
				fn.Name,
			)
			continue
		}
		maps.Copy(fnImports, rimps)

		goName := naming.ExportedFunctionName(fn.Name)
		if stripped := strings.TrimPrefix(goName, prefix); stripped != goName &&
			stripped != "" && stripped[0] >= 'A' && stripped[0] <= 'Z' {
			if !takenNames[stripped] {
				goName = stripped
			}
		}
		if goName == "" || takenNames[goName] {
			continue
		}
		takenNames[goName] = true

		varName := "_fn" + goName
		retSig := ""
		if retType != "" {
			retSig = " " + retType
		}
		abiRet := cfuncRetABI(kind, retType)
		// The wrapper is committed: merge its type imports, and add objc when a
		// parameter or the return crosses the ABI as an objc.ID.
		maps.Copy(imports, fnImports)
		if abiRet == "objc.ID" || slices.Contains(abiParts, "objc.ID") {
			imports["objc"] = objcImportPath
		}
		funcs = append(funcs, view.Func{
			GoName:  goName,
			CName:   fn.Name,
			VarName: varName,
			CommentLine: fmt.Sprintf(
				"// %s calls the %s framework function %s.\n",
				goName,
				fw.Framework,
				fn.Name,
			),
			ABIParams: abiParts,
			ABIRet:    abiRet,
			SigParams: sigParts,
			RetSig:    retSig,
			Kind:      genericFuncKind(kind),
			Call:      fmt.Sprintf("%s(%s)", varName, strings.Join(callArgs, ", ")),
			Wrap:      wrap,
		})
	}

	if len(funcs) == 0 {
		return nil
	}
	body, err := render.Funcs(funcs)
	if err != nil {
		return err
	}

	fname := pkgName + "_cfunctions_generated.go"
	return emit.WriteGoFile(
		filepath.Join(outDir, fname),
		assembleFile(pkgName, imports, body),
	)
}

// genericFuncKind maps the idiomatic return classification to the FuncKind a
// generic C-function wrapper renders. Object and array results share the wrap
// path; everything else returns the raw value.
func genericFuncKind(k objKind) view.FuncKind {
	switch k {
	case kindVoid:
		return view.FuncVoid
	case kindString:
		return view.FuncString
	case kindObject, kindArray:
		return view.FuncObject
	default:
		return view.FuncScalar
	}
}

// emitClassMethodFunctions writes <pkgname>_classmethods_generated.go: one
// package-level forwarding function per ObjC class (static) method in fw. These
// are the factory and accessor methods the rest of the idiomatic layer omits —
// instance methods get receivers and inits become New* constructors, but class
// methods (e.g. +[VZBridgedNetworkInterface networkInterfaces],
// +[NSProcessInfo processInfo]) had no idiomatic surface and forced callers to
// drop to the raw bindings.
//
// Each wrapper forwards to the raw package-level function the raw emitter
// produced for the class method (raw.<Class><Method>), reusing the shared method
// machinery so array returns become []T slices, NSString returns become Go
// strings, same-package class returns become wrapped trial types, BoolNSError
// becomes error and completion handlers become (ctx) … methods.
//
// Names prefer a fluent, class-prefix-stripped form (NetworkInterfaces); on any
// collision — including the wrapper type of the same name — they fall back to a
// class-qualified form, then skip with a diagnostic. takenNames is updated so the
// generic C-function pass cannot redeclare a name reserved here.
func emitClassMethodFunctions(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	fc *frameworkContext,
	m *typemap.Mapper,
	trialNames trialNameMap,
	handFuncs map[string]bool,
	takenNames map[string]bool,
	abstractBases abstractBaseIndex,
) error {
	fw := fc.fw
	// Every wrapper dispatches to a class object (objc.ID(_class(...))), so objc is
	// always used; every other import is contributed by an emitted method's own
	// import set. objref is added per method only when an argument actually marshals
	// through objref.IDOf, since (unlike an instance method) the class receiver does
	// not use it.
	imports := map[string]string{"objc": objcImportPath}

	var body bytes.Buffer
	for _, className := range sortedKeys(fw.Classes) {
		cls := fw.Classes[className]
		if cls.Availability.IsUnavailable {
			continue
		}
		ctx := typemap.Context{
			ClassName:     className,
			Framework:     fw.Framework,
			GenericParams: cls.GenericParams,
		}
		rawNames := classRawMethodNames(cls, className, fw)

		seenSel := map[string]bool{}
		for _, method := range cls.Methods {
			if !method.IsClassMethod || !emit.MethodWillBeEmitted(method) {
				continue
			}
			if seenSel[method.Selector] {
				continue
			}
			seenSel[method.Selector] = true

			rawName := rawNames[method.Selector]
			if rawName == "" {
				continue
			}
			entry := buildMethod(
				method,
				rawName,
				cls,
				fc,
				ctx,
				m,
				rawPkgAlias,
				trialNames,
				abstractBases,
			)
			if entry == nil {
				continue
			}

			// The emitted function name is independent of the raw symbol it
			// calls (entry.rawGoName), so pick the most fluent free name.
			fluent := classFuncShortName(entry.goName, className)
			if fluent == "" {
				fluent = entry.goName
			}
			qualified := className + fluent
			var name string
			switch {
			case isFreeFuncName(fluent, takenNames, handFuncs):
				name = fluent
			case isFreeFuncName(qualified, takenNames, handFuncs):
				name = qualified
			default:
				m.AppendDiagnostic(
					"%s: idiomatic class-method wrapper for +[%s %s] skipped (names %s/%s already taken)",
					fw.Framework,
					className,
					method.Selector,
					fluent,
					qualified,
				)
				continue
			}
			takenNames[name] = true
			entry.goName = name

			fnImps := maps.Clone(entry.extraImports)
			if !classFuncUsesObjref(*entry) {
				// The class receiver is objc.ID(_class(...)), not objref.IDOf(x);
				// objref is needed only when an argument marshals through it.
				delete(fnImps, "objref")
			}
			maps.Copy(imports, fnImps)
			writeClassFunc(&body, fmt.Sprintf("objc.ID(_class(%q))", className), *entry)
		}
	}

	if body.Len() == 0 {
		return nil
	}

	fname := pkgName + "_classmethods_generated.go"
	file := assembleFile(pkgName, imports, body.Bytes())
	return emit.WriteGoFile(filepath.Join(outDir, fname), file)
}

// classFuncUsesObjref reports whether a class-method wrapper's body references
// objref — true when any marshaled argument runs through objref.IDOf. The class
// receiver itself dispatches via objc.ID(_class(...)), so unlike an instance
// method the receiver contributes no objref use.
func classFuncUsesObjref(entry methodEntry) bool {
	for _, p := range entry.plainParams {
		if strings.Contains(p.rawExpr, "objref.") {
			return true
		}
	}
	for _, p := range entry.asyncNonBlockParams {
		if strings.Contains(p.rawExpr, "objref.") {
			return true
		}
	}
	return false
}

// classRawMethodNames computes, per class-method selector, the exact Go
// package-level function name the raw emitter produced — the className-prefixed
// name from emit.ClassMethodGoNameFromMeta plus the numeric suffix the raw
// emitter appends to disambiguate colliding names. Mirrors instanceRawMethodNames
// for the class-method namespace, which the raw emitter keeps separate from
// instance methods (the latter are unprefixed), so a class-only pool reproduces
// the raw suffixes exactly.
func classRawMethodNames(
	cls meta.Class,
	className string,
	fw *meta.FrameworkMeta,
) map[string]string {
	count := map[string]int{}
	for _, method := range cls.Methods {
		if !method.IsClassMethod || !emit.MethodWillBeEmitted(method) {
			continue
		}
		count[emit.ClassMethodGoNameFromMeta(className, method.Selector, fw)]++
	}
	seen := map[string]int{}
	selSeen := map[string]bool{}
	out := make(map[string]string, len(count))
	for _, method := range cls.Methods {
		if !method.IsClassMethod || !emit.MethodWillBeEmitted(method) {
			continue
		}
		if selSeen[method.Selector] {
			continue
		}
		selSeen[method.Selector] = true
		name := emit.ClassMethodGoNameFromMeta(className, method.Selector, fw)
		seen[name]++
		if count[name] > 1 && seen[name] > 1 {
			name = fmt.Sprintf("%s%d", name, seen[name])
		}
		out[method.Selector] = name
	}
	return out
}

// classFuncShortName strips the owning class name prefix from a class-method's
// derived Go name to produce a fluent package-level name (for class
// VZBridgedNetworkInterface, "VZBridgedNetworkInterfaceNetworkInterfaces" →
// "NetworkInterfaces"). Returns "" when there is no prefix to strip or the
// result would be unexported/start with a digit, signalling the caller to use a
// qualified fallback.
func classFuncShortName(goName, className string) string {
	if !strings.HasPrefix(goName, className) {
		return ""
	}
	s := goName[len(className):]
	if s == "" {
		return ""
	}
	if c := s[0]; c < 'A' || c > 'Z' {
		return ""
	}
	return s
}

// isFreeFuncName reports whether name is a usable, unclaimed package-level
// identifier (not already emitted/reserved and not hand-authored).
func isFreeFuncName(name string, takenNames, handFuncs map[string]bool) bool {
	return name != "" && !takenNames[name] && !handFuncs[name]
}

// CF parameter classification for emitCFFunctionWrappers.
const (
	cfNone      = iota // not a CoreFoundation reference
	cfInputRef         // a CF…Ref / CFTypeRef value passed in
	cfOutputRef        // a CF…Ref * / CFTypeRef * out-parameter
)

// isOSStatusType reports whether an ObjC return type is a plain OSStatus result
// code (not a pointer to one).
func isOSStatusType(objcType string) bool {
	return strings.Contains(objcType, "OSStatus") && !strings.Contains(objcType, "*")
}

// cfParamKind classifies a parameter's ObjC type for CF bridging: an input CF
// reference value, a CF reference out-parameter (pointer), or neither.
func cfParamKind(objcType string) int {
	hasCFRef := false
	for _, tok := range strings.Fields(objcType) {
		if strings.HasPrefix(tok, "CF") && strings.HasSuffix(tok, "Ref") && len(tok) > 4 {
			hasCFRef = true
		}
	}
	if !hasCFRef {
		return cfNone
	}
	if strings.Contains(objcType, "*") {
		return cfOutputRef
	}
	return cfInputRef
}

// emitCFFunctionWrappers writes <pkgname>_cffunctions_generated.go: idiomatic
// wrappers for OSStatus-returning C functions that take or return CoreFoundation
// references. A CFTypeRef and an objc.ID are the same pointer, so CFDictionaryRef
// / CF…Ref inputs become objc.ID (passed through purego.CFRef), CFTypeRef * out-
// parameters become objc.ID returns, and the OSStatus result becomes a Go error
// (purego.OSStatus.Err, which carries the code so callers can tell e.g.
// errSecItemNotFound from a real failure):
//
//	func SecItemCopyMatching(query objc.ID) (objc.ID, error)
//	func SecItemDelete(query objc.ID) error
//
// This is what lets the keychain be written with no raw FFI. Functions claimed
// here are reserved in takenNames so the generic C-function pass skips them.
func emitCFFunctionWrappers(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	fc *frameworkContext,
	m *typemap.Mapper,
	trialNames trialNameMap,
	takenNames map[string]bool,
) error {
	_ = rawPkgPath
	fw := fc.fw
	ctx := typemap.Context{Framework: fw.Framework}
	// Every wrapper binds the symbol (ebipurego) and maps the OSStatus result to a
	// Go error (purego.NewOSStatus). Reference parameters and out-parameters add
	// obj/objref/objc/unsafe as they are emitted; other parameters' type imports
	// come from the type mapper.
	imports := map[string]string{
		"ebipurego": ebipuregoImportPath,
		"purego":    pureobjcImportPath,
	}
	var funcs []view.Func

	for _, fn := range emit.EmittableFunctions(fw, nil) {
		if !isOSStatusType(fn.Return.ObjCType) {
			continue
		}
		goName := naming.ExportedFunctionName(fn.Name)
		if goName == "" || takenNames[goName] {
			continue
		}

		var sigParams, abiParts, callArgs, preLines, outTypes, outReturns, zeros []string
		fnImports := map[string]string{}
		usedNames := map[string]int{}
		outIdx := 0
		ok := true
		for _, p := range fn.Params {
			pName := safeParamName(naming.ParamName(p.Name))
			if pName == "" {
				pName = fmt.Sprintf("arg%d", len(callArgs))
			}
			usedNames[pName]++
			if usedNames[pName] > 1 {
				pName = fmt.Sprintf("%s%d", pName, usedNames[pName])
			}
			switch cfParamKind(p.ObjCType) {
			case cfInputRef:
				sigParams = append(sigParams, pName+" obj.Object")
				callArgs = append(callArgs, "objref.IDOf("+pName+")")
				abiParts = append(abiParts, "objc.ID")
				fnImports["obj"] = objImportPath
				fnImports["objref"] = objrefImportPath
				fnImports["objc"] = objcImportPath
			case cfOutputRef:
				outVar := fmt.Sprintf("_out%d", outIdx)
				outIdx++
				preLines = append(preLines, "var "+outVar+" uintptr")
				callArgs = append(callArgs, "unsafe.Pointer(&"+outVar+")")
				abiParts = append(abiParts, "unsafe.Pointer")
				outTypes = append(outTypes, "obj.Object")
				fnImports["unsafe"] = "unsafe"
				fnImports["obj"] = objImportPath
				fnImports["objc"] = objcImportPath
				outReturns = append(outReturns, "obj.Wrap(objc.ID("+outVar+"))")
				zeros = append(zeros, "nil")
			default:
				sig, argExpr, imps, pok := idiomaticArg(
					pName,
					p.ObjCType,
					ctx,
					m,
					fc,
					rawPkgAlias,
					trialNames,
				)
				if !pok {
					ok = false
				} else {
					maps.Copy(fnImports, imps)
					sigParams = append(sigParams, pName+" "+sig)
					callArgs = append(callArgs, argExpr)
					abiParts = append(abiParts, cfuncABIType(sig, argExpr))
				}
			}
			if !ok {
				break
			}
		}
		if !ok {
			m.AppendDiagnostic(
				"%s: idiomatic wrapper for %s left out (a parameter type is not yet expressible)",
				fw.Framework,
				fn.Name,
			)
			continue
		}

		takenNames[goName] = true
		// The wrapper is committed: merge its type imports, and add objc when a
		// non-reference parameter also crosses the ABI as an objc.ID.
		maps.Copy(imports, fnImports)
		if slices.Contains(abiParts, "objc.ID") {
			imports["objc"] = objcImportPath
		}
		varName := "_fn" + goName
		retSig := "error"
		if len(outTypes) > 0 {
			retSig = "(" + strings.Join(
				append(append([]string{}, outTypes...), "error"),
				", ",
			) + ")"
		}
		failRet := "_err"
		okRet := "nil"
		if len(outReturns) > 0 {
			failRet = strings.Join(zeros, ", ") + ", _err"
			okRet = strings.Join(outReturns, ", ") + ", nil"
		}

		funcs = append(funcs, view.Func{
			GoName:  goName,
			CName:   fn.Name,
			VarName: varName,
			CommentLine: fmt.Sprintf(
				"// %s reports an error if the %s framework function %s fails.\n",
				goName,
				fw.Framework,
				fn.Name,
			),
			ABIParams: abiParts,
			ABIRet:    "int32",
			SigParams: sigParams,
			RetSig:    " " + retSig,
			Kind:      view.FuncOSStatus,
			PreLines:  preLines,
			Call:      fmt.Sprintf("%s(%s)", varName, strings.Join(callArgs, ", ")),
			FailRet:   failRet,
			OkRet:     okRet,
		})
	}

	if len(funcs) == 0 {
		return nil
	}
	body, err := render.Funcs(funcs)
	if err != nil {
		return err
	}

	fname := pkgName + "_cffunctions_generated.go"
	return emit.WriteGoFile(
		filepath.Join(outDir, fname),
		assembleFile(pkgName, imports, body),
	)
}
