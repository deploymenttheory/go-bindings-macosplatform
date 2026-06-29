//go:build darwin

package idiofw

import (
	"fmt"
	"io"
	"maps"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/view"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// buildMethods assembles the instance method entries for a class: the special
// shapes (async, slice getters, BoolNSError) plus a plain pass-through wrapper
// for every other bridgeable instance method (including property accessors,
// which Clang synthesises into class.Methods). Deduplicated by Go method name.
func buildMethods(
	class meta.Class,
	fc *frameworkContext,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	abstractBases abstractBaseIndex,
) []methodModel {
	rawNames := instanceRawMethodNames(class)
	var methods []methodModel
	seenMethod := map[string]bool{}
	for _, method := range class.Methods {
		if method.IsClassMethod || method.IsInit || !instanceBridgeable(method) {
			continue
		}
		rawGoName := rawNames[method.Selector]
		if rawGoName == "" {
			rawGoName = naming.MethodName(method.Selector)
		}
		built := buildMethod(method, rawGoName, class, fc, ctx, mapper, rawPkgAlias, trialNames, abstractBases)
		if built == nil {
			continue
		}
		// A call is main-thread-bound when its class (or an ancestor) is
		// @MainActor or the selector itself is, unless it is explicitly
		// `nonisolated`. The loader has already propagated and stamped these.
		built.mainThread = (class.IsMainThreadRequired || method.IsMainThreadRequired) &&
			!method.IsMainThreadExempt
		if built.mainThread {
			built.extraImports["purego"] = pureobjcImportPath
		}
		// Re-case recognised initialisms in the exported name (UsbControllers →
		// USBControllers). Done before the dedup check so collisions are detected
		// on the final name. The bridge dispatches by selector, so this is a
		// Go-facing rename only.
		built.goName = applyInitialisms(built.goName)
		if seenMethod[built.goName] || isReservedMemberName(built.goName) {
			continue
		}
		built.doc = method.Doc
		seenMethod[built.goName] = true
		methods = append(methods, *built)
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
func stripGetterPrefixes(methods []methodModel) {
	taken := make(map[string]bool, len(methods))
	for _, method := range methods {
		taken[method.goName] = true
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
func strippedGetterName(method methodModel) (string, bool) {
	if method.kind != kindPlain {
		return "", false
	}
	if k := plainDocKind(method); k != docGetter && k != docBoolGetter {
		return "", false
	}
	rest, ok := strings.CutPrefix(method.goName, "Get")
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
func instanceRawMethodNames(class meta.Class) map[string]string {
	count := map[string]int{}
	for _, method := range class.Methods {
		if method.IsClassMethod || !instanceBridgeable(method) {
			continue
		}
		count[naming.MethodName(method.Selector)]++
	}
	seen := map[string]int{}
	selSeen := map[string]bool{}
	out := make(map[string]string, len(count))
	for _, method := range class.Methods {
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

// plainParamModel is one parameter of a plain pass-through method: the Go signature
// type the caller supplies, and the expression handed to the raw method.
type plainParamModel struct {
	goName        string
	goType        string
	rawExpression string
	// isOut marks a value out-parameter (a pointer the Objective-C call fills
	// in). Such a parameter is lifted out of the Go signature and surfaced as an
	// extra return value, the idiomatic Go translation of an out-parameter.
	isOut bool
	// outName is the godoc-friendly name for the lifted return value (E7), kept
	// distinct from goName, which is the internal _outN local the call writes to.
	outName string
}

type methodModel struct {
	kind         methodKind
	goName       string
	rawGoName    string
	doc          string // Apple/header documentation for the underlying method
	extraImports map[string]string

	// mainThread is set when the underlying selector is isolated to Swift's
	// @MainActor (the class, an ancestor, or the method itself), so the emitted
	// body must run on the main thread. The writers wrap the call in purego.Main.
	mainThread bool

	blockObjCParams     []string
	blockGoParamTypes   []string // Go types of the raw completion block's params
	asyncNonBlockParams []asyncParamModel
	// Typed-result completion: when the block carries exactly one non-error
	// param, the wrapper returns (R, error) instead of plain error.
	asyncResultIdx    int          // index into blockGoParamTypes of the result param; -1 = error-only
	asyncResultGoType string       // wrapped Go return type R ("" = error-only)
	asyncResultMode   plainRetMode // plainRetRaw | plainRetString | plainRetTrialWrap
	asyncResultTrial  string       // trial type name for plainRetTrialWrap

	sliceElemGoType              string
	sliceElementConversionFormat string // fmt template (one %s) converting an objc.ID element to sliceElemGoType
	sliceHasError                bool   // raw getter returns (NSArray, error)

	plainParams    []plainParamModel
	plainRetType   string // Go value return type ("" = void); error is added separately
	plainRetMode   plainRetMode
	plainTrialType string // trial type name for plainRetTrialWrap
	plainHasError  bool   // raw method returns a trailing error (IsNSError)

	// selector is the Objective-C selector this method calls, e.g.
	// "objectAtIndex:". The generated body sends it directly to the object.
	selector string
	// plainRetKind says how the call's result is converted back to Go (string,
	// object, scalar, …); plainReturnWrapper is the expression that wraps an object
	// result, with one %s placeholder for the result pointer; plainMessageSendType is
	// the Go type the underlying call returns before conversion.
	plainRetKind         objKind
	plainReturnWrapper   string
	plainMessageSendType string

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

type asyncParamModel struct {
	goName          string
	goType          string
	rawExpression   string
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
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
) *methodModel {
	lastParam := method.Params[len(method.Params)-1]
	blockParams := parseBlockObjCParams(lastParam.ObjCType)
	// Resolve the block's Go signature exactly as the raw emitter did, so the
	// generated closure matches the raw method's parameter type. Degraded
	// blocks come back as objc.Block and are rejected below.
	blockImpSet := make(typemap.ImportSet)
	blockGoType := rawfw.BlockGoFuncType(lastParam.ObjCType, ctx, mapper, blockImpSet, mapper.OwnerIndex)
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
	var nonBlockParams []asyncParamModel
	for _, param := range method.Params[:len(method.Params)-1] {
		pName := safeParamName(naming.ParamName(param.Name))
		if pName == "" {
			pName = fmt.Sprintf("arg%d", len(nonBlockParams))
		}
		sig, argExpr, imports, ok := idiomaticArg(
			pName,
			param.ObjCType,
			ctx,
			mapper,
			fc,
			rawPkgAlias,
			trialNames,
		)
		if !ok {
			return nil
		}
		maps.Copy(extraImports, imports)
		nonBlockParams = append(
			nonBlockParams,
			asyncParamModel{goName: pName, goType: sig, rawExpression: argExpr},
		)
	}
	maps.Copy(extraImports, resultImports)
	return &methodModel{
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
	class meta.Class,
	fc *frameworkContext,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	abstractBases abstractBaseIndex,
) *methodModel {
	// Async completion → (ctx) error. Fall through to a plain wrapper when the
	// block shape can't be expressed idiomatically (non-NSError result params).
	if isAsyncCompletion(method) {
		if built := buildAsyncMethod(method, rawGoName, fc, ctx, mapper, rawPkgAlias, trialNames); built != nil {
			return built
		}
	}

	// BoolNSError — only when no explicit non-error params.
	if method.IsNSError && strings.TrimSpace(method.Return.ObjCType) == "BOOL" &&
		len(method.Params) == 0 {
		return &methodModel{
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
		if built := buildSliceMethod(method, rawGoName, fc, ctx, mapper, rawPkgAlias, trialNames); built != nil {
			return built
		}
	}

	// Everything else: a plain pass-through wrapper so the method is callable on
	// the idiomatic type without dropping to .Unwrap().
	built := buildPlainMethod(method, rawGoName, fc, ctx, mapper, rawPkgAlias, trialNames, abstractBases)
	if built != nil {
		if size := indexGuardSizeFor(method, class); size != "" && len(built.plainParams) == 1 {
			built.indexGuardSize = size
			built.extraImports["errkit"] = errkitImportPath
		}
	}
	return built
}

// indexGuardSizeFor returns the name of the size accessor to bounds-check an
// index accessor's parameter against, or "" when the method is not a recognised
// single-index accessor or the class does not expose the matching size method.
// It lets the generated accessor panic on an out-of-range index instead of
// raising an Objective-C exception that cannot be caught across the boundary.
func indexGuardSizeFor(method meta.Method, class meta.Class) string {
	var sizeMethod, sizeSelector string
	switch method.Selector {
	case "objectAtIndex:", "objectAtIndexedSubscript:":
		sizeMethod, sizeSelector = "Count", "count"
	case "characterAtIndex:":
		sizeMethod, sizeSelector = "Length", "length"
	default:
		return ""
	}
	for _, classMethod := range class.Methods {
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
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
) *methodModel {
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
		goElem := qualifyRaw(mapper.GoType(elemObjC, ctx, impSet), fc, rawPkgAlias, ctx.GenericParams)
		if !isObjectPointerType(goElem, mapper) {
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
	return &methodModel{
		kind:                         kindSlice,
		goName:                       goName,
		rawGoName:                    rawGoName,
		doc:                          method.Doc,
		selector:                     method.Selector,
		sliceElemGoType:              elemGoType,
		sliceElementConversionFormat: convFmt,
		sliceHasError:                method.IsNSError,
		extraImports:                 extraImports,
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
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	abstractBases abstractBaseIndex,
) *methodModel {
	_ = abstractBases
	extraImports := map[string]string{}
	params := make([]plainParamModel, 0, len(method.Params))
	usedParamNames := map[string]int{}
	outIndex := 0
	for i, p := range method.Params {
		// A value out-parameter is dropped from the signature and surfaced as an
		// extra return value; the call receives a pointer to a local.
		if outGo, isOut := outParamGoType(p.ObjCType, ctx, mapper, fc, rawPkgAlias, false); isOut {
			local := fmt.Sprintf("_out%d", outIndex)
			outIndex++
			outName := safeParamName(naming.ParamName(p.Name))
			if outName == "" {
				outName = fmt.Sprintf("out%d", outIndex)
			}
			params = append(params, plainParamModel{
				goName:        local,
				goType:        outGo,
				rawExpression: "unsafe.Pointer(&" + local + ")",
				isOut:         true,
				outName:       outName,
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
		sig, argExpr, imports, ok := idiomaticArg(
			pName,
			p.ObjCType,
			ctx,
			mapper,
			fc,
			rawPkgAlias,
			trialNames,
		)
		if !ok {
			// A parameter type that cannot yet be expressed without naming an
			// Objective-C runtime type; skip the whole method rather than leak one.
			return nil
		}
		maps.Copy(extraImports, imports)
		params = append(params, plainParamModel{goName: pName, goType: sig, rawExpression: argExpr})
	}

	retType, retKind, retWrap, sendType, retImps, ok := idiomaticRet(
		method.Return.ObjCType, ctx, mapper, fc, rawPkgAlias, trialNames)
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

	return &methodModel{
		kind:                 kindPlain,
		goName:               goName,
		rawGoName:            rawGoName,
		doc:                  method.Doc,
		selector:             method.Selector,
		extraImports:         extraImports,
		plainParams:          params,
		plainRetType:         retType,
		plainRetKind:         retKind,
		plainReturnWrapper:   retWrap,
		plainMessageSendType: sendType,
		plainHasError:        method.IsNSError,
	}
}

func writeMethod(w io.Writer, typeName string, method methodModel) {
	recvVar := uniqueReceiver(typeName, methodParamNames(method))
	writeMethodAs(w, fmt.Sprintf("(%s *%s) ", recvVar, typeName), "objref.IDOf("+recvVar+")", method)
}

// methodParamNames returns the Go signature parameter names of a method entry —
// the names a receiver variable must not collide with. Out-parameters are
// excluded (they are return values), slice and bool methods take no parameters,
// and an async method's leading ctx is included.
func methodParamNames(method methodModel) []string {
	var names []string
	switch method.kind {
	case kindAsync:
		names = append(names, "ctx")
		for _, p := range method.asyncNonBlockParams {
			names = append(names, p.goName)
		}
	case kindPlain:
		for _, p := range method.plainParams {
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
func writeClassFunc(w io.Writer, recvExpr string, method methodModel) {
	writeMethodAs(w, "", recvExpr, method)
}

// writeMethodAs renders a method entry as either an instance method (recv
// "(x *Type) ", recvExpr "objref.IDOf(x)") or a package-level function (recv "",
// recvExpr the class object). recvExpr is the object the Objective-C call is
// sent to.
func writeMethodAs(w io.Writer, recv, recvExpr string, method methodModel) {
	switch method.kind {
	case kindAsync:
		writeAsyncMethod(w, recv, recvExpr, method)
	case kindBoolNSError:
		writeBoolNSErrorMethod(w, recv, recvExpr, method)
	case kindSlice:
		writeSliceMethod(w, recv, recvExpr, method)
	case kindPlain:
		writePlainMethod(w, recv, recvExpr, method)
	}
}

// writePlainMethod renders a Go method (or function) that sends one Objective-C
// message to recvExpr, converting the arguments and result between Go and
// Objective-C.
func writePlainMethod(w io.Writer, recv, recvExpr string, method methodModel) {
	paramParts := make([]string, 0, len(method.plainParams))
	sendArgs := []string{recvExpr, fmt.Sprintf("objc.RegisterName(%q)", method.selector)}
	for _, p := range method.plainParams {
		// Out-parameters are not part of the Go signature, but their pointer is
		// still passed positionally to the Objective-C call.
		if !p.isOut {
			paramParts = append(paramParts, p.goName+" "+p.goType)
		}
		sendArgs = append(sendArgs, p.rawExpression)
	}
	if method.plainHasError {
		sendArgs = append(sendArgs, "unsafe.Pointer(&_nsErr)")
	}
	// objc.Send[...](...) is an expression (the marshaled call), not a Go
	// declaration; the body statements around it are rendered by the template.
	call := fmt.Sprintf("objc.Send[%s](%s)", method.plainMessageSendType, strings.Join(sendArgs, ", "))

	// recv is "(rv *Type) " for an instance method (empty for a package-level
	// class function); the receiver variable is the identifier after the opening
	// paren.
	recvVar := ""
	if fields := strings.Fields(recv); len(fields) > 0 {
		recvVar = strings.TrimPrefix(fields[0], "(")
	}

	var guards []string
	if method.indexGuardSize != "" && len(method.plainParams) == 1 && recvVar != "" {
		guards = append(guards, fmt.Sprintf("errkit.CheckIndex(%s, %s.%s())",
			method.plainParams[0].goName, recvVar, method.indexGuardSize))
	}

	var outs []view.DispatchOut
	for _, p := range outParams(method) {
		outs = append(outs, view.DispatchOut{
			GoName: p.goName,
			GoType: p.goType,
			Zero:   zeroLiteral(p.goType),
		})
	}

	retVars, retTypes := mainThreadReturns(method)

	out, err := render.Method(view.Method{
		DocComment: docLeadKind(
			method.goName,
			method.doc,
			synthFallback(method.goName, plainDocKind(method)),
			plainDocKind(method),
		),
		Recv:     recv,
		GoName:   method.goName,
		ParamStr: strings.Join(paramParts, ", "),
		RetSig:   plainRetSig(method, recvVar),
		Dispatch: view.Dispatch{
			Guards:  guards,
			Call:    call,
			Error:   method.plainHasError,
			RetKind: plainRetKindToView(method.plainRetKind),
			RetWrap: method.plainReturnWrapper,
			RetZero: zeroValue(method.plainRetKind, method.plainRetType),
			Outs:    outs,
		},
		MainThread: method.mainThread,
		RetVars:    retVars,
		RetTypes:   retTypes,
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
func plainDocKind(method methodModel) docKind {
	if len(method.plainParams) != 0 || method.plainHasError || method.plainRetType == "" {
		return docMethod
	}
	if method.plainRetKind == kindBool {
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
func outParams(method methodModel) []plainParamModel {
	var outs []plainParamModel
	for _, p := range method.plainParams {
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
func plainRetSig(method methodModel, recvVar string) string {
	type ret struct{ name, typ string }
	var rets []ret
	hasOut := false
	if method.plainRetType != "" {
		valueName := "result"
		if method.plainRetKind == kindBool {
			valueName = "ok"
		}
		rets = append(rets, ret{valueName, method.plainRetType})
	}
	for _, p := range method.plainParams {
		if p.isOut {
			hasOut = true
			rets = append(rets, ret{p.outName, p.goType})
		}
	}
	if method.plainHasError {
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
	for _, p := range method.plainParams {
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

// mainThreadReturns lists the captured-return locals for a main-thread-wrapped
// plain method: a parallel (names, types) pair, one entry per return value in
// the same order as plainRetSig builds them (value, then out-parameters, then a
// trailing error). The template declares the locals, assigns them from the body
// run inside purego.Main, and returns them. A void method yields empty slices.
func mainThreadReturns(method methodModel) (names, types []string) {
	if method.plainRetType != "" {
		types = append(types, method.plainRetType)
	}
	for _, p := range method.plainParams {
		if p.isOut {
			types = append(types, p.goType)
		}
	}
	if method.plainHasError {
		types = append(types, "error")
	}
	names = make([]string, len(types))
	for i := range types {
		names[i] = fmt.Sprintf("_mainthread%d", i)
	}
	return names, types
}

func writeAsyncMethod(w io.Writer, recv, recvExpr string, method methodModel) {
	paramParts := make([]string, 0, 1+len(method.asyncNonBlockParams))
	paramParts = append(paramParts, "ctx context.Context")
	sendArgs := []string{recvExpr, fmt.Sprintf("objc.RegisterName(%q)", method.selector)}
	for _, p := range method.asyncNonBlockParams {
		paramParts = append(paramParts, p.goName+" "+p.goType)
		sendArgs = append(sendArgs, p.rawExpression)
	}

	// Each block parameter arrives as an Objective-C object pointer. The first
	// error parameter (if any) is converted to a Go error.
	closureParams := make([]string, len(method.blockGoParamTypes))
	errIdx := -1
	for i := range method.blockGoParamTypes {
		closureParams[i] = fmt.Sprintf("_p%d objc.ID", i)
		if errIdx < 0 && isNSErrorType(normaliseObjC(method.blockObjCParams[i])) {
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
	if method.asyncResultGoType == "" {
		out, err := render.AsyncMethod(view.AsyncMethod{
			DocComment: docLead(
				method.goName,
				method.doc,
				synthFallback(method.goName, docMethod),
			),
			Recv:          recv,
			GoName:        method.goName,
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
	ri := method.asyncResultIdx
	var resultConvExpr string
	switch method.asyncResultMode {
	case plainRetString:
		resultConvExpr = fmt.Sprintf("purego.GoString(_p%d)", ri)
	case plainRetTrialWrap:
		resultConvExpr = fmt.Sprintf("%sFromID(_p%d)", method.asyncResultTrial, ri)
	default:
		resultConvExpr = fmt.Sprintf("obj.Wrap(_p%d)", ri)
	}
	out, err := render.AsyncMethod(view.AsyncMethod{
		DocComment:     docLead(method.goName, method.doc, synthFallback(method.goName, docMethod)),
		Recv:           recv,
		GoName:         method.goName,
		ParamStr:       strings.Join(paramParts, ", "),
		HasResult:      true,
		ResultGoType:   method.asyncResultGoType,
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

func writeBoolNSErrorMethod(w io.Writer, recv, recvExpr string, method methodModel) {
	out, err := render.BoolNSErrorMethod(view.BoolNSErrorMethod{
		DocComment: docLead(method.goName, method.doc, synthFallback(method.goName, docMethod)),
		Recv:       recv,
		GoName:     method.goName,
		RecvExpr:   recvExpr,
		Selector:   method.selector,
		MainThread: method.mainThread,
	})
	if err != nil {
		panic(err)
	}
	_, _ = w.Write(out)
}

func writeSliceMethod(w io.Writer, recv, recvExpr string, method methodModel) {
	// The per-element conversion closure is an expression; the template wraps it
	// with purego.NSArrayToSlice.
	conv := fmt.Sprintf(method.sliceElementConversionFormat, "_id")
	convClosure := fmt.Sprintf("func(_id objc.ID) %s { return %s }", method.sliceElemGoType, conv)
	out, err := render.SliceMethod(view.SliceMethod{
		DocComment:  docLeadKind(method.goName, method.doc, synthFallback(method.goName, docGetter), docGetter),
		Recv:        recv,
		GoName:      method.goName,
		RecvExpr:    recvExpr,
		Selector:    method.selector,
		ElemGoType:  method.sliceElemGoType,
		HasError:    method.sliceHasError,
		ConvClosure: convClosure,
		MainThread:  method.mainThread,
	})
	if err != nil {
		panic(err)
	}
	_, _ = w.Write(out)
}
