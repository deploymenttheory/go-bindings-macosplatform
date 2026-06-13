package raw

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// Protocols writes a complete _protocols.go file with Go interface definitions
// for all ObjC protocols in the framework.
func EmitProtocols(w io.Writer, pkgName string, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, knownProtocols map[string]string, allClasses map[string]meta.Class) error {
	model := buildProtocolsModel(pkgName, framework, m, knownClasses, knownProtocols, allClasses)
	return executeTemplate(w, "protocols_file", model)
}

// buildProtocolsModel constructs the complete file model for the _protocols.go template.
// Import collection is a side effect of type-mapping method signatures, so it must
// run before the template is executed.
func buildProtocolsModel(pkgName string, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, knownProtocols map[string]string, allClasses map[string]meta.Class) protocolsFileModel {
	usedImports := make(typemap.ImportSet)
	ctx := m.BaseContext(framework.Framework, knownClasses)

	names := sortedStringKeys(framework.Protocols)
	protocols := make([]protocolModel, 0, len(names))
	needsContext := false

	for _, name := range names {
		p := framework.Protocols[name]
		if p.Availability.IsUnavailable {
			continue
		}
		pm, usesCtx := buildProtocolModel(name, p, framework, m, knownClasses, knownProtocols, allClasses, ctx, usedImports)
		if usesCtx {
			needsContext = true
		}
		protocols = append(protocols, pm)
	}

	imports := buildProtocolsImports(usedImports, needsContext)
	return protocolsFileModel{PkgName: pkgName, Imports: imports, Protocols: protocols}
}

// buildProtocolModel builds the model for a single ObjC @protocol → Go interface.
// It also reports whether any method uses context.Context (for import decisions).
func buildProtocolModel(name string, p meta.Protocol, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, knownProtocols map[string]string, allClasses map[string]meta.Class, ctx typemap.Context, usedImports typemap.ImportSet) (protocolModel, bool) {
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
	methods := make([]protocolMethodModel, 0, len(p.Methods))
	usesCtx := false

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
		usesCtx = true

		args := append([]string{"ctx context.Context"}, buildGoArgs(method.Params, method.IsNSError, ctx, m, usedImports)...)
		ret := buildGoReturn(method, ctx, m, "", usedImports)
		methods = append(methods, protocolMethodModel{
			GoName: goMethodName,
			Params: strings.Join(args, ", "),
			Ret:    ret,
		})
	}

	return protocolModel{
		GoName:        goName,
		ObjCName:      name,
		AvailComment:  availabilityComment(p.Availability),
		Embeds:        embeds,
		EmbedComments: embedComments,
		Methods:       methods,
	}, usesCtx
}

// buildProtocolsImports assembles and sorts the import list for a protocols file.
func buildProtocolsImports(usedImports typemap.ImportSet, needsContext bool) []string {
	set := map[string]bool{
		"unsafe": true,
		"github.com/deploymenttheory/go-bindings-macosplatform/internal/objc": true,
	}
	if needsContext {
		set["context"] = true
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
