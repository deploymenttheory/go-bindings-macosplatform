//go:build darwin

package idiomatic

import (
	"bytes"
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit"
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
	fw *meta.FrameworkMeta,
	m *typemap.Mapper,
	trialNames trialNameMap,
	takenNames map[string]bool,
) error {
	_ = rawPkgAlias
	_ = rawPkgPath
	ctx := typemap.Context{Framework: fw.Framework}
	prefix := naming.GoTypeName(strings.ToLower(fw.Framework))

	candidates := map[string]string{
		"ebipurego": ebipuregoImportPath,
		"objc":      objcImportPath,
		"purego":    pureobjcImportPath,
		"obj":       objImportPath,
		"objref":    objrefImportPath,
		"rt":        rtImportPath,
	}

	var body bytes.Buffer
	for _, fn := range emit.EmittableFunctions(fw, nil) {
		// Classify every parameter and the return. If any cannot yet be expressed
		// idiomatically (a function pointer, a varargs, an out-parameter), the
		// whole function is left out and a diagnostic is recorded.
		var sigParts, abiParts, callArgs []string
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
			sig, argExpr, imps, pok := idiomaticArg(pName, param.ObjCType, ctx, m, fw, rawPkgAlias, trialNames)
			if !pok {
				ok = false
				break
			}
			maps.Copy(candidates, imps)
			sigParts = append(sigParts, pName+" "+sig)
			abiParts = append(abiParts, cfuncABIType(sig, argExpr))
			callArgs = append(callArgs, argExpr)
		}
		if !ok {
			m.AppendDiagnostic("%s: idiomatic wrapper for %s left out (a parameter type is not yet expressible)", fw.Framework, fn.Name)
			continue
		}
		retType, kind, wrap, _, rimps, rok := idiomaticRet(fn.Return.ObjCType, ctx, m, fw, rawPkgAlias, trialNames)
		if !rok {
			m.AppendDiagnostic("%s: idiomatic wrapper for %s left out (the return type is not yet expressible)", fw.Framework, fn.Name)
			continue
		}
		maps.Copy(candidates, rimps)

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
		fmt.Fprintf(&body, "var %s func(%s) %s\n\n", varName, strings.Join(abiParts, ", "), cfuncRetABI(kind, retType))
		fmt.Fprintf(&body, "// %s calls the %s framework function %s.\n", goName, fw.Framework, fn.Name)
		fmt.Fprintf(&body, "func %s(%s)%s {\n", goName, strings.Join(sigParts, ", "), retSig)
		body.WriteString("_loadOnce.Do(_loadLibrary)\n")
		fmt.Fprintf(&body, "if %s == nil {\nebipurego.RegisterLibFunc(&%s, _lib, %q)\n}\n", varName, varName, fn.Name)
		call := fmt.Sprintf("%s(%s)", varName, strings.Join(callArgs, ", "))
		switch kind {
		case kindVoid:
			fmt.Fprintf(&body, "%s\n", call)
		case kindString:
			fmt.Fprintf(&body, "_ret := %s\nif _ret == 0 {\nreturn \"\"\n}\nreturn purego.GoString(_ret)\n", call)
		case kindObject, kindArray:
			fmt.Fprintf(&body, "_ret := %s\nreturn %s\n", call, strings.Replace(wrap, "%s", "_ret", 1))
		default:
			fmt.Fprintf(&body, "return %s\n", call)
		}
		body.WriteString("}\n\n")
	}

	if body.Len() == 0 {
		return nil
	}

	fname := pkgName + "_cfunctions_generated.go"
	return emit.WriteGoFile(
		filepath.Join(outDir, fname),
		assembleFile(pkgName, usedImports(body.Bytes(), candidates), body.Bytes()),
	)
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
	fw *meta.FrameworkMeta,
	m *typemap.Mapper,
	trialNames trialNameMap,
	handFuncs map[string]bool,
	takenNames map[string]bool,
	abstractBases abstractBaseIndex,
) error {
	candidates := map[string]string{
		rawPkgAlias:  rawPkgPath,
		"context":    "context",
		"unsafe":     "unsafe",
		"purego":     pureobjcImportPath,
		"foundation": foundationImportPath,
		"objc":       objcImportPath,
		"objref":     objrefImportPath,
		"obj":        objImportPath,
		"rt":         rtImportPath,
		"errkit":     errkitImportPath,
	}

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
				fw,
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

			maps.Copy(candidates, entry.extraImports)
			writeClassFunc(&body, fmt.Sprintf("objc.ID(_class(%q))", className), *entry)
		}
	}

	if body.Len() == 0 {
		return nil
	}

	fname := pkgName + "_classmethods_generated.go"
	file := assembleFile(pkgName, usedImports(body.Bytes(), candidates), body.Bytes())
	return emit.WriteGoFile(filepath.Join(outDir, fname), file)
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
	fw *meta.FrameworkMeta,
	m *typemap.Mapper,
	trialNames trialNameMap,
	takenNames map[string]bool,
) error {
	_ = rawPkgPath
	ctx := typemap.Context{Framework: fw.Framework}
	candidates := map[string]string{
		"ebipurego": ebipuregoImportPath,
		"unsafe":    "unsafe",
		"objc":      objcImportPath,
		"purego":    pureobjcImportPath,
		"obj":       objImportPath,
		"objref":    objrefImportPath,
		"rt":        rtImportPath,
	}
	var body bytes.Buffer

	for _, fn := range emit.EmittableFunctions(fw, nil) {
		if !isOSStatusType(fn.Return.ObjCType) {
			continue
		}
		goName := naming.ExportedFunctionName(fn.Name)
		if goName == "" || takenNames[goName] {
			continue
		}

		var sigParams, abiParts, callArgs, preLines, outTypes, outReturns, zeros []string
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
			case cfOutputRef:
				outVar := fmt.Sprintf("_out%d", outIdx)
				outIdx++
				preLines = append(preLines, "var "+outVar+" uintptr")
				callArgs = append(callArgs, "unsafe.Pointer(&"+outVar+")")
				abiParts = append(abiParts, "unsafe.Pointer")
				outTypes = append(outTypes, "obj.Object")
				outReturns = append(outReturns, "obj.Wrap(objc.ID("+outVar+"))")
				zeros = append(zeros, "nil")
			default:
				sig, argExpr, imps, pok := idiomaticArg(pName, p.ObjCType, ctx, m, fw, rawPkgAlias, trialNames)
				if !pok {
					ok = false
				} else {
					maps.Copy(candidates, imps)
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
			m.AppendDiagnostic("%s: idiomatic wrapper for %s left out (a parameter type is not yet expressible)", fw.Framework, fn.Name)
			continue
		}

		takenNames[goName] = true
		varName := "_fn" + goName
		retSig := "error"
		if len(outTypes) > 0 {
			retSig = "(" + strings.Join(append(append([]string{}, outTypes...), "error"), ", ") + ")"
		}
		failRet := "_err"
		okRet := "nil"
		if len(outReturns) > 0 {
			failRet = strings.Join(zeros, ", ") + ", _err"
			okRet = strings.Join(outReturns, ", ") + ", nil"
		}

		fmt.Fprintf(&body, "var %s func(%s) int32\n\n", varName, strings.Join(abiParts, ", "))
		fmt.Fprintf(&body, "// %s reports an error if the %s framework function %s fails.\n", goName, fw.Framework, fn.Name)
		fmt.Fprintf(&body, "func %s(%s) %s {\n", goName, strings.Join(sigParams, ", "), retSig)
		body.WriteString("_loadOnce.Do(_loadLibrary)\n")
		fmt.Fprintf(&body, "if %s == nil {\nebipurego.RegisterLibFunc(&%s, _lib, %q)\n}\n", varName, varName, fn.Name)
		for _, pl := range preLines {
			body.WriteString(pl + "\n")
		}
		fmt.Fprintf(&body, "_rc := %s(%s)\n", varName, strings.Join(callArgs, ", "))
		fmt.Fprintf(&body, "if _err := purego.NewOSStatus(int(_rc)).Err(); _err != nil {\nreturn %s\n}\nreturn %s\n}\n\n", failRet, okRet)
	}

	if body.Len() == 0 {
		return nil
	}

	fname := pkgName + "_cffunctions_generated.go"
	return emit.WriteGoFile(
		filepath.Join(outDir, fname),
		assembleFile(pkgName, usedImports(body.Bytes(), candidates), body.Bytes()),
	)
}

