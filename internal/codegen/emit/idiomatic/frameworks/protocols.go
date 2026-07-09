//go:build darwin

package idiofw

import (
	"fmt"
	"maps"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/view"
	rawfw "github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// protocols.go surfaces Objective-C delegate protocols as Go interfaces. For
// each selected protocol the emitter writes <Protocol>_delegate_generated.go:
// a Go interface holding the protocol's required methods, a one-method
// "upgrade" interface per @optional method, and an unexported shim builder
// that wraps a Go value in a runtime-registered Objective-C class whose IMPs
// route each callback to the value. Delegate-typed property setters accept the
// interface (see buildWithSetter) and tie the shim's lifetime to the owner via
// an associated object.

// delegateModel is one selected, fully-bridgeable-enough protocol: the
// resolved view plus the file-level imports its conversions need.
type delegateModel struct {
	protocolName string
	view         view.Delegate
	imports      map[string]string
}

// buildDelegates selects the framework's delegate-shaped protocols and builds
// a model for each. Selection is by name shape (*Delegate / *DataSource /
// *Observer), overridden by the idiomatic.json sidecar's delegates.include /
// delegates.exclude lists. A protocol is dropped (with a diagnostic) when a
// required method cannot be bridged or when no method survives; an @optional
// method that cannot be bridged is dropped alone.
func buildDelegates(
	fc *frameworkContext,
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
) []delegateModel {
	framework := fc.framework
	var names []string
	for protocolName, protocol := range framework.Protocols {
		if protocol.Availability.IsUnavailable {
			continue
		}
		if fc.idio.IsDelegateExcluded(protocolName) {
			continue
		}
		if !fc.idio.IsDelegateIncluded(protocolName) && !isDelegateShapedName(protocolName) {
			continue
		}
		names = append(names, protocolName)
	}
	sort.Strings(names)

	usedIfaceNames := make(map[string]bool)
	var delegates []delegateModel
	for _, protocolName := range names {
		model, reason := buildDelegate(
			framework.Protocols[protocolName], protocolName,
			fc, mapper, rawPkgAlias, trialNames, usedIfaceNames,
		)
		if model == nil {
			mapper.AppendDiagnostic(
				"%s: delegate interface for %s not emitted (%s)",
				framework.Framework, protocolName, reason,
			)
			continue
		}
		usedIfaceNames[model.view.IfaceName] = true
		for _, optional := range model.view.Optional {
			usedIfaceNames[optional.OptIfaceName] = true
		}
		delegates = append(delegates, *model)
	}
	return delegates
}

// singleProtocolOf reports the protocol name when objcType is a plain
// single-protocol object reference ("id<VZVirtualMachineDelegate>", with or
// without nullability annotations). Multi-protocol references and class
// pointers return ("", false).
func singleProtocolOf(objcType string) (string, bool) {
	t := normaliseObjC(objcType)
	rest, ok := strings.CutPrefix(t, "id<")
	if !ok {
		return "", false
	}
	name, _, ok := strings.Cut(rest, ">")
	if !ok || strings.ContainsAny(name, ",<*") {
		return "", false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	return name, true
}

// isDelegateShapedName reports whether a protocol's name marks it as a
// callback-receiver protocol by Apple's naming convention.
func isDelegateShapedName(protocolName string) bool {
	for _, suffix := range []string{"Delegate", "DataSource", "Observer"} {
		if strings.HasSuffix(protocolName, suffix) {
			return true
		}
	}
	return false
}

// buildDelegate resolves one protocol into a delegateModel, or returns nil and
// a human-readable reason.
func buildDelegate(
	protocol meta.Protocol,
	protocolName string,
	fc *frameworkContext,
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	usedIfaceNames map[string]bool,
) (*delegateModel, string) {
	ifaceName := trialTypeName(protocolName, fc.prefix)
	if !isExportedGoIdent(ifaceName) {
		return nil, "no exported Go interface name"
	}
	if fc.classGoNames[ifaceName] || usedIfaceNames[ifaceName] {
		return nil, fmt.Sprintf("interface name %s already taken", ifaceName)
	}

	ctx := typemap.Context{Framework: fc.framework.Framework}
	imports := map[string]string{
		"objc": objcImportPath,
		"shim": shimImportPath,
		"sync": "sync",
	}

	var required, optional []view.DelegateMethod
	seenName := map[string]bool{}
	for _, method := range protocol.Methods {
		// A method the SDK no longer offers is never sent; a class method is
		// not a delegate callback.
		if method.IsClassMethod || method.Availability.IsUnavailable {
			continue
		}
		var built view.DelegateMethod
		var methodImports map[string]string
		bridged := !method.IsVariadic
		if bridged {
			built, methodImports, bridged = buildDelegateMethod(
				method,
				ifaceName,
				fc,
				ctx,
				mapper,
				rawPkgAlias,
				trialNames,
			)
		}
		if !bridged {
			if method.IsOptional {
				continue // an optional callback the Go side simply cannot receive
			}
			// A required method the shim cannot implement would crash with an
			// unrecognized selector the first time the framework sends it, so
			// the whole protocol stays unbridged.
			return nil, fmt.Sprintf("required method %s is not bridgeable", method.Selector)
		}
		if seenName[built.GoName] {
			continue
		}
		seenName[built.GoName] = true
		maps.Copy(imports, methodImports)
		if method.IsOptional {
			built.OptIfaceName = ifaceName + built.GoName + "Handler"
			built.AssertIface = built.OptIfaceName
			built.OptionalDoc = fmt.Sprintf(
				"// %s is the optional %s method %s. Implement it on a %s value\n// to receive that callback; a value without it is simply never sent one.\n",
				built.OptIfaceName,
				protocolName,
				method.Selector,
				ifaceName,
			)
			optional = append(optional, built)
		} else {
			built.AssertIface = ifaceName
			required = append(required, built)
		}
	}
	if len(required)+len(optional) == 0 {
		return nil, "no bridgeable methods"
	}

	docComment := fmt.Sprintf(
		"// %s receives the callbacks of the Objective-C protocol %s.\n// Pass an implementation to the matching With… setter; the wrapper builds the\n// Objective-C delegate object for you. Every method listed here is required.\n// The protocol's optional methods are separate one-method interfaces\n// (%s…Handler): implement the ones you need on the same value and the\n// framework will call them too.\n",
		ifaceName,
		protocolName,
		ifaceName,
	)

	return &delegateModel{
		protocolName: protocolName,
		imports:      imports,
		view: view.Delegate{
			DocComment:    docComment,
			IfaceName:     ifaceName,
			ProtocolName:  protocolName,
			ShimClassName: "GoShim" + fc.framework.Framework + ifaceName,
			ShimFuncName:  delegateShimFuncName(ifaceName),
			ClassVar:      "_" + lowerFirst(ifaceName) + "ShimClass",
			Required:      required,
			Optional:      optional,
		},
	}, ""
}

// delegateShimFuncName is the unexported builder that wraps a Go value as an
// Objective-C delegate (VirtualMachineDelegate → newVirtualMachineDelegateShim).
func delegateShimFuncName(ifaceName string) string {
	return "new" + ifaceName + "Shim"
}

// buildDelegateMethod resolves one protocol method into its interface
// signature and callback pieces. ok is false when a parameter or the result
// cannot cross from Objective-C into Go (blocks, by-value structs, object
// returns).
func buildDelegateMethod(
	method meta.Method,
	ifaceName string,
	fc *frameworkContext,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
) (view.DelegateMethod, map[string]string, bool) {
	imports := map[string]string{}
	goName := applyInitialisms(naming.MethodName(method.Selector))
	if !isExportedGoIdent(goName) {
		return view.DelegateMethod{}, nil, false
	}

	var sigParts, abiParams, callArgs []string
	usedParamNames := map[string]int{}
	for i, param := range method.Params {
		local := fmt.Sprintf("_p%d", i)
		sigType, abiType, callArg, paramImports, ok := shimParam(
			local,
			param.ObjCType,
			fc,
			ctx,
			mapper,
			rawPkgAlias,
			trialNames,
		)
		if !ok {
			return view.DelegateMethod{}, nil, false
		}
		maps.Copy(imports, paramImports)
		pName := safeParamName(naming.ParamName(param.Name))
		if pName == "" {
			pName = fmt.Sprintf("arg%d", i)
		}
		usedParamNames[pName]++
		if usedParamNames[pName] > 1 {
			pName = fmt.Sprintf("%s%d", pName, usedParamNames[pName])
		}
		sigParts = append(sigParts, pName+" "+sigType)
		abiParams = append(abiParams, local+" "+abiType)
		callArgs = append(callArgs, callArg)
	}

	retSig, abiRet, retWrap, retZero, retImports, ok := shimReturn(
		method.Return.ObjCType,
		fc,
		ctx,
		mapper,
		rawPkgAlias,
	)
	if !ok {
		return view.DelegateMethod{}, nil, false
	}
	maps.Copy(imports, retImports)

	built := view.DelegateMethod{
		DocComment: docLead(goName, method.Doc, synthFallback(goName, docMethod)),
		GoName:     goName,
		Selector:   method.Selector,
		SigParams:  strings.Join(sigParts, ", "),
		RetSig:     retSig,
		ABIParams:  abiParams,
		ABIRet:     abiRet,
		CallArgs:   callArgs,
		RetZero:    retZero,
	}
	if retSig != "" {
		call := "_h." + goName + "(" + strings.Join(callArgs, ", ") + ")"
		built.RetExpr = strings.Replace(retWrap, "%s", call, 1)
	}
	return built, imports, true
}

// shimParam resolves one incoming callback parameter: the Go interface
// signature type, the IMP's parameter type, and the expression converting the
// IMP parameter (named local) into the Go argument. ok is false for shapes
// that cannot cross into Go (blocks, by-value structs, unresolvable types).
func shimParam(
	local, objcType string,
	fc *frameworkContext,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
) (sigType, abiType, callArg string, imports map[string]string, ok bool) {
	imports = map[string]string{}
	if strings.Contains(objcType, "(^") {
		return "", "", "", nil, false // a block parameter cannot cross into Go
	}
	norm := normaliseObjC(objcType)
	switch {
	case isNSErrorType(norm):
		imports["errkit"] = errkitImportPath
		imports["purego"] = pureobjcImportPath
		return "error", "objc.ID", "errkit.FromObjC(purego.NSErrorToError(" + local + "))", imports, true
	case isNSStringType(norm):
		imports["purego"] = pureobjcImportPath
		return "string", "objc.ID", "purego.GoString(" + local + ")", imports, true
	case isNSURLType(norm):
		imports["rt"] = rtImportPath
		return "string", "objc.ID", "rt.URLString(" + local + ")", imports, true
	case isNSDateType(norm):
		imports["rt"] = rtImportPath
		imports["time"] = "time"
		return "time.Time", "objc.ID", "rt.NSDateToTime(" + local + ")", imports, true
	case isNSDataType(norm):
		imports["rt"] = rtImportPath
		return "[]byte", "objc.ID", "rt.NSDataToBytes(" + local + ")", imports, true
	}

	goType := qualifyRaw(
		rawParamGoType(objcType, ctx, mapper, make(typemap.ImportSet)),
		fc, rawPkgAlias, ctx.GenericParams,
	)
	switch goType {
	case "":
		return "", "", "", nil, false
	case "bool":
		return "bool", "bool", local, imports, true
	}
	if sigEnum, _, isEnum := localizeEnumType(goType, fc, rawPkgAlias); isEnum {
		return sigEnum, sigEnum, local, imports, true
	}
	if base, has := trialWrapClass(goType, rawPkgAlias); has {
		if trialName, named := trialNames[base]; named {
			return "*" + trialName, "objc.ID", trialName + "FromID(" + local + ")", imports, true
		}
	}
	if isObjectGoType(goType, mapper) {
		imports["obj"] = objImportPath
		return "obj.Object", "objc.ID", "obj.Wrap(" + local + ")", imports, true
	}
	if isHermeticScalar(goType) {
		sized := goSizeType(goType)
		if sized != goType {
			return sized, goType, sized + "(" + local + ")", imports, true
		}
		return goType, goType, local, imports, true
	}
	return "", "", "", nil, false // by-value structs and pointers stay unbridged
}

// shimReturn resolves a callback's result: the Go interface return clause, the
// IMP's return type, the Go→Objective-C conversion (one %s for the Go call
// expression), and the zero the IMP returns when the Go value does not
// implement the method. Only value results cross back (void, BOOL, scalars,
// enums, and strings); an object result would need an ownership contract the
// shim cannot promise, so such a method is not bridged.
func shimReturn(
	objcType string,
	fc *frameworkContext,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	rawPkgAlias string,
) (retSig, abiRet, retWrap, retZero string, imports map[string]string, ok bool) {
	imports = map[string]string{}
	ret := strings.TrimSpace(objcType)
	if ret == "" || ret == "void" {
		return "", "", "", "", imports, true
	}
	if strings.Contains(ret, "(^") {
		return "", "", "", "", nil, false
	}
	if isNSStringType(normaliseObjC(ret)) {
		imports["purego"] = pureobjcImportPath
		return " string", "objc.ID", "purego.NSString(%s)", "0", imports, true
	}
	goType := qualifyRaw(
		mapper.GoReturnType(objcType, ctx, make(typemap.ImportSet)),
		fc,
		rawPkgAlias,
		ctx.GenericParams,
	)
	switch goType {
	case "":
		return "", "", "", "", nil, false
	case "bool":
		return " bool", "bool", "%s", "false", imports, true
	}
	if sigEnum, _, isEnum := localizeEnumType(goType, fc, rawPkgAlias); isEnum {
		return " " + sigEnum, sigEnum, "%s", sigEnum + "(0)", imports, true
	}
	if isHermeticScalar(goType) {
		sized := goSizeType(goType)
		if sized != goType {
			return " " + sized, goType, goType + "(%s)", "0", imports, true
		}
		return " " + goType, goType, "%s", "0", imports, true
	}
	return "", "", "", "", nil, false
}

// emitDelegateFiles writes one <Protocol>_delegate_generated.go per built
// delegate.
func emitDelegateFiles(outDir, pkgName string, delegates []delegateModel) error {
	for _, delegate := range delegates {
		body, err := render.Delegate(delegate.view)
		if err != nil {
			return fmt.Errorf("render delegate %s: %w", delegate.protocolName, err)
		}
		fname := delegate.protocolName + "_delegate_generated.go"
		file := assembleFile(pkgName, delegate.imports, body)
		if err := rawfw.WriteGoFile(filepath.Join(outDir, fname), file); err != nil {
			return fmt.Errorf("write delegate %s: %w", delegate.protocolName, err)
		}
	}
	return nil
}
