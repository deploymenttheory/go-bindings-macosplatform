//go:build darwin

package idiomatic

import (
	"fmt"
	"io"
	"maps"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/idiomatic/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/idiomatic/view"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// buildMethods assembles the instance method entries for a class: the special
// shapes (async, slice getters, BoolNSError) plus a plain pass-through wrapper
// for every other bridgeable instance method (including property accessors,
// which Clang synthesises into cls.Methods). Deduplicated by Go method name.
func buildMethods(
	cls meta.Class,
	fc *frameworkContext,
	ctx typemap.Context,
	m *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	abstractBases abstractBaseIndex,
) []methodEntry {
	rawNames := instanceRawMethodNames(cls)
	var methods []methodEntry
	seenMethod := map[string]bool{}
	for _, method := range cls.Methods {
		if method.IsClassMethod || method.IsInit || !instanceBridgeable(method) {
			continue
		}
		rawGoName := rawNames[method.Selector]
		if rawGoName == "" {
			rawGoName = naming.MethodName(method.Selector)
		}
		e := buildMethod(method, rawGoName, cls, fc, ctx, m, rawPkgAlias, trialNames, abstractBases)
		if e == nil {
			continue
		}
		// Re-case recognised initialisms in the exported name (UsbControllers →
		// USBControllers). Done before the dedup check so collisions are detected
		// on the final name. The bridge dispatches by selector, so this is a
		// Go-facing rename only.
		e.goName = applyInitialisms(e.goName)
		if seenMethod[e.goName] || isReservedMemberName(e.goName) {
			continue
		}
		e.doc = method.Doc
		seenMethod[e.goName] = true
		methods = append(methods, *e)
	}
	stripGetterPrefixes(methods)
	return methods
}

// stripGetterPrefixes renames pure accessors so they drop a leading "Get",
// following Effective Go's getter convention (a getter for x is named X, not
// GetX). It runs as a second pass over the fully de-duplicated method set so the
// complete set of names is known: a strip is applied only when the shortened name
// is a valid exported identifier that is not already taken by another method or
// reserved for a promoted root method, which guarantees the rename never collides
// or displaces a genuine method. Buffer-filling Get methods take output
// parameters, so plainDocKind classifies them as actions, not getters, and they
// keep their name. The wrapper dispatches by Objective-C selector, so this is a
// Go-facing rename only.
func stripGetterPrefixes(methods []methodEntry) {
	taken := make(map[string]bool, len(methods))
	for _, me := range methods {
		taken[me.goName] = true
	}
	for i := range methods {
		stripped, ok := strippedGetterName(methods[i])
		if !ok || taken[stripped] || isReservedMemberName(stripped) {
			continue
		}
		taken[stripped] = true
		methods[i].goName = stripped
	}
}

// strippedGetterName returns the method's name with a leading "Get" removed when
// the method is a pure accessor — the no-argument, non-error, value-returning
// shape plainDocKind treats as a getter. The remainder must still be an exported
// identifier (begin with an upper-case ASCII letter), otherwise the original name
// is kept. Plain string operations only — no regular expressions.
func strippedGetterName(me methodEntry) (string, bool) {
	if me.kind != kindPlain {
		return "", false
	}
	if k := plainDocKind(me); k != docGetter && k != docBoolGetter {
		return "", false
	}
	rest, ok := strings.CutPrefix(me.goName, "Get")
	if !ok || rest == "" {
		return "", false
	}
	if c := rest[0]; c < 'A' || c > 'Z' {
		return "", false
	}
	return rest, true
}

// instanceBridgeable mirrors the raw emitter's isMethodBridgeable for instance
// methods: skip unavailable methods and non-format variadics (which the raw
// layer does not emit, so a wrapper calling them would not compile).
func instanceBridgeable(method meta.Method) bool {
	if method.Availability.IsUnavailable {
		return false
	}
	if method.IsVariadic && !isSelectorFormatVariadic(method.Selector) {
		return false
	}
	return true
}

func isSelectorFormatVariadic(selector string) bool {
	for part := range strings.SplitSeq(selector, ":") {
		if strings.Contains(strings.ToLower(part), "format") {
			return true
		}
	}
	return false
}

// instanceRawMethodNames computes, per instance-method selector, the exact Go
// method name the raw emitter produced — including the numeric suffix the raw
// emitter appends to disambiguate colliding names (Foo, Foo2, …). The wrapper
// must call x.inner.<that name>. Class methods are name-prefixed by the raw
// emitter and never collide with instance names, so they are excluded here.
func instanceRawMethodNames(cls meta.Class) map[string]string {
	count := map[string]int{}
	for _, method := range cls.Methods {
		if method.IsClassMethod || !instanceBridgeable(method) {
			continue
		}
		count[naming.MethodName(method.Selector)]++
	}
	seen := map[string]int{}
	selSeen := map[string]bool{}
	out := make(map[string]string, len(count))
	for _, method := range cls.Methods {
		if method.IsClassMethod || !instanceBridgeable(method) {
			continue
		}
		if selSeen[method.Selector] {
			continue
		}
		selSeen[method.Selector] = true
		name := naming.MethodName(method.Selector)
		seen[name]++
		if count[name] > 1 && seen[name] > 1 {
			name = fmt.Sprintf("%s%d", name, seen[name])
		}
		out[method.Selector] = name
	}
	return out
}

type methodKind int

// plainRetMode controls how a plain pass-through method converts the result of
// the underlying raw call at the Go boundary.
type plainRetMode int

// plainParam is one parameter of a plain pass-through method: the Go signature
// type the caller supplies, and the expression handed to the raw method.
type plainParam struct {
	goName  string
	goType  string
	rawExpr string
	// isOut marks a value out-parameter (a pointer the Objective-C call fills
	// in). Such a parameter is lifted out of the Go signature and surfaced as an
	// extra return value, the idiomatic Go translation of an out-parameter.
	isOut bool
	// outName is the godoc-friendly name for the lifted return value (E7), kept
	// distinct from goName, which is the internal _outN local the call writes to.
	outName string
}

type methodEntry struct {
	kind         methodKind
	goName       string
	rawGoName    string
	doc          string // Apple/header documentation for the underlying method
	extraImports map[string]string

	blockObjCParams     []string
	blockGoParamTypes   []string // Go types of the raw completion block's params
	asyncNonBlockParams []asyncParam
	// Typed-result completion: when the block carries exactly one non-error
	// param, the wrapper returns (R, error) instead of plain error.
	asyncResultIdx    int          // index into blockGoParamTypes of the result param; -1 = error-only
	asyncResultGoType string       // wrapped Go return type R ("" = error-only)
	asyncResultMode   plainRetMode // plainRetRaw | plainRetString | plainRetTrialWrap
	asyncResultTrial  string       // trial type name for plainRetTrialWrap

	sliceElemGoType string
	sliceConvFmt    string // fmt template (one %s) converting an objc.ID element to sliceElemGoType
	sliceHasError   bool   // raw getter returns (NSArray, error)

	plainParams    []plainParam
	plainRetType   string // Go value return type ("" = void); error is added separately
	plainRetMode   plainRetMode
	plainTrialType string // trial type name for plainRetTrialWrap
	plainHasError  bool   // raw method returns a trailing error (IsNSError)

	// selector is the Objective-C selector this method calls, e.g.
	// "objectAtIndex:". The generated body sends it directly to the object.
	selector string
	// plainRetKind says how the call's result is converted back to Go (string,
	// object, scalar, …); plainRetWrap is the expression that wraps an object
	// result, with one %s placeholder for the result pointer; plainSendType is
	// the Go type the underlying call returns before conversion.
	plainRetKind  objKind
	plainRetWrap  string
	plainSendType string

	// preamble is rendered verbatim at the top of the method body, before the
	// call — used, for example, to build an Objective-C array from a variadic
	// Go parameter.
	preamble string

	// indexGuardSize, when set, is the name of the size accessor (e.g. "Count"
	// or "Length") used to bounds-check the single index parameter before the
	// call, so an out-of-range index panics in Go instead of raising an
	// uncatchable Objective-C exception.
	indexGuardSize string
}

type asyncParam struct {
	goName          string
	goType          string
	rawExpr         string
	needsFoundation bool
}

// buildAsyncMethod assembles the entry for a completion-handler method. When the
// block only reports an optional NSError the wrapper returns error; when it also
// carries exactly one non-error result the wrapper returns (R, error), where R is
// the same-package trial type (when one exists), a Go string (for NSString), or
// the raw pointer. The generated wrapper blocks on a channel until completion or
// ctx cancellation.
func buildAsyncMethod(
	method meta.Method,
	rawGoName string,
	fc *frameworkContext,
	ctx typemap.Context,
	m *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
) *methodEntry {
	lastParam := method.Params[len(method.Params)-1]
	blockParams := parseBlockObjCParams(lastParam.ObjCType)
	// Resolve the block's Go signature exactly as the raw emitter did, so the
	// generated closure matches the raw method's parameter type. Degraded
	// blocks come back as objc.Block and are rejected below.
	blockImpSet := make(typemap.ImportSet)
	blockGoType := emit.BlockGoFuncType(lastParam.ObjCType, ctx, m, blockImpSet, m.OwnerIndex)
	if !strings.HasPrefix(blockGoType, "func(") || !strings.HasSuffix(blockGoType, ")") {
		return nil
	}
	var blockGoParamTypes []string
	if csv := strings.TrimSuffix(
		strings.TrimPrefix(blockGoType, "func("),
		")",
	); strings.TrimSpace(
		csv,
	) != "" {
		for t := range strings.SplitSeq(csv, ",") {
			blockGoParamTypes = append(
				blockGoParamTypes,
				qualifyRaw(strings.TrimSpace(t), fc, rawPkgAlias, ctx.GenericParams),
			)
		}
	}
	if len(blockGoParamTypes) != len(blockParams) {
		return nil
	}

	// Classify block params into error params and at most one result param.
	resultIdx := -1
	resultMode := plainRetVoid
	resultGoType := ""
	resultTrial := ""
	resultImports := map[string]string{}
	hasErr := false
	for i, bp := range blockParams {
		if isNSErrorType(normaliseObjC(bp)) {
			// The NSError param must be expressible as an objc.ID for conversion.
			t := blockGoParamTypes[i]
			if t != "unsafe.Pointer" && t != "objc.ID" && !strings.HasPrefix(t, "*") {
				return nil
			}
			hasErr = true
			continue
		}
		// A non-error param: we support exactly one, and only as a typed pointer
		// (the raw block adapter already retained + wrapped it). Anything else
		// (a second result, or an objc.ID/unsafe.Pointer degraded param) falls
		// through to the plain wrapper.
		if resultIdx >= 0 {
			return nil
		}
		t := blockGoParamTypes[i]
		if !strings.HasPrefix(t, "*") {
			return nil
		}
		resultIdx = i
		switch t {
		case "*foundation.NSString":
			resultImports["purego"] = pureobjcImportPath
			resultMode, resultGoType = plainRetString, "string"
		default:
			if base, ok := trialWrapClass(t, rawPkgAlias); ok {
				if tt, has := trialNames[base]; has {
					resultMode, resultGoType, resultTrial = plainRetTrialWrap, "*"+tt, tt
				}
			}
			if resultMode != plainRetTrialWrap {
				// Any other object result is surfaced as a generic object.
				resultMode, resultGoType = plainRetRaw, "obj.Object"
			}
		}
	}

	extraImports := map[string]string{
		"context": "context",
		"objc":    objcImportPath,
		"objref":  objrefImportPath,
	}
	// errkit and purego are used only when the completion block carries an
	// NSError (errkit.FromObjC(purego.NSErrorToError(...))); a string result adds
	// purego separately via resultImports.
	if hasErr {
		extraImports["errkit"] = errkitImportPath
		extraImports["purego"] = pureobjcImportPath
	}
	if resultMode == plainRetRaw {
		extraImports["obj"] = objImportPath
	}
	var nonBlockParams []asyncParam
	for _, p := range method.Params[:len(method.Params)-1] {
		pName := safeParamName(naming.ParamName(p.Name))
		if pName == "" {
			pName = fmt.Sprintf("arg%d", len(nonBlockParams))
		}
		sig, argExpr, imps, ok := idiomaticArg(
			pName,
			p.ObjCType,
			ctx,
			m,
			fc,
			rawPkgAlias,
			trialNames,
		)
		if !ok {
			return nil
		}
		maps.Copy(extraImports, imps)
		nonBlockParams = append(
			nonBlockParams,
			asyncParam{goName: pName, goType: sig, rawExpr: argExpr},
		)
	}
	maps.Copy(extraImports, resultImports)
	return &methodEntry{
		kind:                kindAsync,
		goName:              asyncGoMethodName(method.Selector),
		rawGoName:           rawGoName,
		doc:                 method.Doc,
		selector:            method.Selector,
		blockObjCParams:     blockParams,
		blockGoParamTypes:   blockGoParamTypes,
		asyncNonBlockParams: nonBlockParams,
		asyncResultIdx:      resultIdx,
		asyncResultGoType:   resultGoType,
		asyncResultMode:     resultMode,
		asyncResultTrial:    resultTrial,
		extraImports:        extraImports,
	}
}

func buildMethod(
	method meta.Method,
	rawGoName string,
	cls meta.Class,
	fc *frameworkContext,
	ctx typemap.Context,
	m *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	abstractBases abstractBaseIndex,
) *methodEntry {
	// Async completion → (ctx) error. Fall through to a plain wrapper when the
	// block shape can't be expressed idiomatically (non-NSError result params).
	if isAsyncCompletion(method) {
		if e := buildAsyncMethod(method, rawGoName, fc, ctx, m, rawPkgAlias, trialNames); e != nil {
			return e
		}
	}

	// BoolNSError — only when no explicit non-error params.
	if method.IsNSError && strings.TrimSpace(method.Return.ObjCType) == "BOOL" &&
		len(method.Params) == 0 {
		return &methodEntry{
			kind:      kindBoolNSError,
			goName:    boolNSErrorGoMethodName(method.Selector),
			rawGoName: rawGoName,
			selector:  method.Selector,
			extraImports: map[string]string{
				"objc":   objcImportPath,
				"objref": objrefImportPath,
				"unsafe": "unsafe",
				"errkit": errkitImportPath,
				"purego": pureobjcImportPath,
			},
		}
	}

	// NSArray → slice (no params, getter only). Fall through to a plain wrapper
	// when the element type can't be resolved.
	if looksLikeNSArray(method.Return.ObjCType) && len(method.Params) == 0 {
		if e := buildSliceMethod(method, rawGoName, fc, ctx, m, rawPkgAlias, trialNames); e != nil {
			return e
		}
	}

	// Everything else: a plain pass-through wrapper so the method is callable on
	// the idiomatic type without dropping to .Unwrap().
	e := buildPlainMethod(method, rawGoName, fc, ctx, m, rawPkgAlias, trialNames, abstractBases)
	if e != nil {
		if size := indexGuardSizeFor(method, cls); size != "" && len(e.plainParams) == 1 {
			e.indexGuardSize = size
			e.extraImports["errkit"] = errkitImportPath
		}
	}
	return e
}

// indexGuardSizeFor returns the name of the size accessor to bounds-check an
// index accessor's parameter against, or "" when the method is not a recognised
// single-index accessor or the class does not expose the matching size method.
// It lets the generated accessor panic on an out-of-range index instead of
// raising an Objective-C exception that cannot be caught across the boundary.
func indexGuardSizeFor(method meta.Method, cls meta.Class) string {
	var sizeMethod, sizeSelector string
	switch method.Selector {
	case "objectAtIndex:", "objectAtIndexedSubscript:":
		sizeMethod, sizeSelector = "Count", "count"
	case "characterAtIndex:":
		sizeMethod, sizeSelector = "Length", "length"
	default:
		return ""
	}
	for _, classMethod := range cls.Methods {
		if classMethod.Selector == sizeSelector && !classMethod.IsClassMethod {
			return sizeMethod
		}
	}
	return ""
}

// buildSliceMethod builds an NSArray-getter → []T entry, or nil when the element
// type can't be resolved to a typed pointer.
func buildSliceMethod(
	method meta.Method,
	rawGoName string,
	fc *frameworkContext,
	ctx typemap.Context,
	m *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
) *methodEntry {
	elemObjC := extractNSArrayElem(method.Return.ObjCType)
	if elemObjC == "" {
		return nil
	}

	extraImports := map[string]string{
		"purego": pureobjcImportPath,
		"objc":   objcImportPath,
		"objref": objrefImportPath,
	}
	var elemGoType, convFmt string
	switch {
	case isNSStringType(normaliseObjC(elemObjC)):
		// Each element is a string.
		elemGoType, convFmt = "string", "purego.GoString(%s)"
	default:
		impSet := make(typemap.ImportSet)
		goElem := qualifyRaw(m.GoType(elemObjC, ctx, impSet), fc, rawPkgAlias, ctx.GenericParams)
		if !isObjectPointerType(goElem, m) {
			// Only object elements are converted; anything else falls through to a
			// plain method.
			return nil
		}
		if base, ok := trialWrapClass(goElem, rawPkgAlias); ok {
			if tt, has := trialNames[base]; has {
				elemGoType, convFmt = "*"+tt, tt+"FromID(%s)"
			}
		}
		if convFmt == "" {
			elemGoType, convFmt = "obj.Object", "obj.Wrap(%s)"
			extraImports["obj"] = objImportPath
		}
	}

	goName := rawGoName
	if method.IsNSError {
		goName = strings.TrimSuffix(goName, "AndReturnError")
		extraImports["unsafe"] = "unsafe"
		extraImports["errkit"] = errkitImportPath
	}
	return &methodEntry{
		kind:            kindSlice,
		goName:          goName,
		rawGoName:       rawGoName,
		doc:             method.Doc,
		selector:        method.Selector,
		sliceElemGoType: elemGoType,
		sliceConvFmt:    convFmt,
		sliceHasError:   method.IsNSError,
		extraImports:    extraImports,
	}
}

// buildPlainMethod builds a pass-through wrapper for an instance method: it
// converts NSString/NSURL parameters to Go string at the boundary, forwards
// everything else as the raw type, and converts the result back to a Go string
// or a same-package trial type where possible (raw type otherwise). The wrapper
// inherits memory/dispatch correctness from the raw method it calls.
func buildPlainMethod(
	method meta.Method,
	rawGoName string,
	fc *frameworkContext,
	ctx typemap.Context,
	m *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	abstractBases abstractBaseIndex,
) *methodEntry {
	_ = abstractBases
	extraImports := map[string]string{}
	params := make([]plainParam, 0, len(method.Params))
	usedParamNames := map[string]int{}
	outIndex := 0
	for i, p := range method.Params {
		// A value out-parameter is dropped from the signature and surfaced as an
		// extra return value; the call receives a pointer to a local.
		if outGo, isOut := outParamGoType(p.ObjCType, ctx, m, fc, rawPkgAlias); isOut {
			local := fmt.Sprintf("_out%d", outIndex)
			outIndex++
			outName := safeParamName(naming.ParamName(p.Name))
			if outName == "" {
				outName = fmt.Sprintf("out%d", outIndex)
			}
			params = append(params, plainParam{
				goName:  local,
				goType:  outGo,
				rawExpr: "unsafe.Pointer(&" + local + ")",
				isOut:   true,
				outName: outName,
			})
			extraImports["unsafe"] = "unsafe"
			continue
		}
		pName := safeParamName(naming.ParamName(p.Name))
		if pName == "" {
			pName = fmt.Sprintf("arg%d", i)
		}
		if pName == "x" { // collides with the receiver variable
			pName = "x_"
		}
		// Disambiguate duplicate parameter names (e.g. two "recordId" labels):
		// the first keeps the name, later ones get a numeric suffix.
		usedParamNames[pName]++
		if usedParamNames[pName] > 1 {
			pName = fmt.Sprintf("%s%d", pName, usedParamNames[pName])
		}
		sig, argExpr, imps, ok := idiomaticArg(
			pName,
			p.ObjCType,
			ctx,
			m,
			fc,
			rawPkgAlias,
			trialNames,
		)
		if !ok {
			// A parameter type that cannot yet be expressed without naming an
			// Objective-C runtime type; skip the whole method rather than leak one.
			return nil
		}
		maps.Copy(extraImports, imps)
		params = append(params, plainParam{goName: pName, goType: sig, rawExpr: argExpr})
	}

	retType, retKind, retWrap, sendType, retImps, ok := idiomaticRet(
		method.Return.ObjCType, ctx, m, fc, rawPkgAlias, trialNames)
	if !ok {
		return nil
	}
	maps.Copy(extraImports, retImps)

	goName := rawGoName
	// A method that returns a success flag and an error out-parameter is
	// surfaced as returning only error (nil on success): the flag is redundant.
	if method.IsNSError && retKind == kindBool {
		retType, retKind, sendType = "", kindVoid, "bool"
		retWrap = ""
		if strings.HasSuffix(method.Selector, "error:") {
			goName = strings.TrimSuffix(goName, "Error")
		}
	}

	// The dispatch body always calls into the Objective-C runtime through the
	// object's stored pointer.
	extraImports["objc"] = objcImportPath
	extraImports["objref"] = objrefImportPath
	if method.IsNSError {
		extraImports["unsafe"] = "unsafe"
		extraImports["errkit"] = errkitImportPath
		extraImports["purego"] = pureobjcImportPath
	}

	return &methodEntry{
		kind:          kindPlain,
		goName:        goName,
		rawGoName:     rawGoName,
		doc:           method.Doc,
		selector:      method.Selector,
		extraImports:  extraImports,
		plainParams:   params,
		plainRetType:  retType,
		plainRetKind:  retKind,
		plainRetWrap:  retWrap,
		plainSendType: sendType,
		plainHasError: method.IsNSError,
	}
}

func writeMethod(w io.Writer, typeName string, me methodEntry) {
	recvVar := uniqueReceiver(typeName, methodParamNames(me))
	writeMethodAs(w, fmt.Sprintf("(%s *%s) ", recvVar, typeName), "objref.IDOf("+recvVar+")", me)
}

// methodParamNames returns the Go signature parameter names of a method entry —
// the names a receiver variable must not collide with. Out-parameters are
// excluded (they are return values), slice and bool methods take no parameters,
// and an async method's leading ctx is included.
func methodParamNames(me methodEntry) []string {
	var names []string
	switch me.kind {
	case kindAsync:
		names = append(names, "ctx")
		for _, p := range me.asyncNonBlockParams {
			names = append(names, p.goName)
		}
	case kindPlain:
		for _, p := range me.plainParams {
			if !p.isOut {
				names = append(names, p.goName)
			}
		}
	}
	return names
}

// writeClassFunc renders a class-level (static) method as a package-level
// function. recvExpr is the Objective-C expression for the class object, e.g.
// objc.ID(_class("NSString")).
func writeClassFunc(w io.Writer, recvExpr string, me methodEntry) {
	writeMethodAs(w, "", recvExpr, me)
}

// writeMethodAs renders a method entry as either an instance method (recv
// "(x *Type) ", recvExpr "objref.IDOf(x)") or a package-level function (recv "",
// recvExpr the class object). recvExpr is the object the Objective-C call is
// sent to.
func writeMethodAs(w io.Writer, recv, recvExpr string, me methodEntry) {
	switch me.kind {
	case kindAsync:
		writeAsyncMethod(w, recv, recvExpr, me)
	case kindBoolNSError:
		writeBoolNSErrorMethod(w, recv, recvExpr, me)
	case kindSlice:
		writeSliceMethod(w, recv, recvExpr, me)
	case kindPlain:
		writePlainMethod(w, recv, recvExpr, me)
	}
}

// writePlainMethod renders a Go method (or function) that sends one Objective-C
// message to recvExpr, converting the arguments and result between Go and
// Objective-C.
func writePlainMethod(w io.Writer, recv, recvExpr string, me methodEntry) {
	paramParts := make([]string, 0, len(me.plainParams))
	sendArgs := []string{recvExpr, fmt.Sprintf("objc.RegisterName(%q)", me.selector)}
	for _, p := range me.plainParams {
		// Out-parameters are not part of the Go signature, but their pointer is
		// still passed positionally to the Objective-C call.
		if !p.isOut {
			paramParts = append(paramParts, p.goName+" "+p.goType)
		}
		sendArgs = append(sendArgs, p.rawExpr)
	}
	if me.plainHasError {
		sendArgs = append(sendArgs, "unsafe.Pointer(&_nsErr)")
	}
	// objc.Send[...](...) is an expression (the marshaled call), not a Go
	// declaration; the body statements around it are rendered by the template.
	call := fmt.Sprintf("objc.Send[%s](%s)", me.plainSendType, strings.Join(sendArgs, ", "))

	// recv is "(rv *Type) " for an instance method (empty for a package-level
	// class function); the receiver variable is the identifier after the opening
	// paren.
	recvVar := ""
	if fields := strings.Fields(recv); len(fields) > 0 {
		recvVar = strings.TrimPrefix(fields[0], "(")
	}

	var guards []string
	if me.indexGuardSize != "" && len(me.plainParams) == 1 && recvVar != "" {
		guards = append(guards, fmt.Sprintf("errkit.CheckIndex(%s, %s.%s())",
			me.plainParams[0].goName, recvVar, me.indexGuardSize))
	}

	var outs []view.DispatchOut
	for _, p := range outParams(me) {
		outs = append(outs, view.DispatchOut{
			GoName: p.goName,
			GoType: p.goType,
			Zero:   zeroLiteral(p.goType),
		})
	}

	out, err := render.Method(view.Method{
		DocComment: docLeadKind(me.goName, me.doc, synthFallback(me.goName, plainDocKind(me)), plainDocKind(me)),
		Recv:       recv,
		GoName:     me.goName,
		ParamStr:   strings.Join(paramParts, ", "),
		RetSig:     plainRetSig(me, recvVar),
		Dispatch: view.Dispatch{
			Guards:  guards,
			Call:    call,
			Error:   me.plainHasError,
			RetKind: plainRetKindToView(me.plainRetKind),
			RetWrap: me.plainRetWrap,
			RetZero: zeroValue(me.plainRetKind, me.plainRetType),
			Outs:    outs,
		},
	})
	if err != nil {
		panic(err)
	}
	_, _ = w.Write(out)
}

// plainDocKind classifies a plain pass-through method for documentation: a
// no-argument, non-error call that returns a value is an accessor (a boolean one
// when its result is a bool), so its doc comment gets a "returns"/"reports
// whether" lead; everything else is an action method whose Apple prose is
// already verb-led.
func plainDocKind(me methodEntry) docKind {
	if len(me.plainParams) != 0 || me.plainHasError || me.plainRetType == "" {
		return docMethod
	}
	if me.plainRetKind == kindBool {
		return docBoolGetter
	}
	return docGetter
}

// plainRetKindToView maps the internal objKind result classification to the
// view.RetKind the method template renders. Object and array results share the
// wrap path; scalar, bool, and enum all return the raw value unchanged.
func plainRetKindToView(k objKind) view.RetKind {
	switch k {
	case kindVoid:
		return view.RetVoid
	case kindString:
		return view.RetString
	case kindObject, kindArray:
		return view.RetObject
	default:
		return view.RetScalar
	}
}

// outParams returns the method's value out-parameters, in declaration order.
func outParams(me methodEntry) []plainParam {
	var outs []plainParam
	for _, p := range me.plainParams {
		if p.isOut {
			outs = append(outs, p)
		}
	}
	return outs
}

// plainRetSig is the return clause for a plain method:
// " (value, error)" | " error" | " value" | "". recvVar is the receiver
// variable (empty for a package-level function); named returns are chosen so
// they never collide with it.
func plainRetSig(me methodEntry, recvVar string) string {
	type ret struct{ name, typ string }
	var rets []ret
	hasOut := false
	if me.plainRetType != "" {
		valueName := "result"
		if me.plainRetKind == kindBool {
			valueName = "ok"
		}
		rets = append(rets, ret{valueName, me.plainRetType})
	}
	for _, p := range me.plainParams {
		if p.isOut {
			hasOut = true
			rets = append(rets, ret{p.outName, p.goType})
		}
	}
	if me.plainHasError {
		rets = append(rets, ret{"err", "error"})
	}
	// A single non-out return is unnamed (Effective Go: name only when it
	// clarifies); anything with more than one value, or a lifted out-parameter,
	// uses named returns so the godoc reads as documentation (E7).
	if len(rets) == 0 {
		return ""
	}
	if len(rets) == 1 && !hasOut {
		return " " + rets[0].typ
	}
	taken := map[string]bool{}
	if recvVar != "" {
		taken[recvVar] = true
	}
	for _, p := range me.plainParams {
		if !p.isOut {
			taken[p.goName] = true
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

func writeAsyncMethod(w io.Writer, recv, recvExpr string, me methodEntry) {
	paramParts := make([]string, 0, 1+len(me.asyncNonBlockParams))
	paramParts = append(paramParts, "ctx context.Context")
	sendArgs := []string{recvExpr, fmt.Sprintf("objc.RegisterName(%q)", me.selector)}
	for _, p := range me.asyncNonBlockParams {
		paramParts = append(paramParts, p.goName+" "+p.goType)
		sendArgs = append(sendArgs, p.rawExpr)
	}

	// Each block parameter arrives as an Objective-C object pointer. The first
	// error parameter (if any) is converted to a Go error.
	closureParams := make([]string, len(me.blockGoParamTypes))
	errIdx := -1
	for i := range me.blockGoParamTypes {
		closureParams[i] = fmt.Sprintf("_p%d objc.ID", i)
		if errIdx < 0 && isNSErrorType(normaliseObjC(me.blockObjCParams[i])) {
			errIdx = i
		}
	}
	// errConvExpr is the right-hand side that converts the block's NSError
	// parameter to a Go error (an expression), or "" when there is no error param.
	errConvExpr := ""
	if errIdx >= 0 {
		errConvExpr = fmt.Sprintf("errkit.FromObjC(purego.NSErrorToError(_p%d))", errIdx)
	}
	// sendCall hands the block to the Objective-C method; it references the local
	// _block declared by the template. An expression, not a declaration.
	sendCall := "objc.Send[objc.ID](" + strings.Join(
		append(append([]string{}, sendArgs...), "_block"),
		", ",
	) + ")"

	// Error-only completion: returns plain error.
	if me.asyncResultGoType == "" {
		out, err := render.AsyncMethod(view.AsyncMethod{
			DocComment: docLead(
				me.goName,
				me.doc,
				synthFallback(me.goName, docMethod),
			),
			Recv:          recv,
			GoName:        me.goName,
			ParamStr:      strings.Join(paramParts, ", "),
			HasResult:     false,
			ClosureParams: closureParams,
			SendCall:      sendCall,
			ErrConvExpr:   errConvExpr,
		})
		if err != nil {
			panic(err)
		}
		_, _ = w.Write(out)
		return
	}

	// Result-and-error completion: returns (R, error).
	ri := me.asyncResultIdx
	var resultConvExpr string
	switch me.asyncResultMode {
	case plainRetString:
		resultConvExpr = fmt.Sprintf("purego.GoString(_p%d)", ri)
	case plainRetTrialWrap:
		resultConvExpr = fmt.Sprintf("%sFromID(_p%d)", me.asyncResultTrial, ri)
	default:
		resultConvExpr = fmt.Sprintf("obj.Wrap(_p%d)", ri)
	}
	out, err := render.AsyncMethod(view.AsyncMethod{
		DocComment:     docLead(me.goName, me.doc, synthFallback(me.goName, docMethod)),
		Recv:           recv,
		GoName:         me.goName,
		ParamStr:       strings.Join(paramParts, ", "),
		HasResult:      true,
		ResultGoType:   me.asyncResultGoType,
		ClosureParams:  closureParams,
		SendCall:       sendCall,
		ErrConvExpr:    errConvExpr,
		ResultConvExpr: resultConvExpr,
	})
	if err != nil {
		panic(err)
	}
	_, _ = w.Write(out)
}

func writeBoolNSErrorMethod(w io.Writer, recv, recvExpr string, me methodEntry) {
	out, err := render.BoolNSErrorMethod(view.BoolNSErrorMethod{
		DocComment: docLead(me.goName, me.doc, synthFallback(me.goName, docMethod)),
		Recv:       recv,
		GoName:     me.goName,
		RecvExpr:   recvExpr,
		Selector:   me.selector,
	})
	if err != nil {
		panic(err)
	}
	_, _ = w.Write(out)
}

func writeSliceMethod(w io.Writer, recv, recvExpr string, me methodEntry) {
	// The per-element conversion closure is an expression; the template wraps it
	// with purego.NSArrayToSlice.
	conv := fmt.Sprintf(me.sliceConvFmt, "_id")
	convClosure := fmt.Sprintf("func(_id objc.ID) %s { return %s }", me.sliceElemGoType, conv)
	out, err := render.SliceMethod(view.SliceMethod{
		DocComment:  docLeadKind(me.goName, me.doc, synthFallback(me.goName, docGetter), docGetter),
		Recv:        recv,
		GoName:      me.goName,
		RecvExpr:    recvExpr,
		Selector:    me.selector,
		ElemGoType:  me.sliceElemGoType,
		HasError:    me.sliceHasError,
		ConvClosure: convClosure,
	})
	if err != nil {
		panic(err)
	}
	_, _ = w.Write(out)
}
