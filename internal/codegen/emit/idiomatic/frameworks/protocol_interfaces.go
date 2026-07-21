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
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emitmanifest"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// emitProtocols writes <pkgname>_protocols_generated.go: a Go interface for every
// Objective-C protocol the framework declares that is not already surfaced as a
// richer delegate interface. Each interface lists the protocol's required
// (non-optional, available) methods with idiomatic Go signatures, so a Go value
// satisfies the protocol by declaring them. The interfaces are standalone (no
// wrapper references them yet), so emitting them only adds declarations.
//
// Precedence: a protocol already emitted as a delegate (fc.delegates) is left to
// that richer form, and a protocol whose interface name is already claimed by a
// class wrapper, provider, or delegate is skipped — those cases keep the existing
// counterpart, which the parity manifest already credits.
func emitProtocols(
	outDir, pkgName, rawPkgAlias string,
	fc *frameworkContext,
	mapper *typemap.Mapper,
	trialNames trialNameMap,
	takenNames map[string]bool,
) error {
	framework := fc.framework
	ctx := typemap.Context{Framework: framework.Framework}

	names := make([]string, 0, len(framework.Protocols))
	for name := range framework.Protocols {
		names = append(names, name)
	}
	sort.Strings(names)

	imports := map[string]string{}
	var protocols []view.Protocol
	for _, protocolName := range names {
		protocol := framework.Protocols[protocolName]
		if protocol.Availability.IsUnavailable {
			continue
		}
		// A delegate already provides this protocol's richer Go form (and its
		// KindProtocol manifest entry). Still record its required methods so their
		// parity is credited to that delegate interface.
		if _, isDelegate := fc.delegates[protocolName]; isDelegate {
			_, methodImports := buildProtocolInterfaceMethods(
				protocol, protocolName, framework.Framework, pkgName,
				fc, ctx, mapper, rawPkgAlias, trialNames,
			)
			maps.Copy(imports, methodImports)
			continue
		}
		// Resolve the interface name, disambiguating a clash with a class wrapper or
		// another claimed name by the raw layer's "Protocol" suffix (NSObject the
		// class vs NSObjectProtocol the protocol interface).
		ifaceName := trialTypeName(protocolName, fc.prefix)
		if !isExportedGoIdent(ifaceName) {
			continue
		}
		if takenNames[ifaceName] || fc.classGoNames[ifaceName] {
			ifaceName += "Protocol"
			if takenNames[ifaceName] || fc.classGoNames[ifaceName] {
				continue // still clashing — leave it (a rare residual gap)
			}
		}
		takenNames[ifaceName] = true

		methods, methodImports := buildProtocolInterfaceMethods(
			protocol, protocolName, framework.Framework, pkgName,
			fc, ctx, mapper, rawPkgAlias, trialNames,
		)
		maps.Copy(imports, methodImports)

		fc.manifest.Record(emitmanifest.Entry{
			Style:     emitmanifest.StyleIdiomatic,
			Kind:      emitmanifest.KindProtocol,
			Framework: framework.Framework,
			MetaKey: emitmanifest.MetaKey(
				framework.Framework,
				emitmanifest.KindProtocol,
				protocolName,
				"",
			),
			GoPkg:    pkgName,
			GoSymbol: ifaceName,
		})

		protocols = append(protocols, view.Protocol{
			Doc: fmt.Sprintf(
				"%s is the Go form of the Objective-C protocol %s.",
				ifaceName,
				protocolName,
			),
			GoName:  ifaceName,
			Methods: methods,
		})
	}
	if len(protocols) == 0 {
		return nil
	}

	body, err := render.Protocols(protocols)
	if err != nil {
		return err
	}
	// idiomaticArg/idiomaticRet report the imports their value CONVERSIONS need,
	// but an interface signature uses only the type — so keep only the imports
	// whose package selector actually appears in the rendered signatures.
	used := make(map[string]string, len(imports))
	bodyStr := string(body)
	for alias, path := range imports {
		if strings.Contains(bodyStr, alias+".") {
			used[alias] = path
		}
	}
	fname := pkgName + "_protocols_generated.go"
	return rawfw.WriteGoFile(filepath.Join(outDir, fname), assembleFile(pkgName, used, body))
}

// buildProtocolInterfaceMethods resolves a protocol's required methods into Go
// interface methods with idiomatic signatures, recording each in the parity
// manifest (keyed on the protocol name plus the ObjC selector, matching the raw
// oracle). A parameter or return that cannot be expressed idiomatically degrades
// to obj.Object rather than dropping the method.
func buildProtocolInterfaceMethods(
	protocol meta.Protocol,
	protocolName, framework, pkgName string,
	fc *frameworkContext,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
) ([]view.ProtocolMethod, map[string]string) {
	imports := map[string]string{}
	var methods []view.ProtocolMethod
	seenGoNames := make(map[string]bool)
	for _, method := range protocol.Methods {
		if method.Availability.IsUnavailable || method.IsOptional {
			continue
		}
		goName := applyInitialisms(naming.MethodName(method.Selector))
		if !isExportedGoIdent(goName) || seenGoNames[goName] {
			continue
		}
		seenGoNames[goName] = true

		signature := buildProtocolMethodSignature(
			method,
			fc,
			ctx,
			mapper,
			rawPkgAlias,
			trialNames,
			imports,
		)
		methods = append(methods, view.ProtocolMethod{GoName: goName, Signature: signature})

		fc.manifest.Record(emitmanifest.Entry{
			Style:     emitmanifest.StyleIdiomatic,
			Kind:      emitmanifest.KindProtocolMethod,
			Framework: framework,
			MetaKey: emitmanifest.MetaKey(
				framework,
				emitmanifest.KindProtocolMethod,
				protocolName,
				method.Selector,
			),
			GoPkg:    pkgName,
			GoSymbol: goName,
		})
	}
	return methods, imports
}

// buildProtocolMethodSignature builds one interface method's parameter list and
// return clause using the same idiomatic type resolution as free functions,
// degrading an unexpressible parameter or return to obj.Object.
func buildProtocolMethodSignature(
	method meta.Method,
	fc *frameworkContext,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	imports map[string]string,
) string {
	degradeObj := func() string {
		imports["obj"] = objImportPath
		return "obj.Object"
	}

	params := make([]string, 0, len(method.Params))
	usedNames := map[string]int{}
	for i, param := range method.Params {
		pName := safeParamName(naming.ParamName(param.Name))
		if pName == "" {
			pName = fmt.Sprintf("arg%d", i)
		}
		usedNames[pName]++
		if usedNames[pName] > 1 {
			pName = fmt.Sprintf("%s%d", pName, usedNames[pName])
		}
		sig, _, imps, ok := idiomaticArg(
			pName,
			param.ObjCType,
			ctx,
			mapper,
			fc,
			rawPkgAlias,
			trialNames,
		)
		if !ok {
			sig = degradeObj()
		} else {
			maps.Copy(imports, imps)
		}
		params = append(params, pName+" "+sig)
	}

	retType, _, _, _, rimps, ok := idiomaticRet(
		method.Return.ObjCType,
		ctx,
		mapper,
		fc,
		rawPkgAlias,
		trialNames,
	)
	retSig := ""
	switch {
	case !ok:
		retSig = " " + degradeObj()
	case retType != "":
		maps.Copy(imports, rimps)
		retSig = " " + retType
	}
	return "(" + strings.Join(params, ", ") + ")" + retSig
}
