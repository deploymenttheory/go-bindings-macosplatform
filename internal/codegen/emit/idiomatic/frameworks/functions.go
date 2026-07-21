//go:build darwin

package idiofw

import (
	"bytes"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/view"
	rawfw "github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emitmanifest"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// recordIdiomaticFunction records one emitted free-function wrapper into the
// parity manifest, keyed on the C function name (identical to the raw oracle's
// key). The idiomatic free-function emitters are split across three passes that
// coordinate via takenNames, so a given C function is committed by exactly one
// of them; Compare deduplicates in any case. No-op on a nil recorder.
func recordIdiomaticFunction(rec *emitmanifest.Recorder, framework, pkgName, cName, goName string) {
	if rec == nil {
		return
	}
	rec.Record(emitmanifest.Entry{
		Style:     emitmanifest.StyleIdiomatic,
		Kind:      emitmanifest.KindFunction,
		Framework: framework,
		MetaKey:   emitmanifest.MetaKey(framework, emitmanifest.KindFunction, cName, ""),
		GoPkg:     pkgName,
		GoSymbol:  goName,
	})
}

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
// (an object, an NSString, a URL, a date, a data buffer, or a collection), the C
// function receives an objc.ID; otherwise the value — a C string, a number, an
// enum, or a value struct — is passed unchanged.
func cfuncABIType(sig, argExpr string) string {
	if strings.HasPrefix(argExpr, "objc.NewBlock(") {
		return "objc.Block"
	}
	for _, prefix := range []string{
		"purego.NSString(", "objref.IDOf(", "purego.SliceToNSArray(",
		"rt.FileURL(", "rt.TimeToNSDate(", "rt.BytesToNSData(",
		"rt.MapToDict(", "rt.SliceToNSSet(",
	} {
		if strings.HasPrefix(argExpr, prefix) {
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

// buildCFuncRetSig assembles the Go return clause for a C-function wrapper. A
// lone return is unnamed (" int"); a multi-value clause — the function's own
// result plus any lifted out-parameters — is named (" (result int, vcpu uint64,
// exit *HvVcpuExitT)") so the godoc reads as documentation (E7). Names mirror the
// methods path: "result" (or "ok" for bool) for the result, the parameter name
// for each out, de-duplicated against the Go parameter names. It joins resolved
// type strings only — the gather phase decided each type.
func buildCFuncRetSig(
	retType string,
	kind objKind,
	outNames []string,
	outs []view.DispatchOut,
	sigParts []string,
) string {
	type ret struct{ name, typ string }
	var rets []ret
	if retType != "" {
		name := "result"
		if kind == kindBool {
			name = "ok"
		}
		rets = append(rets, ret{name, retType})
	}
	for i, o := range outs {
		rets = append(rets, ret{outNames[i], o.GoType})
	}
	switch {
	case len(rets) == 0:
		return ""
	case len(rets) == 1 && len(outs) == 0:
		return " " + rets[0].typ
	}
	taken := map[string]bool{}
	for _, sp := range sigParts {
		if name, _, ok := strings.Cut(sp, " "); ok {
			taken[name] = true
		}
	}
	parts := make([]string, len(rets))
	for i, r := range rets {
		name := safeReturnName(r.name, taken)
		taken[name] = true
		parts[i] = name + " " + r.typ
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// isArrayPointerParam reports whether the parameter at index i is a C array
// passed as the "(T *items, size_t count)" idiom — a single pointer immediately
// followed by a count parameter. Such a pointer is an INPUT the caller fills, so
// it is passed through (as an unsafe.Pointer) rather than lifted into a return
// value the way a true scalar out-parameter (e.g. uint64_t *value) is. Without
// this, hv_vcpus_exit(hv_vcpu_t *vcpus, uint32_t vcpu_count) would wrongly turn
// its input vcpus array into a returned value.
func isArrayPointerParam(objcType string, params []meta.Param, i int) bool {
	if strings.Contains(objcType, "(^") || strings.Count(normaliseObjC(objcType), "*") != 1 {
		return false
	}
	return i+1 < len(params) && isCountParamName(params[i+1].Name)
}

// isCountParamName reports whether a parameter name denotes an element count or
// byte length, which marks the preceding pointer as an array rather than an
// out-parameter.
func isCountParamName(name string) bool {
	n := strings.ToLower(name)
	switch n {
	case "count", "len", "length", "size", "num", "n":
		return true
	}
	for _, suffix := range []string{"_count", "count", "_len", "_length", "_size", "_num"} {
		if strings.HasSuffix(n, suffix) {
			return true
		}
	}
	return false
}

func emitGenericFunctionWrappers(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	fc *frameworkContext,
	mapper *typemap.Mapper,
	trialNames trialNameMap,
	takenNames map[string]bool,
) error {
	framework := fc.framework
	ctx := typemap.Context{Framework: framework.Framework}
	prefix := naming.GoTypeName(strings.ToLower(framework.Framework))

	// Imports are accumulated from what each wrapper actually emits: ebipurego is
	// always used to bind the symbol, parameter/return type imports come from the
	// type mapper, and objc is added below when a parameter or the return crosses
	// the ABI as an objc.ID.
	imports := map[string]string{"ebipurego": ebipuregoImportPath}

	var funcs []view.Func
	for _, fn := range rawfw.EmittableFunctions(framework, nil) {
		// Classify every parameter and the return. If any cannot yet be expressed
		// idiomatically (a function pointer, a varargs, an out-parameter), the
		// whole function is left out and a diagnostic is recorded. Type imports are
		// gathered per function and merged only once the wrapper is committed, so a
		// skipped function never contributes an unused import.
		var sigParts, abiParts, callArgs []string
		var outs []view.DispatchOut
		var outNames []string // readable return names for the signature (E7)
		fnImports := map[string]string{}
		ok := true
		usedNames := make(map[string]int)
		for i, param := range fn.Params {
			pName := safeParamName(naming.ParamName(param.Name))
			if pName == "" {
				pName = fmt.Sprintf("arg%d", len(sigParts)+len(outs))
			}
			usedNames[pName]++
			if usedNames[pName] > 1 {
				pName = fmt.Sprintf("%s%d", pName, usedNames[pName])
			}
			// A pointer immediately followed by a count parameter is the C
			// "(array, count)" idiom — an INPUT array the caller fills, not an
			// out-parameter. (hv_vcpus_exit(hv_vcpu_t *vcpus, uint32_t vcpu_count)
			// reads vcpus; it must not be lifted to a return like a uint64_t* out.)
			// Pass it through as an unsafe.Pointer.
			if isArrayPointerParam(param.ObjCType, fn.Params, i) {
				sigParts = append(sigParts, pName+" unsafe.Pointer")
				abiParts = append(abiParts, "unsafe.Pointer")
				callArgs = append(callArgs, pName)
				fnImports["unsafe"] = "unsafe"
				continue
			}
			// A pointer out-parameter is lifted into an extra return value: declare a
			// local (named _outN so it never clashes with the readable return name),
			// pass its address to the call, and return it after the result.
			if outGo, isOut := outParamGoType(
				param.ObjCType,
				ctx,
				mapper,
				fc,
				rawPkgAlias,
				true,
			); isOut {
				local := fmt.Sprintf("_out%d", len(outs))
				outs = append(outs, view.DispatchOut{
					GoName: local,
					GoType: outGo,
					Zero:   zeroLiteral(outGo),
				})
				outNames = append(outNames, pName)
				callArgs = append(callArgs, "unsafe.Pointer(&"+local+")")
				abiParts = append(abiParts, "unsafe.Pointer")
				fnImports["unsafe"] = "unsafe"
				if strings.Contains(outGo, rawPkgAlias+".") {
					fnImports[rawPkgAlias] = rawPkgPath
				}
				continue
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
		retType, kind, wrap, _, rimps, rok := idiomaticRet(
			fn.Return.ObjCType,
			ctx,
			mapper,
			fc,
			rawPkgAlias,
			trialNames,
		)
		if !rok {
			mapper.AppendDiagnostic(
				"%s: idiomatic wrapper for %s left out (the return type is not yet expressible)",
				framework.Framework,
				fn.Name,
			)
			continue
		}
		maps.Copy(fnImports, rimps)

		goName := naming.ExportedFunctionName(fn.Name)
		if curated, isCurated := fc.idio.FunctionGoName(fn.Name); isCurated {
			goName = curated
		} else if stripped := strings.TrimPrefix(goName, prefix); stripped != goName &&
			stripped != "" && stripped[0] >= 'A' && stripped[0] <= 'Z' {
			if !takenNames[stripped] {
				goName = stripped
			}
		}
		if goName == "" || takenNames[goName] {
			continue
		}
		takenNames[goName] = true
		recordIdiomaticFunction(fc.manifest, framework.Framework, pkgName, fn.Name, goName)

		varName := "_fn" + goName

		// A return whose typedef is registered in the sidecar's error_typedefs
		// (hv_return_t, kern_return_t, …) is a status code: the wrapper returns a
		// Go error (nil on the success value) instead of a bare integer, with any
		// lifted out-parameters returned on success and zeroed on failure.
		if statusTypedef, isStatus := fc.idio.ErrorTypedefFor(
			strings.TrimSpace(normaliseObjC(fn.Return.ObjCType)),
		); isStatus && (kind == kindScalar || kind == kindEnum) {
			fnImports["errkit"] = errkitImportPath
			maps.Copy(imports, fnImports)
			if slices.Contains(abiParts, "objc.ID") {
				imports["objc"] = objcImportPath
			}
			funcs = append(funcs, buildStatusCodeFunc(statusCodeFuncInput{
				goName:       goName,
				fn:           fn,
				framework:    framework,
				varName:      varName,
				retType:      retType,
				statusDomain: statusTypedef.Domain,
				successValue: statusTypedef.SuccessValue,
				sigParts:     sigParts,
				abiParts:     abiParts,
				callArgs:     callArgs,
				outs:         outs,
				outNames:     outNames,
			}, mapper))
			continue
		}

		retSig := buildCFuncRetSig(retType, kind, outNames, outs, sigParts)
		abiRet := cfuncRetABI(kind, retType)
		// Bind the C function at its C-faithful ABI return width (e.g. int32 for a
		// C int / hv_return_t) so purego marshals the result correctly — the
		// ergonomic public type widens 32-bit ints to Go int, which leaves junk in
		// the upper bits. The wrapper converts the result back: int(_fn(...)).
		rawCall := fmt.Sprintf("%s(%s)", varName, strings.Join(callArgs, ", "))
		call := rawCall
		if genericFuncKind(kind) == view.FuncScalar {
			if abi := mapper.GoABIType(fn.Return.ObjCType, retType); abi != retType {
				abiRet = abi
				call = retType + "(" + rawCall + ")"
			}
		}
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
				framework.Framework,
				fn.Name,
			),
			ABIParams: abiParts,
			ABIRet:    abiRet,
			SigParams: sigParts,
			RetSig:    retSig,
			Kind:      genericFuncKind(kind),
			Call:      call,
			Wrap:      wrap,
			Outs:      outs,
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
	return rawfw.WriteGoFile(
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
// package-level forwarding function per ObjC class (static) method in framework. These
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
	mapper *typemap.Mapper,
	trialNames trialNameMap,
	handFuncs map[string]bool,
	takenNames map[string]bool,
	abstractBases abstractBaseIndex,
) error {
	framework := fc.framework
	// Every wrapper dispatches to a class object (objc.ID(_class(...))), so objc is
	// always used; every other import is contributed by an emitted method's own
	// import set. objref is added per method only when an argument actually marshals
	// through objref.IDOf, since (unlike an instance method) the class receiver does
	// not use it.
	imports := map[string]string{"objc": objcImportPath}

	var body bytes.Buffer
	for _, className := range sortedKeys(framework.Classes) {
		class := framework.Classes[className]
		if class.Availability.IsUnavailable {
			continue
		}
		ctx := typemap.Context{
			ClassName:     className,
			Framework:     framework.Framework,
			GenericParams: class.GenericParams,
		}
		rawNames := classRawMethodNames(class, className, framework)

		seenSel := map[string]bool{}
		for _, method := range class.Methods {
			if !method.IsClassMethod || !rawfw.MethodWillBeEmitted(method) {
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
				class,
				fc,
				ctx,
				mapper,
				rawPkgAlias,
				trialNames,
				abstractBases,
			)
			if entry == nil {
				continue
			}

			var name string
			if curated, isCurated := fc.idio.MethodGoName(
				className,
				method.Selector,
				true,
			); isCurated {
				// A curated sidecar name is used verbatim (it still must be free).
				if !isFreeFuncName(curated, takenNames, handFuncs) {
					mapper.AppendDiagnostic(
						"%s: curated name %s for +[%s %s] already taken; wrapper skipped",
						framework.Framework, curated, className, method.Selector,
					)
					continue
				}
				name = curated
			} else {
				// An error-returning class function drops a redundant trailing
				// error label, mirroring repairMethodNames (the free-name checks
				// below still apply to whichever spelling is chosen).
				if methodReturnsError(*entry) {
					for _, suffix := range []string{"AndReturnError", "WithError", "Error"} {
						if stripped, ok := strings.CutSuffix(entry.goName, suffix); ok {
							if stripped != "" && stripped[0] >= 'A' && stripped[0] <= 'Z' {
								entry.goName = stripped
							}
							break
						}
					}
				}

				// The emitted function name is independent of the raw symbol it
				// calls (entry.rawGoName), so pick the most fluent free name.
				fluent := classFuncShortName(entry.goName, className)
				if fluent == "" {
					fluent = entry.goName
				}
				qualified := className + fluent
				switch {
				case isFreeFuncName(fluent, takenNames, handFuncs):
					name = fluent
				case isFreeFuncName(qualified, takenNames, handFuncs):
					name = qualified
				default:
					mapper.AppendDiagnostic(
						"%s: idiomatic class-method wrapper for +[%s %s] skipped (names %s/%s already taken)",
						framework.Framework,
						className,
						method.Selector,
						fluent,
						qualified,
					)
					continue
				}
			}
			takenNames[name] = true
			entry.goName = name

			fnImps := maps.Clone(entry.extraImports)
			if !classFuncUsesObjref(*entry) {
				// The class receiver is objc.ID(_class(...)), not objref.IDOf(x);
				// objref is needed only when an argument marshals through it.
				delete(fnImps, "objref")
			}
			// A class function has no receiver to keep alive; runtime is needed
			// only when a wrapper argument is (defer runtime.KeepAlive).
			if len(keepAliveNames("", *entry)) == 0 {
				delete(fnImps, "runtime")
			} else {
				fnImps["runtime"] = "runtime"
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
	return rawfw.WriteGoFile(filepath.Join(outDir, fname), file)
}

// classFuncUsesObjref reports whether a class-method wrapper's body references
// objref — true when any marshaled argument runs through objref.IDOf. The class
// receiver itself dispatches via objc.ID(_class(...)), so unlike an instance
// method the receiver contributes no objref use.
func classFuncUsesObjref(entry methodModel) bool {
	for _, param := range entry.plainParams {
		if strings.Contains(param.rawExpression, "objref.") {
			return true
		}
	}
	for _, param := range entry.asyncNonBlockParams {
		if strings.Contains(param.rawExpression, "objref.") {
			return true
		}
	}
	return false
}

// classRawMethodNames computes, per class-method selector, the exact Go
// package-level function name the raw emitter produced — the className-prefixed
// name from rawfw.ClassMethodGoNameFromMeta plus the numeric suffix the raw
// emitter appends to disambiguate colliding names. Mirrors instanceRawMethodNames
// for the class-method namespace, which the raw emitter keeps separate from
// instance methods (the latter are unprefixed), so a class-only pool reproduces
// the raw suffixes exactly.
func classRawMethodNames(
	class meta.Class,
	className string,
	framework *meta.FrameworkMeta,
) map[string]string {
	count := map[string]int{}
	for _, method := range class.Methods {
		if !method.IsClassMethod || !rawfw.MethodWillBeEmitted(method) {
			continue
		}
		count[rawfw.ClassMethodGoNameFromMeta(className, method.Selector, framework)]++
	}
	seen := map[string]int{}
	selSeen := map[string]bool{}
	out := make(map[string]string, len(count))
	for _, method := range class.Methods {
		if !method.IsClassMethod || !rawfw.MethodWillBeEmitted(method) {
			continue
		}
		if selSeen[method.Selector] {
			continue
		}
		selSeen[method.Selector] = true
		name := rawfw.ClassMethodGoNameFromMeta(className, method.Selector, framework)
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
	mapper *typemap.Mapper,
	trialNames trialNameMap,
	takenNames map[string]bool,
) error {
	_ = rawPkgPath
	framework := fc.framework
	ctx := typemap.Context{Framework: framework.Framework}
	// Every wrapper binds the symbol (ebipurego) and maps the OSStatus result to a
	// Go error (purego.NewOSStatus). Reference parameters and out-parameters add
	// obj/objref/objc/unsafe as they are emitted; other parameters' type imports
	// come from the type mapper.
	imports := map[string]string{
		"ebipurego": ebipuregoImportPath,
		"purego":    pureobjcImportPath,
	}
	var funcs []view.Func

	for _, fn := range rawfw.EmittableFunctions(framework, nil) {
		if !isOSStatusType(fn.Return.ObjCType) {
			continue
		}
		goName := naming.ExportedFunctionName(fn.Name)
		if curated, isCurated := fc.idio.FunctionGoName(fn.Name); isCurated {
			goName = curated
		}
		if goName == "" || takenNames[goName] {
			continue
		}

		var sigParams, abiParts, callArgs, preLines, outTypes, outReturns, zeros []string
		fnImports := map[string]string{}
		usedNames := map[string]int{}
		outIdx := 0
		ok := true
		for _, param := range fn.Params {
			pName := safeParamName(naming.ParamName(param.Name))
			if pName == "" {
				pName = fmt.Sprintf("arg%d", len(callArgs))
			}
			usedNames[pName]++
			if usedNames[pName] > 1 {
				pName = fmt.Sprintf("%s%d", pName, usedNames[pName])
			}
			switch cfParamKind(param.ObjCType) {
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
				} else {
					maps.Copy(fnImports, imports)
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
			mapper.AppendDiagnostic(
				"%s: idiomatic wrapper for %s left out (a parameter type is not yet expressible)",
				framework.Framework,
				fn.Name,
			)
			continue
		}

		takenNames[goName] = true
		recordIdiomaticFunction(fc.manifest, framework.Framework, pkgName, fn.Name, goName)
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
				framework.Framework,
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
	return rawfw.WriteGoFile(
		filepath.Join(outDir, fname),
		assembleFile(pkgName, imports, body),
	)
}

// statusCodeFuncInput carries the pieces emitGenericFunctionWrappers has
// already resolved for one status-code-returning C function into
// buildStatusCodeFunc.
type statusCodeFuncInput struct {
	goName       string
	fn           meta.Function
	framework    *meta.FrameworkMeta
	varName      string
	retType      string
	statusDomain string
	successValue int64
	sigParts     []string
	abiParts     []string
	callArgs     []string
	outs         []view.DispatchOut
	outNames     []string
}

// buildStatusCodeFunc assembles the view for a C function whose registered
// status-code return becomes a Go error (nil on the success value): any lifted
// out-parameters are declared as locals, returned on success, and zeroed on
// failure alongside the error.
func buildStatusCodeFunc(in statusCodeFuncInput, mapper *typemap.Mapper) view.Func {
	preLines := make([]string, 0, len(in.outs))
	failParts := make([]string, 0, len(in.outs)+1)
	okParts := make([]string, 0, len(in.outs)+1)
	retParts := make([]string, 0, len(in.outs)+1)
	returnNamesTaken := map[string]bool{}
	for _, sigPart := range in.sigParts {
		if paramName, _, hasSpace := strings.Cut(sigPart, " "); hasSpace {
			returnNamesTaken[paramName] = true
		}
	}
	for i, out := range in.outs {
		preLines = append(preLines, "var "+out.GoName+" "+out.GoType)
		// zeroLiteral's "0" covers scalars and enums; a lifted value-struct
		// out (e.g. HvVcpuSmeStateT) zeroes to its composite literal.
		zero := out.Zero
		if mapper.EmittableStructs[out.GoType] {
			zero = out.GoType + "{}"
		}
		failParts = append(failParts, zero)
		okParts = append(okParts, out.GoName)
		returnName := safeReturnName(in.outNames[i], returnNamesTaken)
		returnNamesTaken[returnName] = true
		retParts = append(retParts, returnName+" "+out.GoType)
	}
	failParts = append(failParts, "_err")
	okParts = append(okParts, "nil")
	retSig := " error"
	if len(retParts) > 0 {
		retSig = " (" + strings.Join(append(retParts, "err error"), ", ") + ")"
	}
	return view.Func{
		GoName:  in.goName,
		CName:   in.fn.Name,
		VarName: in.varName,
		CommentLine: fmt.Sprintf(
			"// %s reports an error if the %s framework function %s fails.\n",
			in.goName,
			in.framework.Framework,
			in.fn.Name,
		),
		ABIParams: in.abiParts,
		ABIRet:    mapper.GoABIType(in.fn.Return.ObjCType, in.retType),
		SigParams: in.sigParts,
		RetSig:    retSig,
		Kind:      view.FuncStatusCode,
		PreLines:  preLines,
		Call:      fmt.Sprintf("%s(%s)", in.varName, strings.Join(in.callArgs, ", ")),
		ErrExpr: fmt.Sprintf(
			"errkit.FromCode(%q, int64(_rc), %d)",
			in.statusDomain,
			in.successValue,
		),
		FailRet: strings.Join(failParts, ", "),
		OkRet:   strings.Join(okParts, ", "),
	}
}
