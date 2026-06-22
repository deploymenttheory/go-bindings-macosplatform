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

// buildInheritedSetters walks cls's superclass chain within fw and returns the
// With* setters inherited from those base classes, so a subclass wrapper exposes
// them too (calling through x.inner, which has the underlying setter via raw
// struct embedding). Setters whose Go name already appears on the subclass
// (ownWith) or on a nearer ancestor are dropped, so subclass and nearer ancestor
// win. The walk stops at the first superclass not defined in fw (e.g. a
// cross-framework base such as NSObject).
func buildInheritedSetters(
	cls meta.Class,
	className string,
	fc *frameworkContext,
	m *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	prefix string,
	abstractBases abstractBaseIndex,
	ownWith []withEntry,
) []withEntry {
	fw := fc.fw
	seen := make(map[string]bool, len(ownWith))
	for _, we := range ownWith {
		seen[we.goName] = true
	}

	var inherited []withEntry
	for super := cls.Super; super != ""; {
		anc, ok := fw.Classes[super]
		if !ok || anc.Availability.IsUnavailable {
			break
		}
		// Dispatch through the explicit embedded-base path (e.g. x.inner.NSView)
		// rather than relying on Go's promotion of x.inner's methods, which is
		// ambiguous when an intermediate class redeclares the setter's Go name.
		ancCtx := typemap.Context{
			ClassName:     super,
			Framework:     fw.Framework,
			GenericParams: anc.GenericParams,
		}
		for _, we := range buildWithSetters(anc, "", fc, ancCtx, m, rawPkgAlias, trialNames, prefix, abstractBases) {
			if seen[we.goName] {
				continue
			}
			seen[we.goName] = true
			inherited = append(inherited, we)
		}
		super = anc.Super
	}
	return inherited
}

// buildWithSetters assembles the fluent With* setter entries for a class's
// settable properties, deduplicated by Go method name.
func buildWithSetters(
	cls meta.Class,
	goTypeName string,
	fc *frameworkContext,
	ctx typemap.Context,
	m *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	prefix string,
	abstractBases abstractBaseIndex,
) []withEntry {
	var withMethods []withEntry
	seenWithName := map[string]bool{}
	for _, prop := range cls.Properties {
		if prop.IsReadOnly || prop.Availability.IsUnavailable {
			continue
		}
		we := buildWithSetter(
			prop,
			goTypeName,
			cls,
			fc,
			ctx,
			m,
			rawPkgAlias,
			trialNames,
			prefix,
			abstractBases,
		)
		if we == nil || seenWithName[we.goName] || isReservedMemberName(we.goName) {
			continue
		}
		we.doc = prop.Doc
		seenWithName[we.goName] = true
		withMethods = append(withMethods, *we)
	}
	return withMethods
}

type withEntry struct {
	goName          string // e.g. "WithVariableStore"
	doc             string // Apple/header documentation for the underlying property
	rawSetterGoName string // e.g. "SetVariableStore"
	setterSelector  string // the Objective-C setter selector, e.g. "setVariableStore:"
	param           withParam
	isNSArray       bool   // true: variadic slice → NSArray
	sliceElemType   string // parameter elem type (a provider interface or an object type)
	extraImports    map[string]string
}

type withParam struct {
	goName  string // variable name in the generated func
	goType  string // Go type (string for NSURL/NSString, *raw.T or ProviderIface for objects, uint64 etc.)
	rawExpr string // expression passed to the raw setter (may differ from param name)
}

func buildWithSetter(
	prop meta.Property,
	_ string,
	cls meta.Class,
	fc *frameworkContext,
	ctx typemap.Context,
	m *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	_ string,
	abstractBases abstractBaseIndex,
) *withEntry {
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
	for _, method := range cls.Methods {
		if method.Selector == setterSel && !method.IsClassMethod {
			setterExists = true
			break
		}
	}
	if !setterExists {
		return nil
	}

	// Derive With* Go name from the property name.
	goWithName := "With" + strings.ToUpper(prop.Name[:1]) + prop.Name[1:]

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
		goElem := qualifyRaw(m.GoType(elemObjC, ctx, impSet), fc, rawPkgAlias, ctx.GenericParams)
		if !isObjectPointerType(goElem, m) {
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
		return &withEntry{
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
		rawParamGoType(prop.ObjCType, ctx, m, impSet),
		fc,
		rawPkgAlias,
		ctx.GenericParams,
	)
	rawClass := strings.TrimPrefix(strings.TrimPrefix(goType, "*"), rawPkgAlias+".")
	if baseGoTypeName, isBase := abstractBases[rawClass]; isBase {
		return &withEntry{
			goName:          goWithName,
			rawSetterGoName: rawSetterGoName,
			setterSelector:  setterSel,
			param: withParam{
				goName:  pName,
				goType:  providerInterfaceName(baseGoTypeName),
				rawExpr: "objref.IDOf(" + pName + ")",
			},
			extraImports: extraImports,
		}
	}

	sig, argExpr, imps, ok := idiomaticArg(
		pName,
		prop.ObjCType,
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
	return &withEntry{
		goName:          goWithName,
		rawSetterGoName: rawSetterGoName,
		setterSelector:  setterSel,
		param:           withParam{goName: pName, goType: sig, rawExpr: argExpr},
		extraImports:    extraImports,
	}
}

func writeWithMethod(w io.Writer, typeName string, we withEntry) {
	view := withSetterView{
		DocComment: docLead(
			we.goName,
			we.doc,
			"sets the property and returns the receiver so calls can be chained.",
		),
		TypeName:       typeName,
		GoName:         we.goName,
		SetterSelector: we.setterSelector,
		IsNSArray:      we.isNSArray,
	}
	if we.isNSArray {
		view.SliceElemType = we.sliceElemType
	} else {
		view.ParamName = we.param.goName
		view.ParamType = we.param.goType
		view.ParamRawExpr = we.param.rawExpr
	}

	renderTemplate(w, "with_setter", view)
}

// withSetterView is the template data for with_setter.tmpl.
type withSetterView struct {
	DocComment     string
	TypeName       string
	GoName         string
	SetterSelector string
	IsNSArray      bool

	// NSArray collection setter:
	SliceElemType string

	// Scalar/object setter:
	ParamName    string
	ParamType    string
	ParamRawExpr string
}
