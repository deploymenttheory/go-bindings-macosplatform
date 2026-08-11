package rawlib

import (
	"fmt"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/view"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/scanner"
)

// Protocols writes a complete _protocols.go file with Go interface definitions
// for all ObjC protocols in the framework.
func EmitProtocols(w io.Writer, pkgName string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, knownProtocols map[string]string, allClasses map[string]macosplatformmetadata.Class) error {
	model := buildProtocolsModel(pkgName, framework, m, knownClasses, knownProtocols, allClasses)
	return render.Execute(w, "protocols_file", model)
}

// buildProtocolsModel constructs the complete file model for the _protocols.go template.
// Import collection is a side effect of type-mapping method signatures, so it must
// run before the template is executed.
func buildProtocolsModel(pkgName string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, knownProtocols map[string]string, allClasses map[string]macosplatformmetadata.Class) view.ProtocolsFileModel {
	usedImports := make(typemap.ImportSet)
	ctx := m.BaseContext(framework.Framework, knownClasses)

	names := sortedStringKeys(framework.Protocols)
	protocols := make([]view.ProtocolModel, 0, len(names))

	for _, name := range names {
		p := framework.Protocols[name]
		if p.Availability.IsUnavailable {
			continue
		}
		pm := buildProtocolModel(name, p, framework, m, knownClasses, knownProtocols, allClasses, ctx, usedImports)
		protocols = append(protocols, pm)
	}

	// The purego backend must not import runtime/cgo (it pulls the cgo runtime
	// into a CGO_ENABLED=0 package); embed the pure-Go objptr.Object instead —
	// the same type cgo.Object aliases.
	purego := scanner.CLibraryBackend(framework.Framework) == scanner.BackendPurego
	rootObject := "cgo.Object"
	if purego {
		rootObject = "objptr.Object"
	}
	imports := buildProtocolsImports(usedImports, purego)
	return view.ProtocolsFileModel{PkgName: pkgName, Imports: imports, RootObject: rootObject, Protocols: protocols}
}

// buildProtocolModel builds the model for a single ObjC @protocol → Go interface.
func buildProtocolModel(name string, p macosplatformmetadata.Protocol, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, knownProtocols map[string]string, allClasses map[string]macosplatformmetadata.Class, ctx typemap.Context, usedImports typemap.ImportSet) view.ProtocolModel {
	goName := naming.ProtocolGoTypeName(name, m.OwnerIndex)

	// Resolve embedded parent protocols.
	var embeds []string
	var embedComments []string
	blocked := m.BlockedImports[framework.Framework]

	for _, impl := range p.InheritedProtocols {
		if _, ok := framework.Protocols[impl]; ok {
			embeds = append(embeds, naming.ProtocolGoTypeName(impl, m.OwnerIndex))
			continue
		}
		ownerFW := knownProtocols[impl]
		if ownerFW == "" {
			continue
		}
		ownerPkg := strings.ToLower(ownerFW)
		implGoName := naming.ProtocolGoTypeName(impl, m.OwnerIndex)
		if blocked[ownerFW] {
			embedComments = append(embedComments, fmt.Sprintf(
				"// implements %s.%s (import blocked: cycle between %s and %s)",
				ownerPkg, implGoName,
				strings.ToLower(framework.Framework), ownerPkg))
			continue
		}
		if importPath, ok := usedImports[ownerPkg]; !ok || importPath == "" {
			importPath = m.ModulePrefix + "/" + ownerPkg
			usedImports[ownerPkg] = importPath
		}
		embeds = append(embeds, ownerPkg+"."+implGoName)
	}

	// Build method signatures.
	seenMethods := make(map[string]bool)
	methods := make([]view.ProtocolMethodModel, 0, len(p.Methods))

	for _, method := range p.Methods {
		if shouldSkipBridgeMethod(method) {
			continue
		}
		if methodRefsUnavailableClass(method, framework, allClasses) {
			continue
		}
		goMethodName := naming.MethodName(method.Selector)
		if seenMethods[goMethodName] {
			continue
		}
		seenMethods[goMethodName] = true

		args := buildGoArgs(method.Params, method.IsNSError, ctx, m, usedImports)
		ret := buildGoReturn(method, ctx, m, "", usedImports)
		methods = append(methods, view.ProtocolMethodModel{
			GoName: goMethodName,
			Params: strings.Join(args, ", "),
			Ret:    ret,
		})
	}

	return view.ProtocolModel{
		GoName:        goName,
		ObjCName:      name,
		AvailComment:  availabilityComment(p.Availability),
		Embeds:        embeds,
		EmbedComments: embedComments,
		Methods:       methods,
	}
}

// buildProtocolsImports assembles and sorts the import list for a protocols file.
func buildProtocolsImports(usedImports typemap.ImportSet, purego bool) []string {
	rootPkg := "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/cgo"
	if purego {
		rootPkg = "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/objptr"
	}
	set := map[string]bool{
		"unsafe": true,
		rootPkg:  true,
	}
	for _, path := range usedImports {
		set[path] = true
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
