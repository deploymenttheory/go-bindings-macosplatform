package rawlib

import (
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/view"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// EmitClassInterfaces writes one [ClassName]able interface per concrete ObjC class,
// modelled on the Kiota [Type]able pattern. Each interface lists all instance methods
// with identical signatures to those emitted on *ClassName, enabling mock
// implementations for testing and polymorphic acceptance of compatible types.
//
// Generic classes (e.g. NSArray[T]) are skipped in this pass — their interface
// signatures require type parameters that complicate the embedding chain.
func EmitClassInterfaces(w io.Writer, pkgName string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, allClasses map[string]macosplatformmetadata.Class) error {
	if len(framework.Classes) == 0 {
		return nil
	}

	usedImports := make(typemap.ImportSet)
	ctx := m.BaseContext(framework.Framework, knownClasses)

	names := make([]string, 0, len(framework.Classes))
	for k := range framework.Classes {
		names = append(names, k)
	}
	sort.Strings(names)

	// Phase 1: build models, collecting cross-framework imports as a side effect.
	var interfaces []view.ClassInterfaceModel
	for _, name := range names {
		cls := framework.Classes[name]
		if cls.Availability.IsUnavailable {
			continue
		}
		if len(cls.GenericParams) > 0 {
			continue // generic classes deferred — interface signatures involve type params
		}
		// Skip if a real ObjC protocol with the same "able" name already exists
		// in this framework (e.g. protocol SCNActionable vs class SCNAction).
		// The protocol interface takes precedence; emitting both would cause a
		// redeclaration error.
		if _, hasProto := framework.Protocols[name+"able"]; hasProto {
			continue
		}
		interfaces = append(interfaces, buildClassInterfaceModel(name, cls, framework, m, ctx, allClasses, usedImports))
	}

	if len(interfaces) == 0 {
		return nil
	}

	// Phase 2: compute which imports are needed from the collected models.
	usesUnsafe := false
	usesObjc := false
	for _, iface := range interfaces {
		if strings.Contains(iface.EmbedLine, "cgo.") {
			usesObjc = true
		}
		for _, method := range iface.Methods {
			if strings.Contains(method.Params, "unsafe.") || strings.Contains(method.Ret, "unsafe.") {
				usesUnsafe = true
			}
			if strings.Contains(method.Params, "cgo.") || strings.Contains(method.Ret, "cgo.") {
				usesObjc = true
			}
		}
	}

	// Phase 3: assemble the file model and render via template.
	var allImports []string
	if usesUnsafe {
		allImports = append(allImports, "unsafe")
	}
	if usesObjc {
		allImports = append(allImports, "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/cgo")
	}
	for _, path := range usedImports {
		allImports = append(allImports, path)
	}
	sort.Strings(allImports)
	seen := map[string]bool{}
	var deduped []string
	for _, imp := range allImports {
		if !seen[imp] {
			seen[imp] = true
			deduped = append(deduped, imp)
		}
	}

	return render.Execute(w, "interfaces_file", view.ClassInterfacesFileModel{
		PkgName:    pkgName,
		Imports:    deduped,
		UsesUnsafe: usesUnsafe,
		UsesObjc:   usesObjc,
		Interfaces: interfaces,
	})
}

// buildClassInterfaceModel constructs a view.ClassInterfaceModel for a single ObjC class.
// Cross-framework embed imports are recorded in usedImports as a side effect.
func buildClassInterfaceModel(name string, cls macosplatformmetadata.Class, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, ctx typemap.Context, allClasses map[string]macosplatformmetadata.Class, usedImports typemap.ImportSet) view.ClassInterfaceModel {
	si := classifySuper(name, cls, framework, m)

	iCtx := ctx
	iCtx.ClassName = name
	iCtx.GenericParams = cls.GenericParams
	iCtx.IsClassMethod = false

	// Determine the embed line for the interface body.
	var embedLine string
	switch {
	case si.isRoot || si.superIsGeneric:
		embedLine = "cgo.Object"
	case si.pkg != "":
		if si.importPath != "" {
			usedImports[si.pkg] = si.importPath
		}
		embedLine = si.pkg + "." + si.name + "able"
	default:
		embedLine = si.name + "able"
	}

	// Collect Go method names promoted via the embedding chain to detect
	// selector-to-name collisions (different selectors that map to the same
	// Go method name). Selector-level inheritance filtering is done upstream
	// by filterNovelMethods in the loader.
	inheritedGoNames := make(map[string]bool)
	for super := cls.Super; super != ""; {
		parent, ok := allClasses[super]
		if !ok {
			break
		}
		parentGoNameCount := make(map[string]int)
		parentCounted := make(map[string]bool)
		for _, mm := range parent.Methods {
			if mm.IsClassMethod || shouldSkipBridgeMethod(mm) {
				continue
			}
			if mm.Availability.IsUnavailable || methodRefsUnavailableClass(mm, framework, allClasses) {
				continue
			}
			mk := methodKey(mm.Selector, mm.IsClassMethod)
			if parentCounted[mk] {
				continue
			}
			parentCounted[mk] = true
			parentGoNameCount[naming.MethodName(mm.Selector)]++
		}
		for _, mm := range parent.Methods {
			if mm.IsClassMethod || mm.IsInit {
				continue
			}
			if shouldSkipBridgeMethod(mm) {
				continue
			}
			if mm.Availability.IsUnavailable || methodRefsUnavailableClass(mm, framework, allClasses) {
				continue
			}
			gn := naming.MethodName(mm.Selector)
			resolved, skip := naming.ResolveGoMethodName(gn, mm, parent.Methods, parentGoNameCount)
			if !skip {
				inheritedGoNames[resolved] = true
			}
		}
		super = parent.Super
	}

	// Build per-kind name counts. Class methods and instance methods live in
	// separate Go namespaces: class methods become ClassName-prefixed package-
	// level funcs; instance methods become methods on *ClassName. A shared count
	// map was the root cause of the _2/_3 suffix bug: a class method would
	// "consume" the base name and force the instance method to get a spurious
	// suffix even though the two names never actually collide in Go.
	instanceGoNameCount := make(map[string]int)
	classGoNameCount := make(map[string]int)
	countedKeys := make(map[string]bool)
	for _, method := range cls.Methods {
		if shouldSkipBridgeMethod(method) {
			continue
		}
		if method.Availability.IsUnavailable || methodRefsUnavailableClass(method, framework, allClasses) {
			continue
		}
		mk := methodKey(method.Selector, method.IsClassMethod)
		if countedKeys[mk] {
			continue
		}
		countedKeys[mk] = true
		if method.IsClassMethod {
			classGoNameCount[naming.MethodName(method.Selector)]++
		} else {
			instanceGoNameCount[naming.MethodName(method.Selector)]++
		}
	}

	// Build the method list using the same per-kind disambiguation logic as
	// EmitClass so that interface method names always match the concrete type.
	var methods []view.InterfaceMethodModel
	seenMethodKeys := make(map[string]bool)
	for _, method := range cls.Methods {
		if shouldSkipBridgeMethod(method) {
			continue
		}
		if method.Availability.IsUnavailable || methodRefsUnavailableClass(method, framework, allClasses) {
			continue
		}
		mk := methodKey(method.Selector, method.IsClassMethod)
		if seenMethodKeys[mk] {
			continue
		}
		seenMethodKeys[mk] = true

		gn := naming.MethodName(method.Selector)
		goNameCount := instanceGoNameCount
		if method.IsClassMethod {
			goNameCount = classGoNameCount
		}
		if goNameCount[gn] > 1 {
			resolved, skip := naming.ResolveGoMethodName(gn, method, cls.Methods, goNameCount)
			if skip {
				continue
			}
			gn = resolved
		}

		// Class methods are not emitted in the interface.
		if method.IsClassMethod {
			continue
		}

		// Initializers return instancetype, which becomes the concrete subclass
		// type. Subclasses declare different return types for the same selector
		// ("init", "initWithCoder:", etc.) so they cannot participate in
		// interface embedding — Go interfaces don't allow covariant overrides.
		if method.IsInit {
			continue
		}

		// Skip methods where a different ancestor selector maps to the same Go
		// name — re-declaring in the interface would produce a duplicate method error.
		if inheritedGoNames[gn] {
			continue
		}

		args := buildGoArgs(method.Params, method.IsNSError, iCtx, m, usedImports)
		ret := buildGoReturn(method, iCtx, m, name, usedImports)

		methods = append(methods, view.InterfaceMethodModel{
			GoName: gn,
			Params: strings.Join(args, ", "),
			Ret:    ret,
		})
	}

	return view.ClassInterfaceModel{
		GoName:      name,
		EmbedLine:   embedLine,
		Methods:     methods,
		HasNSCoding: classConformsToCoding(cls),
	}
}
