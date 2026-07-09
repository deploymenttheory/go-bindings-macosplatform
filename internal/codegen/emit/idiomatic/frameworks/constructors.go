//go:build darwin

package idiofw

import (
	"fmt"
	"io"
	"maps"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// buildConstructors assembles the constructor entries for a class: one per
// parameterised init, plus a plain +new constructor when appropriate.
func buildConstructors(
	class meta.Class,
	className, goTypeName string,
	fc *frameworkContext,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	abstractBases abstractBaseIndex,
) []constructorModel {
	result := buildConstructorList(
		class,
		className,
		goTypeName,
		fc,
		ctx,
		mapper,
		rawPkgAlias,
		trialNames,
		abstractBases,
	)
	// A constructor of a @MainActor class allocates and initialises a UI object,
	// which must happen on the main thread; the template wraps it in purego.Main.
	if class.IsMainThreadRequired {
		for i := range result {
			result[i].mainThread = true
			if result[i].extraImports == nil {
				result[i].extraImports = map[string]string{}
			}
			result[i].extraImports["purego"] = pureobjcImportPath
		}
	}
	return result
}

func buildConstructorList(
	class meta.Class,
	className, goTypeName string,
	fc *frameworkContext,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	abstractBases abstractBaseIndex,
) []constructorModel {
	var ctors []constructorModel
	hasExplicitParamInit := false

	for _, method := range class.Methods {
		if method.Availability.IsUnavailable || method.IsClassMethod || !method.IsInit {
			continue
		}
		if len(method.Params) == 0 {
			continue // plain init handled below as +new fallback
		}
		built := buildParamConstructor(
			method,
			goTypeName,
			class,
			fc,
			ctx,
			mapper,
			rawPkgAlias,
			trialNames,
			abstractBases,
		)
		if built != nil {
			built.doc = method.Doc
			ctors = append(ctors, *built)
			hasExplicitParamInit = true
		}
	}

	// Every class gets a plain +new constructor unless it has a param-init that
	// already covers the no-arg case.
	if !hasExplicitParamInit {
		ctors = append(
			[]constructorModel{buildNewConstructor(className, goTypeName, rawPkgAlias)},
			ctors...)
		repairConstructorNames(ctors)
		return ctors
	}
	// Still provide a plain constructor if there is a class-specific plain init too.
	for _, method := range class.Methods {
		if method.IsInit && len(method.Params) == 0 && !method.Availability.IsUnavailable {
			ctors = append(
				[]constructorModel{buildNewConstructor(className, goTypeName, rawPkgAlias)},
				ctors...)
			break
		}
	}
	repairConstructorNames(ctors)
	return ctors
}

// repairConstructorNames drops the trailing error label from an error-returning
// constructor's name — the (result, err) signature already says it
// (NewStringWithContentsOfFileEncodingError → NewStringWithContentsOfFileEncoding).
// Same free-name discipline as repairMethodNames: a strip applies only when the
// shorter name is a free exported identifier within this class's constructor
// set, and curated sidecar names are never touched.
func repairConstructorNames(ctors []constructorModel) {
	taken := make(map[string]bool, len(ctors))
	for _, c := range ctors {
		taken[c.goName] = true
	}
	for i := range ctors {
		if !ctors[i].hasNSError || ctors[i].nameCurated {
			continue
		}
		for _, suffix := range []string{"AndReturnError", "WithError", "Error"} {
			stripped, ok := strings.CutSuffix(ctors[i].goName, suffix)
			if !ok {
				continue
			}
			if stripped != "" && stripped[0] >= 'A' && stripped[0] <= 'Z' && !taken[stripped] {
				taken[stripped] = true
				ctors[i].goName = stripped
			}
			break
		}
	}
}

type constructorModel struct {
	goName string
	// nameCurated marks a goName assigned by the idiomatic.json sidecar; the
	// error-label repair pass leaves such a name untouched.
	nameCurated     bool
	doc             string // Apple/header documentation for the underlying init
	rawInitGoName   string // "" = use +new
	rawInitSelector string // ObjC selector, e.g. "initWithURL:readOnly:error:"
	params          []constructorParamModel
	hasNSError      bool
	needsFoundation bool
	mainThread      bool // run alloc/init on the main thread (@MainActor class)
	extraImports    map[string]string
	// preamble is rendered verbatim at the top of the constructor body, before
	// alloc/init — used to build a raw NSArray from a variadic ergonomic param.
	preamble string
}

type constructorParamModel struct {
	goName        string
	goType        string
	rawExpression string // expression passed to the raw init method
	isObject      bool   // true when the param is an ObjC object (send .Ptr())
}

// buildNewConstructor generates a plain +new constructor: New<Type>() *Type.
func buildNewConstructor(_, goTypeName, _ string) constructorModel {
	return constructorModel{
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
	mapper *typemap.Mapper,
	rawPkgAlias string,
	trialNames trialNameMap,
	_ abstractBaseIndex,
) *constructorModel {
	rawInit := naming.MethodName(method.Selector)
	extraImports := map[string]string{}

	var params []constructorParamModel
	for i, param := range method.Params {
		pName := safeParamName(naming.ParamName(param.Name))
		if pName == "" {
			pName = fmt.Sprintf("arg%d", i)
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
			// A parameter that cannot yet be expressed without an Objective-C
			// runtime type; drop this constructor rather than leak one.
			return nil
		}
		maps.Copy(extraImports, imports)
		if strings.HasPrefix(argExpr, "objref.IDOf(") {
			// The wrapper argument's pointer is read before the init send; keep
			// the wrapper alive across the call (defer runtime.KeepAlive).
			extraImports["runtime"] = "runtime"
		}
		params = append(
			params,
			constructorParamModel{goName: pName, goType: sig, rawExpression: argExpr},
		)
	}

	goName := applyInitialisms("New" + goTypeName + strings.TrimPrefix(rawInit, "Init"))
	nameCurated := false
	if curated, ok := fc.idio.MethodGoName(ctx.ClassName, method.Selector, false); ok {
		goName, nameCurated = curated, true
	}
	return &constructorModel{
		goName:          goName,
		nameCurated:     nameCurated,
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
	c constructorModel,
) {
	_ = rawPkgAlias
	_ = genericArgs
	adoptFn := adoptHelperName(goTypeName)

	paramParts := make([]string, 0, len(c.params))
	for _, param := range c.params {
		paramParts = append(paramParts, param.goName+" "+param.goType)
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
		MainThread: c.mainThread,
	}
	for _, param := range c.params {
		// A wrapper argument passed by pointer must outlive the init send
		// (defer runtime.KeepAlive); collections retain their elements.
		if strings.HasPrefix(param.rawExpression, "objref.IDOf(") {
			view.KeepAlive = append(view.KeepAlive, param.goName)
		}
	}
	if !view.IsPlainNew {
		// alloc + send the init selector directly via objc.Send[objc.ID],
		// bypassing the raw init method's return type (objc.ID or *T).
		sendArgs := make([]string, 0, len(c.params))
		for _, param := range c.params {
			sendArgs = append(sendArgs, ctorParamSendExpr(param))
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

	render.Must(w, "constructor", view)
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
	MainThread  bool   // wrap alloc/init in purego.Main (@MainActor class)
	SendAllArgs string // alloc-init path: `_alloc, objc.RegisterName("sel"), …`
	// KeepAlive names wrapper arguments kept alive until the constructor
	// returns (defer runtime.KeepAlive) so the collector cannot finalize them
	// mid-send.
	KeepAlive []string
}

// ctorParamSendExpr returns the expression for passing a constructor parameter
// to the Objective-C init call. The parameter's value is already converted to an
// Objective-C-compatible form, so it is passed unchanged.
func ctorParamSendExpr(param constructorParamModel) string {
	return param.rawExpression
}

// isObjectPointerType reports whether a resolved Go type is a single pointer to
// an ObjC class (and therefore has a Ptr() method). Pointers to primitives or C
// structs (e.g. *uint8, *raw.IOUSBHostCIMessage) are not objects.
func isObjectPointerType(goType string, mapper *typemap.Mapper) bool {
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
	_, ok := mapper.OwnerIndex[base]
	return ok
}
