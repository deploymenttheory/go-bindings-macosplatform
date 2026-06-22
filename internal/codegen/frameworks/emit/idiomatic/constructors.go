//go:build darwin

package idiomatic

import (
	"fmt"
	"io"
	"maps"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// buildConstructors assembles the constructor entries for a class: one per
// parameterised init, plus a plain +new constructor when appropriate.
func buildConstructors(
	cls meta.Class,
	className, goTypeName string,
	fc *frameworkContext,
	ctx typemap.Context,
	m *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	abstractBases abstractBaseIndex,
) []ctorEntry {
	var ctors []ctorEntry
	hasExplicitParamInit := false

	for _, method := range cls.Methods {
		if method.Availability.IsUnavailable || method.IsClassMethod || !method.IsInit {
			continue
		}
		if len(method.Params) == 0 {
			continue // plain init handled below as +new fallback
		}
		e := buildParamConstructor(
			method,
			goTypeName,
			cls,
			fc,
			ctx,
			m,
			rawPkgAlias,
			trialNames,
			abstractBases,
		)
		if e != nil {
			e.doc = method.Doc
			ctors = append(ctors, *e)
			hasExplicitParamInit = true
		}
	}

	// Every class gets a plain +new constructor unless it has a param-init that
	// already covers the no-arg case.
	if !hasExplicitParamInit {
		return append(
			[]ctorEntry{buildNewConstructor(className, goTypeName, rawPkgAlias)},
			ctors...)
	}
	// Still provide a plain constructor if there is a class-specific plain init too.
	for _, method := range cls.Methods {
		if method.IsInit && len(method.Params) == 0 && !method.Availability.IsUnavailable {
			return append(
				[]ctorEntry{buildNewConstructor(className, goTypeName, rawPkgAlias)},
				ctors...)
		}
	}
	return ctors
}

type ctorEntry struct {
	goName          string
	doc             string // Apple/header documentation for the underlying init
	rawInitGoName   string // "" = use +new
	rawInitSelector string // ObjC selector, e.g. "initWithURL:readOnly:error:"
	params          []ctorParam
	hasNSError      bool
	needsFoundation bool
	extraImports    map[string]string
	// preamble is rendered verbatim at the top of the constructor body, before
	// alloc/init — used to build a raw NSArray from a variadic ergonomic param.
	preamble string
}

type ctorParam struct {
	goName   string
	goType   string
	rawExpr  string // expression passed to the raw init method
	isObject bool   // true when the param is an ObjC object (send .Ptr())
}

// buildNewConstructor generates a plain +new constructor: New<Type>() *Type.
func buildNewConstructor(_, goTypeName, _ string) ctorEntry {
	return ctorEntry{
		goName:        "New" + goTypeName,
		rawInitGoName: "", // signals +new
	}
}

// buildParamConstructor generates a constructor from an init method with parameters.
func buildParamConstructor(
	method meta.Method,
	goTypeName string,
	_ meta.Class,
	fc *frameworkContext,
	ctx typemap.Context,
	m *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	_ abstractBaseIndex,
) *ctorEntry {
	rawInit := naming.MethodName(method.Selector)
	extraImports := map[string]string{}

	var params []ctorParam
	for i, p := range method.Params {
		pName := safeParamName(naming.ParamName(p.Name))
		if pName == "" {
			pName = fmt.Sprintf("arg%d", i)
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
			// A parameter that cannot yet be expressed without an Objective-C
			// runtime type; drop this constructor rather than leak one.
			return nil
		}
		maps.Copy(extraImports, imps)
		params = append(params, ctorParam{goName: pName, goType: sig, rawExpr: argExpr})
	}

	return &ctorEntry{
		goName:          applyInitialisms("New" + goTypeName + strings.TrimPrefix(rawInit, "Init")),
		rawInitGoName:   rawInit,
		rawInitSelector: method.Selector,
		params:          params,
		hasNSError:      method.IsNSError,
		extraImports:    extraImports,
	}
}

func writeConstructor(
	w io.Writer,
	goTypeName, rawPkgAlias, className, genericArgs string,
	c ctorEntry,
) {
	_ = rawPkgAlias
	_ = genericArgs
	adoptFn := adoptHelperName(goTypeName)

	paramParts := make([]string, 0, len(c.params))
	for _, p := range c.params {
		paramParts = append(paramParts, p.goName+" "+p.goType)
	}

	retStr := "*" + goTypeName
	if c.hasNSError {
		// Two return values → named returns (E7).
		retStr = "(result *" + goTypeName + ", err error)"
	}

	view := constructorView{
		DocComment: docLead(c.goName, c.doc, "creates a new "+goTypeName+"."),
		GoName:     c.goName,
		GoTypeName: goTypeName,
		ParamStr:   strings.Join(paramParts, ", "),
		RetStr:     retStr,
		ClassName:  className,
		AdoptFn:    adoptFn,
		Preamble:   c.preamble,
		IsPlainNew: c.rawInitGoName == "",
		HasNSError: c.hasNSError,
	}
	if !view.IsPlainNew {
		// alloc + send the init selector directly via objc.Send[objc.ID],
		// bypassing the raw init method's return type (objc.ID or *T).
		sendArgs := make([]string, 0, len(c.params))
		for _, p := range c.params {
			sendArgs = append(sendArgs, ctorParamSendExpr(p))
		}
		allArgs := append(
			[]string{"_alloc", fmt.Sprintf("objc.RegisterName(%q)", c.rawInitSelector)},
			sendArgs...,
		)
		if c.hasNSError {
			allArgs = append(allArgs, "unsafe.Pointer(&_nsErr)")
		}
		view.SendAllArgs = strings.Join(allArgs, ", ")
	}

	renderTemplate(w, "constructor", view)
}

// constructorView is the template data for constructor.tmpl.
type constructorView struct {
	DocComment  string
	GoName      string
	GoTypeName  string
	ParamStr    string
	RetStr      string
	ClassName   string
	AdoptFn     string // helper that wraps the freshly created object, e.g. "stringAdopt"
	Preamble    string
	IsPlainNew  bool
	HasNSError  bool
	SendAllArgs string // alloc-init path: `_alloc, objc.RegisterName("sel"), …`
}

// ctorParamSendExpr returns the expression for passing a constructor parameter
// to the Objective-C init call. The parameter's value is already converted to an
// Objective-C-compatible form, so it is passed unchanged.
func ctorParamSendExpr(p ctorParam) string {
	return p.rawExpr
}

// isObjectPointerType reports whether a resolved Go type is a single pointer to
// an ObjC class (and therefore has a Ptr() method). Pointers to primitives or C
// structs (e.g. *uint8, *raw.IOUSBHostCIMessage) are not objects.
func isObjectPointerType(goType string, m *typemap.Mapper) bool {
	if !strings.HasPrefix(goType, "*") {
		return false
	}
	base := strings.TrimPrefix(goType, "*")
	if strings.HasPrefix(base, "*") {
		return false // double pointer (out-param)
	}
	if idx := strings.IndexByte(base, '['); idx >= 0 {
		base = base[:idx]
	}
	if idx := strings.LastIndexByte(base, '.'); idx >= 0 {
		base = base[idx+1:]
	}
	_, ok := m.OwnerIndex[base]
	return ok
}
