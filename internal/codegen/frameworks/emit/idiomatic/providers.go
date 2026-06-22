//go:build darwin

package idiomatic

import (
	"bytes"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// providerMethodEntry describes a provider method to be emitted on a trial wrapper type.
type providerMethodEntry struct {
	methodName string
	rawType    string
	bodyExpr   string
}

// providerMethodItem / providerMethodView are the template data for
// provider_method.tmpl.
type providerMethodItem struct {
	MethodName string
	RawType    string
	BodyExpr   string
}

type providerMethodView struct {
	GoTypeName string
	Items      []providerMethodItem
}

// buildProviderMethods assembles the provider interface implementations for a
// class: one for itself when it is an abstract base, and one per abstract base
// ancestor (returning the embedded raw superclass field).
func buildProviderMethods(
	cls meta.Class,
	className, goTypeName string,
	fw *meta.FrameworkMeta,
	m *typemap.Mapper,
	rawPkgAlias string,
	abstractBases abstractBaseIndex,
) ([]providerMethodEntry, typemap.ImportSet) {
	// A provider interface is satisfied by any wrapper for the right kind of
	// object simply by being an object (the interface embeds objref.Object), so
	// no per-wrapper accessor method is emitted.
	_ = cls
	_ = className
	_ = goTypeName
	_ = fw
	_ = m
	_ = rawPkgAlias
	_ = abstractBases
	return nil, make(typemap.ImportSet)
}

// genericInstantiation returns the type-argument list used to instantiate a
// generic raw class in trial code, e.g. "[objc.ID]" or "[objc.ID, objc.ID]".
// objc.ID matches the raw emitter's own degradation default for generics.
func genericInstantiation(n int) string {
	if n == 0 {
		return ""
	}
	args := make([]string, n)
	for i := range args {
		args[i] = "objc.ID"
	}
	return "[" + strings.Join(args, ", ") + "]"
}

// emitDictionaryAugment adds an object-pointer builder to the generated
// NSMutableDictionary wrapper. The generated setObject:forKey: types the key as
// the NSCopying interface, which a bare objc.ID (e.g. the CFStringRef constant
// security.KSecClass()) does not satisfy, so building such a dictionary
// otherwise still needs a manual message send. Set takes objc.ID key and value
// directly. (Reading a dictionary result back is covered by the generic
// <Type>FromID constructor plus ObjectForKey.)
func emitDictionaryAugment(w io.Writer, className, goTypeName, _, _ string) {
	if className != "NSMutableDictionary" {
		return
	}
	renderTemplate(w, "dict_augment", struct {
		GoTypeName string
		RecvVar    string
	}{GoTypeName: goTypeName, RecvVar: receiverName(goTypeName)})
}

// providerIfaceItem / providersView are the template data for providers.tmpl.
type providerIfaceItem struct {
	IfaceName  string
	GoTypeName string
	BaseName   string
	MethName   string
	RawType    string
	// MarkerName is the unexported sealing method the interface requires, so only
	// the base class and its subclasses (which promote it) satisfy the interface.
	MarkerName string
}

type providersView struct {
	Items []providerIfaceItem
}

// emitProvidersFile writes <pkgname>_providers_generated.go containing one provider
// interface per abstract base class in the framework.
func emitProvidersFile(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	abstractBases abstractBaseIndex,
	fw *meta.FrameworkMeta,
	m *typemap.Mapper,
) error {
	if len(abstractBases) == 0 {
		return nil
	}

	baseNames := make([]string, 0, len(abstractBases))
	for name := range abstractBases {
		baseNames = append(baseNames, name)
	}
	sort.Strings(baseNames)

	_ = rawPkgAlias
	_ = fw
	_ = m
	var view providersView
	for _, baseName := range baseNames {
		goTypeName := abstractBases[baseName]
		view.Items = append(view.Items, providerIfaceItem{
			IfaceName:  providerInterfaceName(goTypeName),
			GoTypeName: goTypeName,
			BaseName:   baseName,
			MarkerName: markerMethodName(goTypeName),
		})
	}
	if len(view.Items) == 0 {
		return nil
	}
	var body bytes.Buffer
	if err := executeTemplate(&body, "providers", view); err != nil {
		return err
	}

	// Every provider interface embeds objref.Object, and the function returned
	// early when there were no items, so objref is always (and only) used.
	imports := map[string]string{"objref": objrefImportPath}
	_ = rawPkgPath

	fname := pkgName + "_providers_generated.go"
	file := assembleFile(pkgName, imports, body.Bytes())
	return emit.WriteGoFile(filepath.Join(outDir, fname), file)
}
