//go:build darwin

package idiomatic

import (
	"io"
	"maps"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// buildInheritedSetters walks class's superclass chain within framework and returns the
// With* setters inherited from those base classes, so a subclass wrapper exposes
// them too (calling through x.inner, which has the underlying setter via raw
// struct embedding). Setters whose Go name already appears on the subclass
// (ownWith) or on a nearer ancestor are dropped, so subclass and nearer ancestor
// win. The walk stops at the first superclass not defined in framework (e.g. a
// cross-framework base such as NSObject).
func buildInheritedSetters(
	class meta.Class,
	className string,
	fc *frameworkContext,
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	prefix string,
	abstractBases abstractBaseIndex,
	ownWith []withSetterModel,
) []withSetterModel {
	framework := fc.framework
	seen := make(map[string]bool, len(ownWith))
	for _, setter := range ownWith {
		seen[setter.goName] = true
	}

	var inherited []withSetterModel
	for super := class.Super; super != ""; {
		anc, ok := framework.Classes[super]
		if !ok || anc.Availability.IsUnavailable {
			break
		}
		// Dispatch through the explicit embedded-base path (e.g. x.inner.NSView)
		// rather than relying on Go's promotion of x.inner's methods, which is
		// ambiguous when an intermediate class redeclares the setter's Go name.
		ancCtx := typemap.Context{
			ClassName:     super,
			Framework:     framework.Framework,
			GenericParams: anc.GenericParams,
		}
		for _, setter := range buildWithSetters(anc, "", fc, ancCtx, mapper, rawPkgAlias, trialNames, prefix, abstractBases) {
			if seen[setter.goName] {
				continue
			}
			seen[setter.goName] = true
			inherited = append(inherited, setter)
		}
		super = anc.Super
	}
	return inherited
}

// buildWithSetters assembles the fluent With* setter entries for a class's
// settable properties, deduplicated by Go method name.
func buildWithSetters(
	class meta.Class,
	goTypeName string,
	fc *frameworkContext,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	prefix string,
	abstractBases abstractBaseIndex,
) []withSetterModel {
	var withMethods []withSetterModel
	seenWithName := map[string]bool{}
	for _, prop := range class.Properties {
		if prop.IsReadOnly || prop.Availability.IsUnavailable {
			continue
		}
		setter := buildWithSetter(
			prop,
			goTypeName,
			class,
			fc,
			ctx,
			mapper,
			rawPkgAlias,
			trialNames,
			prefix,
			abstractBases,
		)
		if setter == nil || seenWithName[setter.goName] || isReservedMemberName(setter.goName) {
			continue
		}
		setter.doc = prop.Doc
		// A setter on a @MainActor class mutates UI state and must run on the main
		// thread; the template wraps the send in purego.Main.
		if class.IsMainThreadRequired {
			setter.mainThread = true
			if setter.extraImports == nil {
				setter.extraImports = map[string]string{}
			}
			setter.extraImports["purego"] = pureobjcImportPath
		}
		seenWithName[setter.goName] = true
		withMethods = append(withMethods, *setter)
	}
	return withMethods
}

type withSetterModel struct {
	goName          string // e.g. "WithVariableStore"
	doc             string // Apple/header documentation for the underlying property
	rawSetterGoName string // e.g. "SetVariableStore"
	setterSelector  string // the Objective-C setter selector, e.g. "setVariableStore:"
	param           setterParamModel
	isNSArray       bool   // true: variadic slice → NSArray
	sliceElemType   string // parameter elem type (a provider interface or an object type)
	mainThread      bool   // run the setter on the main thread (@MainActor class)
	extraImports    map[string]string
}

type setterParamModel struct {
	goName        string // variable name in the generated func
	goType        string // Go type (string for NSURL/NSString, *raw.T or ProviderIface for objects, uint64 etc.)
	rawExpression string // expression passed to the raw setter (may differ from param name)
}

func buildWithSetter(
	prop meta.Property,
	_ string,
	class meta.Class,
	fc *frameworkContext,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	_ string,
	abstractBases abstractBaseIndex,
) *withSetterModel {
	// Derive the raw setter Go method name from the selector.
	setterSel := prop.Setter
	if setterSel == "" {
		if len(prop.Name) == 0 {
			return nil
		}
		setterSel = "set" + strings.ToUpper(prop.Name[:1]) + prop.Name[1:] + ":"
	}
	rawSetterGoName := naming.MethodName(setterSel)

	// Verify the setter method actually exists in the raw bindings.
	setterExists := false
	for _, method := range class.Methods {
		if method.Selector == setterSel && !method.IsClassMethod {
			setterExists = true
			break
		}
	}
	if !setterExists {
		return nil
	}

	// Derive With* Go name from the property name, re-casing any initialism
	// (WithUsbController → WithUSBController) for idiomatic capitalisation.
	goWithName := applyInitialisms("With" + strings.ToUpper(prop.Name[:1]) + prop.Name[1:])

	norm := normaliseObjC(prop.ObjCType)

	extraImports := map[string]string{}

	_ = norm

	// NSArray property → variadic With*(items ...Elem). Elem is a provider
	// interface when the array holds an abstract base class, the element's own
	// wrapper type when it has one, otherwise a generic object.
	if looksLikeNSArray(prop.ObjCType) {
		elemObjC := extractNSArrayElem(prop.ObjCType)
		if elemObjC == "" {
			return nil
		}
		impSet := make(typemap.ImportSet)
		goElem := qualifyRaw(mapper.GoType(elemObjC, ctx, impSet), fc, rawPkgAlias, ctx.GenericParams)
		if !isObjectPointerType(goElem, mapper) {
			return nil
		}
		extraImports["objref"] = objrefImportPath
		extraImports["objc"] = objcImportPath
		extraImports["purego"] = pureobjcImportPath
		elemType := "obj.Object"
		extraImports["obj"] = objImportPath
		rawElemClass := strings.TrimPrefix(strings.TrimPrefix(goElem, "*"), rawPkgAlias+".")
		if baseGoTypeName, isBase := abstractBases[rawElemClass]; isBase {
			elemType = providerInterfaceName(baseGoTypeName)
			delete(extraImports, "obj")
		} else if base, has := trialWrapClass(goElem, rawPkgAlias); has {
			if tt, named := trialNames[base]; named {
				elemType = "*" + tt
				delete(extraImports, "obj")
			}
		}
		return &withSetterModel{
			goName:          goWithName,
			rawSetterGoName: rawSetterGoName,
			setterSelector:  setterSel,
			isNSArray:       true,
			sliceElemType:   elemType,
			extraImports:    extraImports,
		}
	}

	pName := safeParamName(naming.ParamName(prop.Name))
	if pName == "" {
		pName = "v"
	}
	extraImports["objref"] = objrefImportPath
	extraImports["objc"] = objcImportPath

	// A property typed as an abstract base class accepts a provider interface so
	// any concrete subtype can be passed.
	impSet := make(typemap.ImportSet)
	goType := qualifyRaw(
		rawParamGoType(prop.ObjCType, ctx, mapper, impSet),
		fc,
		rawPkgAlias,
		ctx.GenericParams,
	)
	rawClass := strings.TrimPrefix(strings.TrimPrefix(goType, "*"), rawPkgAlias+".")
	if baseGoTypeName, isBase := abstractBases[rawClass]; isBase {
		return &withSetterModel{
			goName:          goWithName,
			rawSetterGoName: rawSetterGoName,
			setterSelector:  setterSel,
			param: setterParamModel{
				goName:        pName,
				goType:        providerInterfaceName(baseGoTypeName),
				rawExpression: "objref.IDOf(" + pName + ")",
			},
			extraImports: extraImports,
		}
	}

	sig, argExpr, imports, ok := idiomaticArg(
		pName,
		prop.ObjCType,
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
	return &withSetterModel{
		goName:          goWithName,
		rawSetterGoName: rawSetterGoName,
		setterSelector:  setterSel,
		param:           setterParamModel{goName: pName, goType: sig, rawExpression: argExpr},
		extraImports:    extraImports,
	}
}

func writeWithMethod(w io.Writer, typeName string, setter withSetterModel) {
	// The setter's single parameter ("items" for a collection, otherwise the
	// property name) must not collide with the receiver variable.
	paramName := "items"
	if !setter.isNSArray {
		paramName = setter.param.goName
	}
	view := withSetterView{
		DocComment: docLeadKind(
			setter.goName,
			setter.doc,
			synthFallback(setter.goName, docSetter),
			docSetter,
		),
		TypeName:       typeName,
		RecvVar:        uniqueReceiver(typeName, []string{paramName}),
		GoName:         setter.goName,
		SetterSelector: setter.setterSelector,
		IsNSArray:      setter.isNSArray,
		MainThread:     setter.mainThread,
	}
	if setter.isNSArray {
		view.SliceElemType = setter.sliceElemType
	} else {
		view.ParamName = setter.param.goName
		view.ParamType = setter.param.goType
		view.ParamRawExpr = setter.param.rawExpression
	}

	renderTemplate(w, "with_setter", view)
}

// withSetterView is the template data for with_setter.tmpl.
type withSetterView struct {
	DocComment     string
	TypeName       string
	RecvVar        string
	GoName         string
	SetterSelector string
	IsNSArray      bool
	MainThread     bool // wrap the setter send in purego.Main (@MainActor class)

	// NSArray collection setter:
	SliceElemType string

	// Scalar/object setter:
	ParamName    string
	ParamType    string
	ParamRawExpr string
}
