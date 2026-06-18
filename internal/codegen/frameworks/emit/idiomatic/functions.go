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
func emitGenericFunctionWrappers(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	fw *meta.FrameworkMeta,
	m *typemap.Mapper,
	takenNames map[string]bool,
) error {
	ctx := typemap.Context{Framework: fw.Framework}
	prefix := naming.GoTypeName(strings.ToLower(fw.Framework))

	type wrapEntry struct {
		goName    string // wrapper name in the idiomatic package
		rawGoName string // exported name in the raw package
		params    []string
		callArgs  []string
		retType   string
	}

	var entries []wrapEntry
	var body bytes.Buffer

	for _, fn := range emit.EmittableFunctions(fw, nil) {
		// CFError out-param functions are wrapped (with error conversion) by
		// emitFunctionWrappers.
		if len(fn.Params) > 0 && isCFErrorOutParam(fn.Params[len(fn.Params)-1].ObjCType) {
			continue
		}

		rawGoName := naming.ExportedFunctionName(fn.Name)
		goName := rawGoName
		if stripped := strings.TrimPrefix(rawGoName, prefix); stripped != rawGoName &&
			stripped != "" && stripped[0] >= 'A' && stripped[0] <= 'Z' {
			goName = stripped
		}
		if takenNames[goName] {
			goName = rawGoName
		}
		if takenNames[goName] {
			m.AppendDiagnostic(
				"%s: idiomatic wrapper for %s skipped (name %s already taken)",
				fw.Framework, fn.Name, goName,
			)
			continue
		}
		takenNames[goName] = true

		// Mirror the raw emitter's signature exactly so the wrapper forwards
		// arguments unchanged.
		var params, callArgs []string
		usedNames := make(map[string]int)
		unexportedRef := false
		for _, param := range fn.Params {
			paramName := naming.ParamName(param.Name)
			usedNames[paramName]++
			if usedNames[paramName] > 1 {
				paramName = fmt.Sprintf("%s%d", paramName, usedNames[paramName])
			}
			impSet := make(typemap.ImportSet)
			goType := rawParamGoType(param.ObjCType, ctx, m, impSet)
			if isUnexportedXPkgType(goType) {
				goType = "unsafe.Pointer"
			}
			goType = qualifyRaw(goType, fw, rawPkgAlias, nil)
			if referencesUnexportedQualified(goType) {
				unexportedRef = true
			}
			params = append(params, paramName+" "+goType)
			callArgs = append(callArgs, paramName)
		}

		retType := ""
		if _, retIsBlock := m.ResolveBlockSignature(fn.Return.ObjCType); retIsBlock {
			retType = "objc.Block"
		} else if fn.Return.ObjCType != "void" && fn.Return.ObjCType != "" {
			impSet := make(typemap.ImportSet)
			retType = m.GoReturnType(fn.Return.ObjCType, ctx, impSet)
			if retType == "" || isUnexportedXPkgType(retType) {
				retType = "unsafe.Pointer"
			}
			retType = qualifyRaw(retType, fw, rawPkgAlias, nil)
			if referencesUnexportedQualified(retType) {
				unexportedRef = true
			}
		}

		// Some raw types stay unexported (underscore-prefixed enums like
		// _CGLError) — they cannot be referenced from another package, so the
		// function cannot be wrapped.
		if unexportedRef {
			delete(takenNames, goName)
			m.AppendDiagnostic(
				"%s: idiomatic wrapper for %s skipped (signature references an unexported raw type)",
				fw.Framework,
				fn.Name,
			)
			continue
		}

		entries = append(entries, wrapEntry{
			goName:    goName,
			rawGoName: rawGoName,
			params:    params,
			callArgs:  callArgs,
			retType:   retType,
		})
	}

	if len(entries) == 0 {
		return nil
	}

	view := cfunctionsView{RawAlias: rawPkgAlias, Entries: make([]cfuncEntry, 0, len(entries))}
	for _, entry := range entries {
		retSig := ""
		if entry.retType != "" {
			retSig = " " + entry.retType
		}
		view.Entries = append(view.Entries, cfuncEntry{
			GoName:    entry.goName,
			RawGoName: entry.rawGoName,
			CName:     cFunctionNameFor(entry.rawGoName, fw),
			Params:    strings.Join(entry.params, ", "),
			CallArgs:  strings.Join(entry.callArgs, ", "),
			RetType:   entry.retType,
			RetSig:    retSig,
		})
	}
	if err := executeTemplate(&body, "cfunctions", view); err != nil {
		return err
	}

	// Render-then-scan imports: only include packages the body references.
	bodyStr := body.String()
	imports := map[string]string{rawPkgAlias: rawPkgPath}
	if strings.Contains(bodyStr, "unsafe.") {
		imports["unsafe"] = "unsafe"
	}
	if referencesPackage(bodyStr, "objc") {
		imports["objc"] = objcImportPath
	}
	if referencesPackage(bodyStr, "foundation") {
		imports["foundation"] = foundationImportPath
	}
	for _, crossPkg := range collectCrossPackageRefs(bodyStr, rawPkgAlias) {
		imports[crossPkg] = strings.TrimSuffix(
			rawPkgPath,
			naming.PackageName(fw.Framework),
		) + crossPkg
	}

	fname := pkgName + "_cfunctions_generated.go"
	return emit.WriteGoFile(filepath.Join(outDir, fname), assembleFile(pkgName, imports, body.Bytes()))
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
) error {
	candidates := map[string]string{
		rawPkgAlias:  rawPkgPath,
		"context":    "context",
		"unsafe":     "unsafe",
		"purego":     pureobjcImportPath,
		"foundation": foundationImportPath,
		"objc":       objcImportPath,
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
			entry := buildMethod(method, rawName, cls, fw, ctx, m, rawPkgAlias, trialNames)
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
					fw.Framework, className, method.Selector, fluent, qualified,
				)
				continue
			}
			takenNames[name] = true
			entry.goName = name

			maps.Copy(candidates, entry.extraImports)
			writeClassFunc(&body, rawPkgAlias, *entry)
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
func classRawMethodNames(cls meta.Class, className string, fw *meta.FrameworkMeta) map[string]string {
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
	if c := s[0]; !(c >= 'A' && c <= 'Z') {
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
	takenNames map[string]bool,
) error {
	ctx := typemap.Context{Framework: fw.Framework}
	var body bytes.Buffer
	view := cffunctionsView{RawAlias: rawPkgAlias}

	for _, fn := range emit.EmittableFunctions(fw, nil) {
		if !isOSStatusType(fn.Return.ObjCType) {
			continue
		}
		goName := naming.ExportedFunctionName(fn.Name)
		if goName == "" || takenNames[goName] {
			continue
		}

		var sigParams, callArgs, preLines, outTypes, outReturns, zeros []string
		usedNames := map[string]int{}
		outIdx := 0
		ok := true
		for _, p := range fn.Params {
			pName := naming.ParamName(p.Name)
			if pName == "" {
				pName = fmt.Sprintf("arg%d", len(callArgs))
			}
			usedNames[pName]++
			if usedNames[pName] > 1 {
				pName = fmt.Sprintf("%s%d", pName, usedNames[pName])
			}
			switch cfParamKind(p.ObjCType) {
			case cfInputRef:
				sigParams = append(sigParams, pName+" objc.ID")
				callArgs = append(callArgs, "purego.CFRef("+pName+")")
			case cfOutputRef:
				outVar := fmt.Sprintf("_out%d", outIdx)
				outIdx++
				preLines = append(preLines, "\tvar "+outVar+" uintptr")
				callArgs = append(callArgs, "unsafe.Pointer(&"+outVar+")")
				outTypes = append(outTypes, "objc.ID")
				outReturns = append(outReturns, "objc.ID("+outVar+")")
				zeros = append(zeros, "0")
			default:
				// Non-CF parameter: keep the raw mapped type. Bail (leave the whole
				// function to the generic pass) if it degrades to a pointer/block we
				// would not improve on.
				impSet := make(typemap.ImportSet)
				goType := qualifyRaw(rawParamGoType(p.ObjCType, ctx, m, impSet), fw, rawPkgAlias, nil)
				if goType == "" || strings.HasPrefix(goType, "func(") || strings.Contains(goType, "unsafe.Pointer") {
					ok = false
				} else {
					sigParams = append(sigParams, pName+" "+goType)
					callArgs = append(callArgs, pName)
				}
			}
			if !ok {
				break
			}
		}
		if !ok {
			continue
		}

		takenNames[goName] = true

		rawCall := fmt.Sprintf("%s.%s(%s)", rawPkgAlias, goName, strings.Join(callArgs, ", "))
		retSig := "error"
		if len(outTypes) > 0 {
			retSig = "(" + strings.Join(append(append([]string{}, outTypes...), "error"), ", ") + ")"
		}

		view.Entries = append(view.Entries, cffuncEntry{
			GoName:     goName,
			SigParams:  strings.Join(sigParams, ", "),
			RetSig:     retSig,
			PreLines:   preLines,
			RawCall:    rawCall,
			HasOut:     len(outReturns) > 0,
			OutReturns: strings.Join(outReturns, ", "),
			Zeros:      strings.Join(zeros, ", "),
		})
	}

	if len(view.Entries) == 0 {
		return nil
	}
	if err := executeTemplate(&body, "cffunctions", view); err != nil {
		return err
	}

	// Render-then-scan imports, mirroring emitGenericFunctionWrappers so non-CF
	// parameter/return types that reference other framework packages (e.g.
	// coreaudiotypes, corefoundation) are imported too.
	bodyStr := body.String()
	imports := map[string]string{
		rawPkgAlias: rawPkgPath,
		"purego":    pureobjcImportPath,
	}
	if strings.Contains(bodyStr, "unsafe.") {
		imports["unsafe"] = "unsafe"
	}
	if referencesPackage(bodyStr, "objc") {
		imports["objc"] = objcImportPath
	}
	if referencesPackage(bodyStr, "foundation") {
		imports["foundation"] = foundationImportPath
	}
	for _, crossPkg := range collectCrossPackageRefs(bodyStr, rawPkgAlias) {
		imports[crossPkg] = strings.TrimSuffix(rawPkgPath, naming.PackageName(fw.Framework)) + crossPkg
	}

	fname := pkgName + "_cffunctions_generated.go"
	return emit.WriteGoFile(filepath.Join(outDir, fname), assembleFile(pkgName, imports, body.Bytes()))
}

// cFunctionNameFor recovers the original C symbol for an exported Go function
// name by scanning the framework's function list. Used only for doc comments.
func cFunctionNameFor(rawGoName string, fw *meta.FrameworkMeta) string {
	for _, fn := range fw.Functions {
		if naming.ExportedFunctionName(fn.Name) == rawGoName {
			return fn.Name
		}
	}
	return rawGoName
}

// isUnexportedXPkgType reports whether a Go type string references an
// unexported identifier from another package (compile error if emitted).
func isUnexportedXPkgType(goType string) bool {
	trimmed := strings.TrimPrefix(goType, "*")
	dotIdx := strings.LastIndex(trimmed, ".")
	if dotIdx < 0 {
		return false
	}
	after := trimmed[dotIdx+1:]
	if brIdx := strings.Index(after, "["); brIdx >= 0 {
		after = after[:brIdx]
	}
	return len(after) > 0 && after[0] >= 'a' && after[0] <= 'z'
}

// referencesUnexportedQualified reports whether a (possibly qualified) Go
// type string contains a package-qualified unexported identifier such as
// raw._CGLError — a compile error when emitted outside the raw package.
func referencesUnexportedQualified(goType string) bool {
	for i := 1; i < len(goType)-1; i++ {
		if goType[i] != '.' || !isIdentByte(goType[i-1], false) {
			continue
		}
		c := goType[i+1]
		if c == '_' || (c >= 'a' && c <= 'z') {
			return true
		}
	}
	return false
}

// referencesPackage reports whether the body references pkg as a package
// qualifier ("pkg.") at a non-identifier boundary, avoiding false hits on
// names like "purego." when checking for "objc.".
func referencesPackage(body, pkg string) bool {
	needle := pkg + "."
	for idx := strings.Index(body, needle); idx >= 0; {
		if idx == 0 || !isIdentByte(body[idx-1], false) {
			return true
		}
		next := strings.Index(body[idx+1:], needle)
		if next < 0 {
			return false
		}
		idx += 1 + next
	}
	return false
}

// collectCrossPackageRefs finds cross-framework package qualifiers used in the
// body (lowercase identifiers followed by '.' and an exported identifier),
// excluding the raw alias and well-known packages handled explicitly.
func collectCrossPackageRefs(body, rawPkgAlias string) []string {
	known := map[string]bool{
		rawPkgAlias: true, "unsafe": true, "objc": true, "foundation": true,
		"fmt": true, "strings": true, "purego": true,
	}
	var refs []string
	seen := map[string]bool{}
	for i := 0; i < len(body); {
		if !isIdentByte(body[i], true) {
			i++
			continue
		}
		j := i + 1
		for j < len(body) && isIdentByte(body[j], false) {
			j++
		}
		word := body[i:j]
		prevIsDot := i > 0 && body[i-1] == '.'
		if !prevIsDot && j < len(body)-1 && body[j] == '.' &&
			word == strings.ToLower(word) && !known[word] && !seen[word] &&
			body[j+1] >= 'A' && body[j+1] <= 'Z' {
			seen[word] = true
			refs = append(refs, word)
		}
		i = j
	}
	return refs
}

// cfuncEntry / cfunctionsView are the template data for cfunctions.tmpl. The
// signature/argument strings are pre-joined by emitGenericFunctionWrappers.
type cfuncEntry struct {
	GoName    string
	RawGoName string
	CName     string
	Params    string // "name type, …"
	CallArgs  string // "name, …"
	RetType   string // "" for a void function
	RetSig    string // " <retType>" or ""
}

type cfunctionsView struct {
	RawAlias string
	Entries  []cfuncEntry
}

// cffuncEntry / cffunctionsView are the template data for cffunctions.tmpl (the
// OSStatus + CoreFoundation-reference bridging wrappers).
type cffuncEntry struct {
	GoName     string
	SigParams  string   // "name type, …"
	RetSig     string   // "error" or "(objc.ID, …, error)"
	PreLines   []string // per-out-param "var _outN uintptr" lines
	RawCall    string   // raw.Foo(args)
	HasOut     bool     // has CF out-parameters
	OutReturns string   // "objc.ID(_out0), …"
	Zeros      string   // "0, …" (zero values for the error path)
}

type cffunctionsView struct {
	RawAlias string
	Entries  []cffuncEntry
}
