package rawlib

import (
	"bytes"
	"fmt"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/view"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
	"github.com/deploymenttheory/go-bindings-macosplatform/metadata/objcclasshierarchy"
)

// Classes writes one Go source file per ObjC class into outDir.
// allClasses is the combined class map from all frameworks, used for building
// cross-framework embedding chains and constructors.
func EmitClasses(
	outDir string,
	framework *macosplatformmetadata.FrameworkMeta,
	m *typemap.Mapper,
	knownClasses map[string]bool,
	allClasses map[string]macosplatformmetadata.Class,
	packageName string,
) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	names := make([]string, 0, len(framework.Classes))
	for k := range framework.Classes {
		names = append(names, k)
	}
	sort.Strings(names)

	// Track used filenames (lowercase) to detect case-insensitive collisions on
	// macOS APFS volumes. When two ObjC class names differ only in case (e.g.,
	// MTROTASoftware... vs MTROtaSoftware...), the second class gets a _2 suffix.
	usedFilenames := make(map[string]bool)

	for _, name := range names {
		base := name + ".go"
		if usedFilenames[strings.ToLower(base)] {
			i := 2
			for {
				base = fmt.Sprintf("%s_%d.go", name, i)
				if !usedFilenames[strings.ToLower(base)] {
					break
				}
				i++
			}
		}
		usedFilenames[strings.ToLower(base)] = true

		cls := framework.Classes[name]
		if cls.Availability.IsUnavailable {
			continue
		}
		path := filepath.Join(outDir, base)
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("create %s: %w", path, err)
		}
		if err := EmitClass(
			f,
			name,
			cls,
			framework,
			m,
			knownClasses,
			allClasses,
			packageName,
		); err != nil {
			f.Close()
			return fmt.Errorf("write class %s: %w", name, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close %s: %w", path, err)
		}
	}
	return nil
}

// WriteClass writes a single ObjC class as a Go source file using struct embedding
// for inheritance. Root classes (no superclass or unknown superclass) retain a plain
// `ptr unsafe.Pointer` field. Non-root classes embed their immediate superclass by
// value; Go's method promotion gives access to all ancestor methods.
func EmitClass(
	w io.Writer,
	name string,
	cls macosplatformmetadata.Class,
	framework *macosplatformmetadata.FrameworkMeta,
	m *typemap.Mapper,
	knownClasses map[string]bool,
	allClasses map[string]macosplatformmetadata.Class,
	packageName string,
) error {
	isGeneric := len(cls.GenericParams) > 0
	receiver := receiverType(name, isGeneric)
	superInfo := classifySuper(name, cls, framework, m)

	// usedImports collects cross-framework imports discovered during type resolution.
	usedImports := make(typemap.ImportSet)
	ctx := m.BaseContext(framework.Framework, knownClasses)
	ctx.ClassName = name
	ctx.GenericParams = cls.GenericParams

	// If this class embeds a cross-framework super, that import is needed.
	if superInfo.pkg != "" && superInfo.importPath != "" {
		usedImports[superInfo.pkg] = superInfo.importPath
	}

	// Phase 1: generate body into a buffer (discovers additional imports as side effect).
	var body bytes.Buffer
	if err := writeStructDef(&body, name, cls, isGeneric, superInfo); err != nil {
		return err
	}
	if err := writeConstructors(
		&body,
		name,
		cls,
		isGeneric,
		superInfo,
		framework,
		allClasses,
		m,
		knownClasses,
		usedImports,
	); err != nil {
		return err
	}
	if isGeneric {
		if err := writeGenericHelper(
			&body,
			name,
			cls,
			superInfo,
			framework,
			allClasses,
			m,
			usedImports,
		); err != nil {
			return err
		}
	}
	// Collect package-level type names (enums, structs, typedefs) so we can
	// skip class methods that would generate a function name conflicting with them.
	pkgTypeNames := make(map[string]bool)
	for n := range framework.Enums {
		pkgTypeNames[n] = true
	}
	for n := range framework.Structs {
		pkgTypeNames[n] = true
	}
	for n := range framework.Typedefs {
		pkgTypeNames[n] = true
	}

	// Pre-scan methods to resolve two kinds of collision:
	//  1. Duplicate (selector, isClassMethod) pairs — can occur when a base class and
	//     a category both expose the same method; skip duplicates.
	//  2. Two different selectors that produce the same Go name on the same receiver
	//     (e.g. "open" and "open:") — disambiguate by appending the argument count.
	bridgeNames := buildClassBridgeNames(framework.Framework, name, cls.Methods)

	seenMethodKeys := make(map[string]bool)
	// Count instance and class methods separately: a class method +[Foo bar] becomes
	// FooBar (a package-level func) while an instance method -[Foo bar] becomes Bar
	// (a method on *Foo). They live in different Go namespaces and must not share a
	// dedup counter — otherwise a class method can force an instance method to get
	// a spurious _2 suffix even though there is no actual name collision.
	instanceGoNameCount := make(map[string]int)
	classGoNameCount := make(map[string]int)
	for _, method := range cls.Methods {
		if shouldSkipBridgeMethod(method) {
			continue
		}
		if method.Availability.IsUnavailable {
			continue
		}
		if methodRefsUnavailableClass(method, framework, allClasses) {
			continue
		}
		mk := methodKey(method.Selector, method.IsClassMethod)
		if seenMethodKeys[mk] {
			continue
		}
		seenMethodKeys[mk] = true
		if method.IsClassMethod {
			classGoNameCount[naming.MethodName(method.Selector)]++
		} else {
			instanceGoNameCount[naming.MethodName(method.Selector)]++
		}
	}

	// Collect Go method names promoted via the embedding chain so we can skip
	// child methods where a DIFFERENT selector maps to the same Go name
	// (e.g. NSObject's -setNeedsDisplay vs NSControl's -setNeedsDisplay:).
	// Selector-level inheritance is already handled upstream by filterNovelMethods
	// in the loader; this pass catches only the Go-name-collision edge case.
	inheritedGoNames := make(map[string]bool)
	for super := cls.Super; super != ""; {
		parent, ok := allClasses[super]
		if !ok {
			break
		}
		// Build the parent's name-count map to reproduce the exact disambiguated
		// Go names the parent's class file emits (e.g. "Foo" vs "Foo_2").
		parentGoNameCount := make(map[string]int)
		parentCounted := make(map[string]bool)
		for _, mm := range parent.Methods {
			if mm.IsClassMethod || shouldSkipBridgeMethod(mm) {
				continue
			}
			if mm.Availability.IsUnavailable ||
				methodRefsUnavailableClass(mm, framework, allClasses) {
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
			if mm.Availability.IsUnavailable ||
				methodRefsUnavailableClass(mm, framework, allClasses) {
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

	seenMethodKeys = make(map[string]bool) // reset for emit pass
	emittedInstanceGoNames := make(map[string]bool)
	emittedClassGoNames := make(map[string]bool)
	seenVariadicKeys := make(map[string]bool)
	needsBlocks := false // true when at least one method argument is a block type
	for _, method := range cls.Methods {
		if method.Availability.IsUnavailable {
			continue
		}
		if methodRefsUnavailableClass(method, framework, allClasses) {
			continue
		}
		if !method.IsClassMethod && !method.IsInit {
			if inheritedGoNames[naming.MethodName(method.Selector)] {
				continue
			}
		}
		// ObjC runtime lifecycle methods (+initialize, +load) are called automatically
		// by the runtime; bridging them explicitly causes duplicate initialization.
		if method.IsClassMethod && (method.Selector == "initialize" || method.Selector == "load") {
			vk := methodKey(method.Selector, method.IsClassMethod)
			if !seenVariadicKeys[vk] {
				seenVariadicKeys[vk] = true
				fmt.Fprintf(
					w,
					"// Class method +[%s %s] — not bridged (ObjC runtime lifecycle; called automatically by the runtime).\n",
					name,
					method.Selector,
				)
			}
			continue
		}
		if method.IsVariadic {
			if !isFormatStringVariadic(method) {
				// Nil-sentinel and other variadic patterns cannot be bridged via CGo.
				// Emit a comment stub so callers know the method exists.
				vk := methodKey(method.Selector, method.IsClassMethod)
				if !seenVariadicKeys[vk] {
					seenVariadicKeys[vk] = true
					if method.IsClassMethod {
						fmt.Fprintf(
							w,
							"// Variadic class method +[%s %s] — not bridged (CGo cannot express C variadic arguments).\n",
							name,
							method.Selector,
						)
					} else {
						fmt.Fprintf(
							w,
							"// Variadic method -[%s %s] — not bridged (CGo cannot express C variadic arguments).\n",
							name,
							method.Selector,
						)
					}
				}
				continue
			}
			// Format-string variadics: bridge with fixed named args only.
			// Pass a pre-formatted string; use fmt.Sprintf on the Go side.
		}
		mk := methodKey(method.Selector, method.IsClassMethod)
		if seenMethodKeys[mk] {
			continue // skip duplicate selector (e.g. re-declared in a category)
		}
		seenMethodKeys[mk] = true

		gn := naming.MethodName(method.Selector)
		// Use per-kind name counts so class-method collisions don't bleed into
		// instance-method dedup (they live in separate Go namespaces).
		goNameCount := instanceGoNameCount
		emittedGoNames := emittedInstanceGoNames
		if method.IsClassMethod {
			goNameCount = classGoNameCount
			emittedGoNames = emittedClassGoNames
		}
		if goNameCount[gn] > 1 {
			resolved, skip := naming.ResolveGoMethodName(gn, method, cls.Methods, goNameCount)
			if skip {
				// IBAction overload: emit a comment so the omission is discoverable,
				// then skip — the zero-arg form (already emitted) is the usable API.
				baseSel := strings.TrimSuffix(method.Selector, ":")
				fmt.Fprintf(
					&body,
					"// -%s omitted: IBAction form of -%s (adds `id sender` for Interface Builder wiring only; not useful in Go)\n\n",
					method.Selector,
					baseSel,
				)
				continue
			}
			gn = resolved
		}
		// Secondary dedup: two selectors with the same Go name AND the same arg count
		// (e.g. "DictionaryRepresentation" vs "dictionaryRepresentation") both land on
		// the same name. Append _2, _3, ... until unique.
		if emittedGoNames[gn] {
			suffix := 2
			for {
				candidate := fmt.Sprintf("%s_%d", gn, suffix)
				if !emittedGoNames[candidate] {
					gn = candidate
					break
				}
				suffix++
			}
		}
		emittedGoNames[gn] = true

		em := emittedMethod{
			method:      method,
			definingCls: name,
			cFuncName:   bridgeNames[mk],
			goName:      gn,
		}
		if err := writeMethod(
			&body,
			em,
			receiver,
			name,
			framework.Framework,
			ctx,
			m,
			framework.Classes,
			allClasses,
			pkgTypeNames,
			usedImports,
		); err != nil {
			return err
		}
		if methodHasBlockArgs(method.Params, m) {
			needsBlocks = true
		}
	}

	// NSCoding/NSSecureCoding convenience methods (non-generic classes only).
	if !isGeneric && (!cls.Availability.IsUnavailable) {
		if err := writeCodingMethods(&body, name, framework.Framework, cls); err != nil {
			return err
		}
	}

	// ObjC category methods from foreign ancestor classes.
	//
	// When this class's canonical direct parent is owned by a different framework,
	// this class is the first class in the current framework's embedding tree
	// above that foreign parent. ObjC category methods that the current framework
	// defines on any foreign ancestor must be emitted here as real Go instance
	// methods — they cannot be added to the foreign type, and subclasses of this
	// class inherit them via Go embedding promotion without re-declaration.
	if err := writeForeignAncestorExtensions(
		&body,
		name,
		receiver,
		framework,
		ctx,
		m,
		allClasses,
		pkgTypeNames,
		usedImports,
		emittedInstanceGoNames,
		&needsBlocks,
	); err != nil {
		return err
	}

	// Phase 2: write header with collected imports, then body.
	allImports := []string{
		"unsafe",
		"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/cgo",
	}
	if needsBlocks {
		allImports = append(
			allImports,
			"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/blocks",
		)
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

	hdr := view.ClassFileHeaderModel{
		Framework:    framework.Framework,
		PkgName:      packageName,
		BridgeHeader: strings.ToLower(framework.Framework) + "_bridge.h",
		Imports:      deduped,
	}
	if err := render.Execute(w, "class_file_header", hdr); err != nil {
		return err
	}
	_, err := w.Write(body.Bytes())
	return err
}

// superInfo describes how a class relates to its superclass for embedding purposes.
type superInfo struct {
	name           string // superclass name (e.g. "NSObject")
	pkg            string // Go package alias for cross-fw supers, "" for same-fw or root
	importPath     string // full import path for cross-fw supers
	isRoot         bool   // true when this class has no walkable superclass (IS the root)
	superIsGeneric bool   // true when the superclass is a parameterized generic type
}

// classifySuper determines whether the class's superclass is in the same framework,
// a different framework, or absent (making this class a root).
func classifySuper(
	className string,
	cls macosplatformmetadata.Class,
	framework *macosplatformmetadata.FrameworkMeta,
	m *typemap.Mapper,
) superInfo {
	if cls.Super == "" {
		return superInfo{isRoot: true}
	}
	owner := m.OwnerIndex[cls.Super]
	if owner == "" {
		// Super is unknown across all loaded frameworks — treat as root.
		return superInfo{isRoot: true}
	}
	// If importing the super's framework would create a cycle, treat as root.
	if m.BlockedImports[framework.Framework][owner] {
		return superInfo{isRoot: true}
	}
	superIsGeneric := m.GenericClasses[cls.Super]
	if strings.EqualFold(owner, framework.Framework) {
		// Same framework.
		return superInfo{name: cls.Super, superIsGeneric: superIsGeneric}
	}
	// Cross-framework super.
	pkg := strings.ToLower(owner)
	importPath := ""
	if m.ModulePrefix != "" {
		importPath = m.ModulePrefix + "/" + pkg
	}
	return superInfo{
		name:           cls.Super,
		pkg:            pkg,
		importPath:     importPath,
		superIsGeneric: superIsGeneric,
	}
}

// writeStructDef emits the Go struct definition.
// Root classes use a plain `ptr unsafe.Pointer` field and define Ptr().
// Non-root classes embed their immediate superclass by value; Ptr() is promoted.
func writeStructDef(
	w io.Writer,
	name string,
	cls macosplatformmetadata.Class,
	isGeneric bool,
	si superInfo,
) error {
	return render.Execute(w, "class_struct", buildClassStructModel(name, cls, isGeneric, si))
}

// buildClassStructModel resolves an ObjC class's Go struct declaration: the doc
// comment block, the (generic) type header, the embedded field (raw ptr for a
// root class or the superclass struct otherwise), and the receiver/assertion
// types used by the promoted Ptr() accessor and the cgo.Object assertion.
func buildClassStructModel(
	name string,
	cls macosplatformmetadata.Class,
	isGeneric bool,
	si superInfo,
) view.ClassStructModel {
	var comment strings.Builder
	fmt.Fprintf(&comment, "// %s wraps the Objective-C %s class.\n", name, name)
	if cls.Super != "" {
		fmt.Fprintf(&comment, "// Superclass: %s\n", cls.Super)
	}
	if len(cls.Protocols) > 0 {
		fmt.Fprintf(&comment, "// Protocols: %s\n", strings.Join(cls.Protocols, ", "))
	}
	if cls.SwiftName != "" {
		fmt.Fprintf(&comment, "// Swift name: %s\n", cls.SwiftName)
	}
	comment.WriteString(renderCommentBlock(cls.Doc, cls.SDKFile, cls.SDKLine, cls.Availability, ""))

	genericDecl := ""
	if isGeneric {
		genericDecl = "[T cgo.Object]"
	}

	assertType := name
	if isGeneric {
		assertType = name + "[cgo.Object]"
	}

	model := view.ClassStructModel{
		CommentBlock: comment.String(),
		TypeHeader:   name + genericDecl,
		IsRoot:       si.isRoot,
		AssertType:   assertType,
	}
	if si.isRoot {
		// Root: owns the ptr field and defines Ptr().
		model.EmbedLine = "ptr unsafe.Pointer"
		model.PtrReceiver = name
		if isGeneric {
			model.PtrReceiver = name + "[T]"
		}
		return model
	}

	// Non-root: embed the immediate superclass by value. Ptr() is promoted
	// through the embedding chain, so no separate definition is needed.
	embedField := si.name
	if si.pkg != "" {
		embedField = si.pkg + "." + si.name
	}
	// For generic superclasses, embed with the same type parameter when the child
	// is also generic (e.g. NSMutableArray[T] embeds NSArray[T]), or with the
	// concrete cgo.Object instantiation when the child is not generic.
	if si.superIsGeneric {
		if isGeneric {
			embedField += "[T]"
		} else {
			embedField += "[cgo.Object]"
		}
	}
	model.EmbedLine = embedField
	return model
}

// isGenericClass is a lightweight check — generic classes in ObjC are collection types
// whose names follow recognisable patterns. This avoids passing the full GenericClasses
// map deep into the struct emitter.
func isGenericClass(name string, genericClasses map[string]bool) bool {
	if genericClasses != nil {
		return genericClasses[name]
	}
	// Fallback heuristic: ObjC generic collection types (those with <ObjectType> params).
	// NSPredicate is intentionally excluded — it is not generic in ObjC.
	switch name {
	case "NSArray", "NSMutableArray", "NSSet", "NSMutableSet",
		"NSOrderedSet", "NSMutableOrderedSet", "NSDictionary", "NSMutableDictionary",
		"NSMapTable", "NSHashTable", "NSCache", "NSCountedSet",
		"NSEnumerator":
		return true
	}
	return false
}

// writeConstructors emits:
//  1. New<ClassName>(ptr) *<ClassName>  — full constructor with runtime tracking
//  2. <ClassName>WithPtr(ptr) <ClassName> — value constructor for embedding in subclasses
func writeConstructors(
	w io.Writer,
	name string,
	cls macosplatformmetadata.Class,
	isGeneric bool,
	si superInfo,
	framework *macosplatformmetadata.FrameworkMeta,
	allClasses map[string]macosplatformmetadata.Class,
	m *typemap.Mapper,
	knownClasses map[string]bool,
	usedImports typemap.ImportSet,
) error {
	genSuffix := ""
	if isGeneric {
		genSuffix = "[cgo.Object]"
	}

	chain := buildValueChain(
		name,
		genSuffix,
		framework.Framework,
		framework.Classes,
		m,
		usedImports,
	)
	if err := render.Execute(w, "class_constructors", view.ClassConstructorsModel{
		Name:       name,
		GenSuffix:  genSuffix,
		Chain:      chain,
		ValueChain: strings.TrimPrefix(chain, "&"), // strip the leading & to get the value
	}); err != nil {
		return err
	}

	{
		ctorCtx := m.BaseContext(framework.Framework, knownClasses)
		ctorCtx.ClassName = name
		if err := writeDesignatedInitConstructors(
			w,
			name,
			cls,
			isGeneric,
			framework,
			ctorCtx,
			m,
			allClasses,
			usedImports,
		); err != nil {
			return err
		}
	}

	// For generic classes: also emit a typed value constructor used by generic subclasses
	// in other packages that need NSMutableArray[T] (not NSMutableArray[cgo.Object]).
	if isGeneric {
		tChain := buildValueChain(
			name,
			"[T]",
			framework.Framework,
			framework.Classes,
			m,
			usedImports,
		)
		if err := render.Execute(w, "typed_with_ptr", view.TypedWithPtrModel{
			Name:        name,
			TValueChain: strings.TrimPrefix(tChain, "&"),
		}); err != nil {
			return err
		}
	}
	return nil
}

// writeDesignatedInitConstructors emits New[ClassName]With[FirstArg] factory functions
// for every designated initializer that has at least one argument. Each factory allocates
// a new ObjC object via a per-class alloc bridge function, wraps it, then calls the
// existing Go instance method for the designated init (which handles CGo + exception).
func writeDesignatedInitConstructors(
	w io.Writer,
	name string,
	cls macosplatformmetadata.Class,
	isGeneric bool,
	framework *macosplatformmetadata.FrameworkMeta,
	ctx typemap.Context,
	m *typemap.Mapper,
	allClasses map[string]macosplatformmetadata.Class,
	imports typemap.ImportSet,
) error {
	if isGeneric {
		return nil // skip: designated init constructors for generic classes need T constraints
	}

	packageName := strings.ToLower(framework.Framework)
	allocFn := packageName + "_" + name + "_alloc"

	var inits []view.DesignatedInitModel
	seenCtorNames := make(map[string]bool)
	for _, method := range cls.Methods {
		if !method.IsDesignatedInit || !method.IsInit || len(method.Params) == 0 {
			continue
		}
		if method.Availability.IsUnavailable {
			continue
		}
		// Skip when any argument or return references a class that is
		// unavailable on macOS — the constructor signature would name an
		// undefined Go type (e.g. INDateComponentsRange's recurrenceRule
		// arg refers to INRecurrenceRule which is iOS-only).
		if methodRefsUnavailableClass(method, framework, allClasses) {
			continue
		}

		// Go method name: e.g. "initWithContentRect:styleMask:backing:defer:" → "InitWithContentRect"
		goMethodName := naming.MethodName(method.Selector)
		// Strip leading "Init" for ergonomic constructor name: "WithContentRect"
		ctorSuffix := strings.TrimPrefix(goMethodName, "Init")
		if ctorSuffix == "" {
			ctorSuffix = goMethodName
		}
		ctorName := "New" + name + ctorSuffix
		if seenCtorNames[ctorName] {
			continue
		}
		seenCtorNames[ctorName] = true

		// Parameters use the same Go types as the generated instance method.
		// buildGoArgs drops the NSError out-arg (the Init method itself converts it
		// into a Go error return); we mirror that here and propagate the error if
		// HasNSError is set.
		goArgs := buildGoArgs(method.Params, method.IsNSError, ctx, m, imports)
		// Call args: just the resolved names (no type annotations).
		callNames := buildParamNames(method.Params)

		returnSig := "*" + name
		if method.IsNSError {
			returnSig = "(*" + name + ", error)"
		}

		// Determine whether the underlying Init method's Go return type is
		// already *name (instancetype or NSName *), bare unsafe.Pointer (id pre-ARC),
		// or cgo.Object (id post-fix). In the unsafe.Pointer and cgo.Object cases we
		// wrap via New<Class> to preserve the constructor's typed signature.
		initCtx := ctx
		initCtx.ClassName = name
		initCtx.GenericParams = cls.GenericParams
		initCtx.IsReturn = true
		initGoRet := m.GoType(method.Return.ObjCType, initCtx, imports)
		initReturnsUnsafe := initGoRet == "unsafe.Pointer"
		initReturnsID := initGoRet == "cgo.Object"
		initReturnsPtr := initReturnsUnsafe || initReturnsID

		// Use the value-type wrapper (no finalizer) for the alloc result so that
		// the finalizer registered inside the Init method is the sole owner.
		// New<Class>(_raw) would register a second finalizer on the same pointer,
		// causing a double-Release when the GC runs.
		callExpr := fmt.Sprintf("_obj.%s(%s)", goMethodName, strings.Join(callNames, ", "))
		model := view.DesignatedInitModel{
			CtorName:  ctorName,
			ClassName: name,
			Selector:  method.Selector,
			ArgList:   strings.Join(goArgs, ", "),
			ReturnSig: returnSig,
			AllocFn:   allocFn,
			CallExpr:  callExpr,
		}
		switch {
		case initReturnsID && method.IsNSError:
			model.Kind = 1 // id (cgo.Object) return: extract Ptr() for the typed constructor
		case initReturnsID:
			model.Kind = 2
		case initReturnsPtr && method.IsNSError:
			model.Kind = 3
		case initReturnsPtr:
			model.Kind = 4
		default:
			model.Kind = 0
		}
		inits = append(inits, model)
	}

	if err := render.Execute(w, "designated_inits", inits); err != nil {
		return err
	}
	return nil
}

// classConformsToCoding reports whether the class conforms to NSSecureCoding or NSCoding.
func classConformsToCoding(cls macosplatformmetadata.Class) bool {
	for _, p := range cls.Protocols {
		if p == "NSSecureCoding" || p == "NSCoding" {
			return true
		}
	}
	return false
}

// writeCodingMethods emits SerializeToArchive and NewXFromArchive convenience
// methods for classes that conform to NSSecureCoding or NSCoding.
func writeCodingMethods(
	w io.Writer,
	name, framework string,
	cls macosplatformmetadata.Class,
) error {
	if !classConformsToCoding(cls) {
		return nil
	}
	packageName := strings.ToLower(framework)
	return render.Execute(w, "coding_methods", view.CodingMethodsModel{
		Name:          name,
		SerializeFn:   packageName + "_" + name + "_serializeToArchive",
		DeserializeFn: packageName + "_" + name + "_newFromArchive",
	})
}

// writeForeignAncestorExtensions emits ObjC category methods from the current
// framework's ForeignExtensions map onto className when className's canonical
// direct parent is owned by a different framework.
//
// In ObjC, category methods are available on a class AND all its subclasses via
// the runtime. In Go, methods cannot be added to types from other packages, so
// category methods on foreign classes are instead emitted on the first class in
// the current framework's embedding tree that sits above the foreign parent.
// Subclasses of className inherit these methods via Go embedding promotion.
//
// The full foreign ancestor chain is walked (not just the direct parent) so that
// categories on e.g. NSObject are emitted even when the direct parent is another
// Foundation class like NSAttributedString.
func writeForeignAncestorExtensions(
	w io.Writer,
	className, receiver string,
	framework *macosplatformmetadata.FrameworkMeta,
	ctx typemap.Context,
	m *typemap.Mapper,
	allClasses map[string]macosplatformmetadata.Class,
	pkgTypeNames map[string]bool,
	usedImports typemap.ImportSet,
	alreadyEmittedGoNames map[string]bool,
	needsBlocks *bool,
) error {
	if len(framework.ForeignExtensions) == 0 {
		return nil
	}
	directParent := objcclasshierarchy.ObjCClassSuperclassIndex[className]
	if directParent == "" {
		return nil // root class — no foreign ancestors
	}
	parentOwner := objcclasshierarchy.ObjCClassFrameworkIndex[directParent]
	if strings.EqualFold(parentOwner, framework.Framework) {
		return nil // direct parent is same-framework — it handles its own ancestors
	}

	// Walk the foreign ancestor chain and emit category methods from each level.
	// seenSelectors prevents emitting the same selector from two different ancestors
	// (the nearest ancestor wins, matching ObjC dispatch semantics).
	seenSelectors := make(map[string]bool)
	for ancestor := directParent; ancestor != ""; ancestor = objcclasshierarchy.ObjCClassSuperclassIndex[ancestor] {
		if strings.EqualFold(
			objcclasshierarchy.ObjCClassFrameworkIndex[ancestor],
			framework.Framework,
		) {
			break // reached a same-framework ancestor — it handles its own extensions
		}
		extensionMethods, ok := framework.ForeignExtensions[ancestor]
		if !ok {
			continue
		}

		// ctx.ClassName must be the current class (not the ancestor) so that
		// instancetype return types resolve to *CurrentClass, not *AncestorClass.
		// The bridge ID comment still shows the ancestor via method.SDKFile/SDKLine.
		methodCtx := ctx
		methodCtx.ClassName = className

		bridgeNames := buildClassBridgeNames(framework.Framework, ancestor, extensionMethods)
		for _, method := range extensionMethods {
			if method.Availability.IsUnavailable || method.IsClassMethod {
				continue
			}
			if shouldSkipBridgeMethod(method) {
				continue
			}
			if methodRefsUnavailableClass(method, framework, allClasses) {
				continue
			}
			if seenSelectors[method.Selector] {
				continue
			}
			seenSelectors[method.Selector] = true

			goName := naming.MethodName(method.Selector)
			// Skip if the class already emitted a method with this Go name
			// (class's own novel methods take precedence over ancestor extensions).
			if alreadyEmittedGoNames[goName] {
				continue
			}

			mk := methodKey(method.Selector, false)
			em := emittedMethod{
				method:      method,
				definingCls: ancestor,
				cFuncName:   bridgeNames[mk],
				goName:      goName,
			}
			if err := writeMethod(
				w,
				em,
				receiver,
				className,
				framework.Framework,
				methodCtx,
				m,
				framework.Classes,
				allClasses,
				pkgTypeNames,
				usedImports,
			); err != nil {
				return err
			}
			if methodHasBlockArgs(method.Params, m) {
				*needsBlocks = true
			}
		}
	}
	return nil
}

// writeGenericHelper emits a package-private generic constructor for classes that
// use Go generics. This lets same-framework instance methods return *Class[T] (with
// T in scope) rather than the fixed *Class[runtime.Object] that New<Cls> returns.
//
// Example output:
//
//	func newNSArrayT[T runtime.Object](ptr unsafe.Pointer) *NSArray[T] {
//	    if ptr == nil { return nil }
//	    o := &NSArray[T]{NSObject: NSObject{ptr: ptr}}
//	    runtime.Track(o, o.Ptr)
//	    return o
//	}
func writeGenericHelper(
	w io.Writer,
	name string,
	cls macosplatformmetadata.Class,
	si superInfo,
	framework *macosplatformmetadata.FrameworkMeta,
	allClasses map[string]macosplatformmetadata.Class,
	m *typemap.Mapper,
	usedImports typemap.ImportSet,
) error {
	return render.Execute(w, "generic_helper", view.GenericHelperModel{
		Name: name,
		TChain: buildValueChain(
			name,
			"[T]",
			framework.Framework,
			framework.Classes,
			m,
			usedImports,
		),
	})
}

// buildValueChain builds the Go struct literal expression (with leading &) needed
// to construct a value of className with genSuffix using only the ptr pointer.
//
// For root classes (ptr field direct):      "&NSObject{ptr: ptr}"
// For same-fw one-level:                    "&NSArray[T]{NSObject: NSObject{ptr: ptr}}"
// For same-fw two-level:                    "&NSMutableArray[T]{NSArray: NSArray[T]{NSObject: NSObject{ptr: ptr}}}"
// For cross-fw one-level:                   "&VZVirtualMachine{NSObject: foundation.NSObjectWithPtr(ptr)}"
//
// currentFW is the framework being generated (used to detect classes that appear in
// fmClasses due to SDK header sharing but are canonically owned by another framework).
// Any cross-framework package identifiers discovered are recorded into usedImports.
// fmClasses is the current framework's class map. m may be nil.
func buildValueChain(
	className, genSuffix, currentFW string,
	fmClasses map[string]macosplatformmetadata.Class,
	m *typemap.Mapper,
	usedImports typemap.ImportSet,
) string {
	return "&" + buildValueChainInner(className, genSuffix, currentFW, fmClasses, m, usedImports)
}

func buildValueChainInner(
	className, genSuffix, currentFW string,
	fmClasses map[string]macosplatformmetadata.Class,
	m *typemap.Mapper,
	usedImports typemap.ImportSet,
) string {
	cls, inFW := fmClasses[className]
	if !inFW {
		return className + genSuffix + "{/* unknown chain */}"
	}

	if cls.Super == "" {
		// Root class: has its own ptr field.
		return className + genSuffix + "{ptr: ptr}"
	}

	// Determine if super is in the same framework or cross-framework.
	// A class present in fmClasses may still be canonically owned by another framework
	// (e.g. NSExpression appears in CoreData headers but belongs to Foundation).
	// OwnerIndex is authoritative: if it points elsewhere, use the cross-fw path.
	_, superInFW := fmClasses[cls.Super]
	if superInFW && m != nil && m.OwnerIndex[cls.Super] != "" {
		if !strings.EqualFold(m.OwnerIndex[cls.Super], currentFW) {
			superInFW = false
		}
	}

	if superInFW {
		// Same-framework super: recurse to build the inner chain.
		// Only propagate genSuffix if the superclass is itself generic.
		superCls := fmClasses[cls.Super]
		superGenSuffix := ""
		if len(superCls.GenericParams) > 0 {
			if genSuffix != "" {
				superGenSuffix = genSuffix // generic child → propagate [T] or [cgo.Object]
			} else {
				superGenSuffix = "[cgo.Object]" // non-generic child of generic super
			}
		}
		innerChain := buildValueChainInner(
			cls.Super,
			superGenSuffix,
			currentFW,
			fmClasses,
			m,
			usedImports,
		)
		fieldName := cls.Super // Go embedding field name = type name (no generic brackets)
		return className + genSuffix + "{" + fieldName + ": " + innerChain + "}"
	}

	// Cross-framework super. If the super's framework is unknown (not loaded),
	// or importing it would create a cycle, treat this class as a root.
	if m == nil || m.OwnerIndex[cls.Super] == "" {
		return className + genSuffix + "{ptr: ptr}"
	}
	owner := m.OwnerIndex[cls.Super]
	if m.BlockedImports[currentFW][owner] {
		return className + genSuffix + "{ptr: ptr}"
	}
	pkg := strings.ToLower(owner)
	if usedImports != nil && m.ModulePrefix != "" {
		usedImports[pkg] = m.ModulePrefix + "/" + pkg
	}
	var superExpr string
	if m.GenericClasses[cls.Super] && genSuffix == "[T]" {
		superExpr = pkg + "." + cls.Super + "TypedWithPtr[T](ptr)"
	} else {
		superExpr = pkg + "." + cls.Super + "WithPtr(ptr)"
	}
	return className + genSuffix + "{" + cls.Super + ": " + superExpr + "}"
}

// emittedMethod pairs a method with the class that originally defined it,
// its pre-resolved collision-free C bridge function name, and its Go method name.
type emittedMethod struct {
	method      macosplatformmetadata.Method
	definingCls string
	cFuncName   string // resolved by buildClassBridgeNames; avoids C name collisions
	goName      string // resolved Go method/function name; avoids Go name collisions
}

// writeMethod emits a single Go method for the given ObjC method.
// Only the class's OWN methods are emitted; inherited methods are promoted by Go embedding.
// pkgTypeNames is the set of package-level type names (enums, structs, typedefs);
// class methods whose generated function name collides with a type are silently skipped.
func writeMethod(
	w io.Writer,
	em emittedMethod,
	receiver string,
	className, framework string,
	ctx typemap.Context,
	m *typemap.Mapper,
	fmClasses map[string]macosplatformmetadata.Class,
	allClasses map[string]macosplatformmetadata.Class,
	pkgTypeNames map[string]bool,
	imports typemap.ImportSet,
) error {
	method := em.method

	methodCtx := ctx
	if method.IsClassMethod {
		methodCtx.IsClassMethod = true
	}

	cFunc := em.cFuncName
	goName := em.goName // pre-resolved, collision-free
	goArgs := buildGoArgs(method.Params, method.IsNSError, methodCtx, m, imports)
	goRet := buildGoReturn(method, methodCtx, m, className, imports)

	var preamble strings.Builder
	preamble.WriteString(
		renderCommentBlock(method.Doc, method.SDKFile, method.SDKLine, method.Availability, ""),
	)
	if method.IsDesignatedInit {
		preamble.WriteString("// Designated initializer.\n")
	}
	if method.IsWarnUnused {
		preamble.WriteString("// Return value must not be discarded.\n")
	}
	if method.SwiftName != "" {
		fmt.Fprintf(&preamble, "// Swift name: %s\n", method.SwiftName)
	}
	// Blocked-import notes so readers understand why certain types are unsafe.Pointer.
	for _, arg := range method.Params {
		if note := m.BlockedImportNote(arg.ObjCType, methodCtx); note != "" {
			fmt.Fprintf(&preamble, "%s\n", note)
		}
	}
	if note := m.BlockedImportNote(method.Return.ObjCType, methodCtx); note != "" {
		fmt.Fprintf(&preamble, "%s\n", note)
	}
	// Annotate out-parameters so callers know which arguments the callee writes back.
	resolved := buildParamNames(method.Params)
	for i, arg := range method.Params {
		if arg.Direction == "out" {
			fmt.Fprintf(
				&preamble,
				"// %s is an out-parameter: pass a pointer the callee will populate.\n",
				resolved[i],
			)
		}
	}

	retStr := ""
	if goRet != "" {
		retStr = " " + goRet
	}
	model := view.ClassMethodModel{
		PreambleComment: preamble.String(),
		IsClassMethod:   method.IsClassMethod,
		GoArgs:          strings.Join(goArgs, ", "),
		RetStr:          retStr,
		Body: buildMethodBodyModel(
			method,
			cFunc,
			method.IsClassMethod,
			methodCtx,
			m,
			fmClasses,
			imports,
		),
	}
	if method.IsClassMethod {
		funcName := className + goName
		model.FuncName = funcName
		model.BridgeID = naming.MethodBridgeID(
			methodCtx.Framework,
			className,
			method.Selector,
			method.IsClassMethod,
		)
		// A class method whose Go name collides with a package-level type is
		// dropped (the type and func cannot share a name); the preamble comment is
		// still emitted, matching the original.
		model.Skip = pkgTypeNames[funcName]
	} else {
		model.Receiver = receiver
		model.GoName = goName
		model.BridgeID = naming.MethodBridgeID(
			methodCtx.Framework,
			methodCtx.ClassName,
			method.Selector,
			method.IsClassMethod,
		)
	}
	if err := render.Execute(w, "class_method", model); err != nil {
		return err
	}

	// Emit NSString Go-string convenience overload when enabled.
	if m.IsNSStringOverloads && nsStringArgIndices(method.Params) != nil {
		if method.IsClassMethod {
			if err := writeNSStringClassOverload(
				w,
				goName,
				className,
				method,
				methodCtx,
				m,
				pkgTypeNames,
				imports,
			); err != nil {
				return err
			}
		} else {
			if err := writeNSStringInstanceOverload(
				w,
				goName,
				receiver,
				method,
				methodCtx,
				m,
				imports,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

// buildMethodBodyModel resolves the CGo body of a class instance/class method.
func buildMethodBodyModel(
	method macosplatformmetadata.Method,
	cFunc string,
	isClassMethod bool,
	ctx typemap.Context,
	m *typemap.Mapper,
	fmClasses map[string]macosplatformmetadata.Class,
	imports typemap.ImportSet,
) view.MethodBodyModel {
	var preambles []string
	var keepAlives []string
	cgoCallArgs := buildCGOCallArgs(
		method.Params,
		isClassMethod,
		method.IsNSError,
		true,
		ctx,
		m,
		&preambles,
		&keepAlives,
		imports,
	)
	return resolveMethodBodyModel(
		method,
		cFunc,
		cgoCallArgs,
		preambles,
		keepAlives,
		!isClassMethod,
		"o",
		isClassMethod,
		ctx,
		m,
		fmClasses,
		imports,
	)
}

// resolveMethodBodyModel builds the method-body model from an already-marshaled
// CGo call: keep-alive defers, argument preambles, the C call, exception/NSError
// handling, and the return conversion. Shared by class methods (receiver "o")
// and id<Protocol> proxy methods (receiver "p").
func resolveMethodBodyModel(
	method macosplatformmetadata.Method,
	cFunc, cgoCallArgs string,
	preambles, keepAlives []string,
	hasReceiver bool,
	receiverVar string,
	isClassMethod bool,
	ctx typemap.Context,
	m *typemap.Mapper,
	fmClasses map[string]macosplatformmetadata.Class,
	imports typemap.ImportSet,
) view.MethodBodyModel {
	retType := primaryReturnType(method, ctx, m, imports)
	isNullableStr := retType == "string" && method.Return.IsNullable
	isValueStruct := isValueStructReturn(retType, m)
	rawCall := fmt.Sprintf("C.%s(%s)", cFunc, cgoCallArgs)

	model := view.MethodBodyModel{
		HasReceiver: hasReceiver,
		ReceiverVar: receiverVar,
		KeepAlives:  keepAlives,
		Preambles:   preambles,
		HasNSError:  method.IsNSError,
		RawCall:     rawCall,
		RetType:     retType,
	}
	switch {
	case retType == "":
		model.RetKind = 0
	case retType == "cgo.Object":
		model.RetKind = 1
	case isObjectReturn(retType):
		model.RetKind = 2
		model.WrapTypedExpr = constructorRef(extractStructType(retType, isClassMethod), fmClasses)
	case isValueStruct:
		model.RetKind = 3
	case isNullableStr:
		model.RetKind = 4
	default:
		model.RetKind = 5
		model.ResultExpr = cgoReturnConvert(rawCall, retType, m)
	}
	return model
}

// extractStructType strips the leading "*" and substitutes [cgo.Object] for [T]
// in class-method context (where T is not in scope).
func extractStructType(retType string, isClassMethod bool) string {
	s := strings.TrimPrefix(retType, "*")
	if isClassMethod {
		s = strings.ReplaceAll(s, "[T]", "[cgo.Object]")
	}
	return s
}

// objectConstructExpr returns the Go expression that wraps _ptr in a typed Go object.
// structType has no leading "*" (e.g. "NSString", "NSArray[T]", "foundation.NSString").
func objectConstructExpr(
	structType, ptrVar string,
	fmClasses map[string]macosplatformmetadata.Class,
	m *typemap.Mapper,
) string {
	// Cross-framework: "foundation.NSString" → foundation.NewNSString(_ptr)
	if isCrossFrameworkType(structType) {
		return crossFrameworkCtor(structType, ptrVar)
	}

	// T-generic same-framework: "NSArray[T]" → NewNSArrayT[T](_ptr)
	if strings.Contains(structType, "[T]") {
		baseName := structType[:strings.Index(structType, "[")]
		return "New" + baseName + "T[T](" + ptrVar + ")"
	}

	// Non-T same-framework: "NSString" or "NSArray[runtime.Object]"
	baseName := structType
	if br := strings.Index(baseName, "["); br > 0 {
		baseName = baseName[:br]
	}
	return "New" + baseName + "(" + ptrVar + ")"
}

// isCrossFrameworkType returns true when structType belongs to a foreign package:
// it has a "." before any "[" (e.g. "foundation.NSString").
func isCrossFrameworkType(structType string) bool {
	bracket := strings.Index(structType, "[")
	dot := strings.Index(structType, ".")
	if dot < 0 {
		return false
	}
	return bracket < 0 || dot < bracket
}

// crossFrameworkCtor builds a foreign constructor call: pkg.NewTypeName(ptr).
func crossFrameworkCtor(structType, ptr string) string {
	dot := strings.Index(structType, ".")
	pkg := structType[:dot]
	typeName := structType[dot+1:]
	isGenericT := strings.Contains(typeName, "[T]")
	if br := strings.Index(typeName, "["); br > 0 {
		typeName = typeName[:br]
	}
	if isGenericT {
		// Use the exported generic constructor to preserve the T type parameter.
		return pkg + ".New" + typeName + "T[T](" + ptr + ")"
	}
	return pkg + ".New" + typeName + "(" + ptr + ")"
}

// constructorRef returns the Go constructor function reference for use with
// cgo.WrapTyped. Unlike objectConstructExpr, it returns the function itself
// (not a call), so the caller passes a nil-guarding pointer to WrapTyped.
//   - Cross-framework: "foundation.NewNSString"
//   - T-generic same-framework: "NewNSArrayT[T]"
//   - Non-T same-framework: "NewNSString"
func constructorRef(structType string, fmClasses map[string]macosplatformmetadata.Class) string {
	if isCrossFrameworkType(structType) {
		dot := strings.Index(structType, ".")
		pkg := structType[:dot]
		typeName := structType[dot+1:]
		isGenericT := strings.Contains(typeName, "[T]")
		if br := strings.Index(typeName, "["); br > 0 {
			typeName = typeName[:br]
		}
		if isGenericT {
			return pkg + ".New" + typeName + "T[T]"
		}
		return pkg + ".New" + typeName
	}
	if strings.Contains(structType, "[T]") {
		baseName := structType[:strings.Index(structType, "[")]
		return "New" + baseName + "T[T]"
	}
	baseName := structType
	if br := strings.Index(baseName, "["); br > 0 {
		baseName = baseName[:br]
	}
	return "New" + baseName
}

// blockNeedsWrapper returns true if any block argument or return type requires
// conversion (NSError * → error, known ObjC class pointer arg or return → *Type).
func blockNeedsWrapper(blockObjCType string, ctx typemap.Context, m *typemap.Mapper) bool {
	retType, args, ok := typemap.ParseBlock(blockObjCType)
	if !ok {
		return false
	}
	for _, a := range args {
		if typemap.IsNSError(a) {
			return true
		}
		if typemap.IsBOOLPointer(a) {
			return true
		}
		// Bare `id` needs wrapping: MakeBlock_* expects unsafe.Pointer but user sees cgo.Object.
		if typemap.IsID(a) {
			return true
		}
		if len(typemap.IDProtocols(a)) > 0 {
			return true
		}
		if cls := typemap.ClassName(
			a,
		); cls != "" &&
			(ctx.ClassNameIndex[cls] || m.StructIndex[cls] != "") {
			return true
		}
		// Framework-specific opaque CF types (CGColorRef, CMSampleBufferRef, etc.)
		n := strings.TrimSpace(typemap.Normalise(a))
		if _, ok := m.CFTypeIndex[n]; ok {
			return true
		}
		// Lowercase typedef that resolves to a known ObjC class pointer
		// (e.g. ar_anchor_t → NSObject<OS_ar_anchor> * → *foundation.NSObject).
		if !strings.ContainsAny(n, " *<>^()") {
			if target, ok := m.TypedefIndex[n]; ok {
				if targetCls := typemap.ClassName(
					strings.TrimSpace(typemap.Normalise(target)),
				); targetCls != "" &&
					ctx.ClassNameIndex[targetCls] {
					return true
				}
			}
		}
		// ObjC generic type parameter in block arg (e.g. ObjectType in NSArray enumerate blocks):
		// needs wrapping unsafe.Pointer → cgo.Object via cgo.WrapObject.
		if !strings.ContainsAny(n, " *<>^()") && len(ctx.GenericParams) > 0 {
			for _, gp := range ctx.GenericParams {
				if n == gp {
					return true
				}
			}
		}
	}
	// Also wrap when the return type is a typed ObjC class pointer or bare id.
	if typemap.IsID(retType) {
		return true
	}
	if len(typemap.IDProtocols(retType)) > 0 {
		return true
	}
	if cls := typemap.ClassName(retType); cls != "" && ctx.ClassNameIndex[cls] {
		return true
	}
	return false
}

// blockArgCtorTyped is like blockArgCtor but distinguishes Go struct pointer
// types (which need an unsafe cast) from ObjC class pointers (which use New* constructors).
// Struct types are detected via m.StructIndex.
func blockArgCtorTyped(goType, pName string, m *typemap.Mapper) string {
	if strings.HasPrefix(goType, "*") && goType != "*bool" {
		typeName := goType[1:]
		baseName := typeName
		if br := strings.Index(typeName, "["); br >= 0 {
			baseName = typeName[:br]
		}
		structName := baseName
		if dot := strings.LastIndex(baseName, "."); dot >= 0 {
			structName = baseName[dot+1:]
		}
		if m.StructIndex[structName] != "" {
			// Go struct pointer: use unsafe cast, not New* constructor.
			return "(" + goType + ")(" + pName + ")"
		}
	}
	return blockArgCtor(goType, pName)
}

// blockArgCtor returns the Go expression that converts a raw unsafe.Pointer block
// argument to its user-facing typed form.
//   - error (NSError *): cgo.NSErrorToError(pName)
//   - *ClassName: NewClassName(pName)
//   - *pkg.ClassName: pkg.NewClassName(pName)
//   - *Generic[T]: (*Generic[T])(unsafe.Pointer(pName)) — unsafe cast preserves type param
//   - unsafe.Pointer: pName (pass through)
func blockArgCtor(goType, pName string) string {
	switch goType {
	case "error":
		return "cgo.NSErrorToError(" + pName + ")"
	case "cgo.Object":
		// Wrap the raw unsafe.Pointer trampoline argument in a minimal ObjC object.
		return "cgo.WrapObject(" + pName + ")"
	case "unsafe.Pointer", "":
		return pName
	}
	if strings.HasPrefix(goType, "*") {
		typeName := goType[1:]
		// Strip generic type parameters before looking for a package qualifier.
		// Without this, strings.Index finds the "." inside "[cgo.Object]" and
		// produces broken output.
		baseName := typeName
		hasTypeParam := false
		if br := strings.Index(typeName, "["); br >= 0 {
			baseName = typeName[:br]
			hasTypeParam = true
		}
		// Generic types require an unsafe cast because the constructor returns
		// *Foo[cgo.Object] but the user func may expect *Foo[T].
		if hasTypeParam {
			return "(" + goType + ")(unsafe.Pointer(" + pName + "))"
		}
		if dot := strings.Index(baseName, "."); dot >= 0 {
			pkg := baseName[:dot]
			name := baseName[dot+1:]
			return "cgo.WrapTyped(" + pName + ", " + pkg + ".New" + name + ")"
		}
		return "cgo.WrapTyped(" + pName + ", New" + baseName + ")"
	}
	return pName
}

// buildBlockWrapper generates an inline Go closure that adapts a user-facing
// func(... TypedArgs ...) to the func(... unsafe.Pointer ...) required by MakeBlock_*.
// NSError * args are converted via cgo.NSErrorToError; known ObjC class pointer args
// are wrapped via their Go constructors; BOOL * args become *bool in-out parameters;
// other args pass through unchanged.
// When the block has a non-void return type, the wrapper emits a return statement.
func buildBlockWrapper(
	blockObjCType, userArgName string,
	ctx typemap.Context,
	m *typemap.Mapper,
	imports typemap.ImportSet,
) string {
	retType, args, ok := typemap.ParseBlock(blockObjCType)
	if !ok {
		return userArgName
	}
	// Derive the conversion expression for each arg directly from the ObjC type,
	// using the same logic as GoBlockUserFuncType. This avoids trying to re-parse
	// the generated Go func type string.
	argCtx := ctx
	argCtx.IsReturn = false

	var innerParams []string
	var calls []string
	// preambles/postambles are used for BOOL * in-out parameters (stop flags).
	var preambles []string
	var postambles []string
	idx := 0
	for _, a := range args {
		if typemap.IsVoid(a) || a == "" {
			continue
		}
		pName := fmt.Sprintf("_p%d", idx)
		idx++

		if typemap.IsBOOLPointer(a) {
			// BOOL *stop: in-out bool pointer. The trampoline passes unsafe.Pointer;
			// we shadow it with a bool, call the user func with &shadow, then write back.
			sName := fmt.Sprintf("_s%d", idx-1)
			innerParams = append(innerParams, pName+" unsafe.Pointer")
			calls = append(calls, "&"+sName)
			preambles = append(
				preambles,
				fmt.Sprintf(
					"var %s bool; if %s != nil { %s = *(*bool)(%s) }",
					sName,
					pName,
					sName,
					pName,
				),
			)
			postambles = append(postambles,
				fmt.Sprintf("if %s != nil { *(*bool)(%s) = %s }", pName, pName, sName))
			continue
		}

		innerType := m.GoBlockArgType(a)
		if innerType == "" {
			innerType = "unsafe.Pointer"
		}
		innerParams = append(innerParams, pName+" "+innerType)

		var goType string
		if typemap.IsNSError(a) {
			goType = "error"
		} else if cls := typemap.ClassName(a); cls != "" && (argCtx.ClassNameIndex[cls] || m.StructIndex[cls] != "") {
			if typed := m.GoType(a, argCtx, imports); typed != "unsafe.Pointer" {
				goType = typed
			}
		} else {
			n := strings.TrimSpace(typemap.Normalise(a))
			// Bare id (any ObjC object) → cgo.Object.
			if typemap.IsID(n) {
				goType = "cgo.Object"
			} else if protos := typemap.IDProtocols(n); len(protos) > 0 {
				// id<Protocol> block arg: use return-position semantics to get *ProtoIDProtocol
				// or cgo.Object — both are constructible from unsafe.Pointer by blockArgCtor.
				retCtx := argCtx
				retCtx.IsReturn = true
				if typed := m.GoType(a, retCtx, imports); typed != "" && typed != "unsafe.Pointer" {
					goType = typed
				} else {
					goType = "cgo.Object"
				}
			} else if _, ok := m.CFTypeIndex[n]; ok {
				// Framework-specific opaque CF types (CGColorRef, CMSampleBufferRef, etc.)
				goType = m.GoType(a, argCtx, imports)
			} else if !strings.ContainsAny(n, " *<>^()") {
				// ObjC generic type parameter in block arg (e.g. ObjectType in NSArray enumerate).
				// blockArgCtor handles "cgo.Object" via cgo.WrapObject.
				if len(argCtx.GenericParams) > 0 {
					for _, gp := range argCtx.GenericParams {
						if n == gp {
							goType = "cgo.Object"
							break
						}
					}
				}
				if goType == "" {
					// Lowercase typedef that resolves to a known ObjC class pointer
					// (e.g. ar_anchor_t → NSObject<OS_ar_anchor> * → *foundation.NSObject).
					if target, ok := m.TypedefIndex[n]; ok {
						if targetCls := typemap.ClassName(
							strings.TrimSpace(typemap.Normalise(target)),
						); targetCls != "" &&
							argCtx.ClassNameIndex[targetCls] {
							goType = m.GoType(a, argCtx, imports)
						}
					}
				}
			}
		}
		calls = append(calls, blockArgCtorTyped(goType, pName, m))
	}
	innerRet := m.GoBlockArgType(retType)
	hasBoolPtr := len(preambles) > 0

	// When the return type is a known ObjC class pointer or bare id, the user-facing
	// func type has a typed return. Generate a wrapper that calls the user's typed
	// closure and converts the result back to unsafe.Pointer for the MakeBlock_* trampoline.
	retIsID := typemap.IsID(strings.TrimSpace(typemap.Normalise(retType)))
	retIsIDProtocol := len(typemap.IDProtocols(strings.TrimSpace(typemap.Normalise(retType)))) > 0
	if retIsID || retIsIDProtocol ||
		(typemap.ClassName(retType) != "" && ctx.ClassNameIndex[typemap.ClassName(retType)]) {
		argCtx := ctx
		argCtx.IsReturn = false
		var typed string
		if retIsID {
			typed = "cgo.Object"
		} else if retIsIDProtocol {
			retCtx := argCtx
			retCtx.IsReturn = true
			if t2 := m.GoType(retType, retCtx, imports); t2 != "" && t2 != "unsafe.Pointer" {
				typed = t2
			} else {
				typed = "cgo.Object"
			}
		} else {
			typed = m.GoType(retType, argCtx, imports)
		}
		if typed != "" && typed != "unsafe.Pointer" {
			userCall := userArgName + "(" + strings.Join(calls, ", ") + ")"
			if typed == "cgo.Object" {
				// cgo.Object return: convert via Ptr() (may be nil).
				if hasBoolPtr {
					var sb strings.Builder
					fmt.Fprintf(&sb, "func(%s) unsafe.Pointer {", strings.Join(innerParams, ", "))
					for _, pre := range preambles {
						fmt.Fprintf(&sb, " %s;", pre)
					}
					fmt.Fprintf(&sb, " _r := %s;", userCall)
					for _, post := range postambles {
						fmt.Fprintf(&sb, " %s;", post)
					}
					sb.WriteString(" if _r == nil { return nil }; return _r.Ptr() }")
					return sb.String()
				}
				return fmt.Sprintf(
					"func(%s) unsafe.Pointer { _r := %s; if _r == nil { return nil }; return _r.Ptr() }",
					strings.Join(innerParams, ", "),
					userCall,
				)
			}
			if hasBoolPtr {
				var sb strings.Builder
				fmt.Fprintf(&sb, "func(%s) unsafe.Pointer {", strings.Join(innerParams, ", "))
				for _, pre := range preambles {
					fmt.Fprintf(&sb, " %s;", pre)
				}
				fmt.Fprintf(&sb, " _r := %s;", userCall)
				for _, post := range postambles {
					fmt.Fprintf(&sb, " %s;", post)
				}
				sb.WriteString(" if _r == nil { return nil }; return unsafe.Pointer(_r.Ptr()) }")
				return sb.String()
			}
			return fmt.Sprintf(
				"func(%s) unsafe.Pointer { _r := %s; if _r == nil { return nil }; return unsafe.Pointer(_r.Ptr()) }",
				strings.Join(innerParams, ", "),
				userCall,
			)
		}
	}

	if hasBoolPtr {
		var sb strings.Builder
		paramsStr := strings.Join(innerParams, ", ")
		callsStr := strings.Join(calls, ", ")
		if innerRet != "" {
			fmt.Fprintf(&sb, "func(%s) %s {", paramsStr, innerRet)
		} else {
			fmt.Fprintf(&sb, "func(%s) {", paramsStr)
		}
		for _, pre := range preambles {
			fmt.Fprintf(&sb, " %s;", pre)
		}
		if innerRet != "" {
			fmt.Fprintf(&sb, " _ret := %s(%s);", userArgName, callsStr)
		} else {
			fmt.Fprintf(&sb, " %s(%s);", userArgName, callsStr)
		}
		for _, post := range postambles {
			fmt.Fprintf(&sb, " %s;", post)
		}
		if innerRet != "" {
			sb.WriteString(" return _ret }")
		} else {
			sb.WriteString(" }")
		}
		return sb.String()
	}

	if innerRet != "" {
		return fmt.Sprintf(
			"func(%s) %s { return %s(%s) }",
			strings.Join(innerParams, ", "),
			innerRet,
			userArgName,
			strings.Join(calls, ", "),
		)
	}
	return fmt.Sprintf(
		"func(%s) { %s(%s) }",
		strings.Join(innerParams, ", "),
		userArgName,
		strings.Join(calls, ", "),
	)
}

// methodHasBlockArgs returns true if any argument is a block type (inline or
// typedef-aliased), matching the same detection logic used in buildCGOCallArgs.
func methodHasBlockArgs(args []macosplatformmetadata.Param, m *typemap.Mapper) bool {
	for _, arg := range args {
		if arg.IsBlock {
			return true
		}
		if target, ok := m.TypedefIndex[typemap.Normalise(arg.ObjCType)]; ok &&
			typemap.IsBlock(target) {
			return true
		}
	}
	return false
}

// buildCGOCallArgs builds the comma-separated CGo call argument list.
// Instance methods prepend o.Ptr() (the promoted pointer accessor).
// When withException is true, &_exc is appended as the final argument
// to match the void **outException out-parameter on every generated bridge function.
// Block args are wrapped in blocks.MakeBlock_* trampolines and freed via defer.
func buildCGOCallArgs(
	args []macosplatformmetadata.Param,
	isClassMethod, hasNSError, withException bool,
	ctx typemap.Context,
	m *typemap.Mapper,
	preambles, keepAlives *[]string,
	imports typemap.ImportSet,
) string {
	var parts []string

	if !isClassMethod {
		parts = append(parts, "o.Ptr()")
	}

	resolved := buildParamNames(args)
	for i, arg := range args {
		// blockObjCType returns the concrete ObjC block type string for this arg,
		// whether the block syntax is inline (IsBlock) or hidden behind a typedef.
		blockObjCType := ""
		if arg.IsBlock {
			blockObjCType = arg.ObjCType
		} else if target, ok := m.TypedefIndex[typemap.Normalise(arg.ObjCType)]; ok && typemap.IsBlock(target) {
			blockObjCType = target
		}
		if blockObjCType != "" {
			sigName := BlockSigName(blockObjCType, m)
			if sigName != "" {
				blkVar := "_blk_" + resolved[i]
				argExpr := resolved[i]
				if blockNeedsWrapper(blockObjCType, ctx, m) {
					argExpr = buildBlockWrapper(blockObjCType, resolved[i], ctx, m, imports)
				}
				*preambles = append(*preambles,
					blkVar+" := blocks.MakeBlock_"+sigName+"("+argExpr+")",
					"defer blocks.FreeBlock("+blkVar+")",
				)
				parts = append(parts, blkVar)
			} else {
				// Fallback: unknown block signature — pass nil.
				parts = append(parts, "unsafe.Pointer(nil)")
			}
			continue
		}
		goType := m.GoType(arg.ObjCType, ctx, imports)
		if goType == "" {
			goType = "unsafe.Pointer"
		}
		parts = append(parts, goCGoArgExpr(goType, resolved[i], m, preambles, keepAlives))
	}

	if hasNSError {
		parts = append(parts, "&_nsErr")
	}

	if withException {
		parts = append(parts, "&_exc")
	}

	return strings.Join(parts, ", ")
}

// goCGoArgExpr converts a Go-typed argument to the CGo call expression.
// ObjC-object arguments whose wrappers must survive the CGo call are appended
// to keepAlives so the caller can emit defer cgo.KeepAlive for each ObjC-object
// instead of emitting individual defer cgo.KeepAlive statements.
func goCGoArgExpr(
	goType, argName string,
	m *typemap.Mapper,
	preambles, keepAlives *[]string,
) string {
	switch goType {
	case "unsafe.Pointer":
		return argName
	case "cgo.Object":
		// Interface value: nil-safe pointer extraction. KeepAlive is handled by
		// the caller via a defer cgo.KeepAlive on the argument name.
		varName := "_objcPtr_" + argName
		*preambles = append(*preambles,
			"var "+varName+" unsafe.Pointer",
			"if "+argName+" != nil { "+varName+" = "+argName+".Ptr() }",
		)
		*keepAlives = append(*keepAlives, argName)
		return varName
	case "bool":
		return "C.bool(" + argName + ")"
	case "int8":
		return "C.int8_t(" + argName + ")"
	case "int16":
		return "C.int16_t(" + argName + ")"
	case "int32":
		return "C.int32_t(" + argName + ")"
	case "int64":
		return "C.int64_t(" + argName + ")"
	case "uint8":
		return "C.uint8_t(" + argName + ")"
	case "uint16":
		return "C.uint16_t(" + argName + ")"
	case "uint32":
		return "C.uint32_t(" + argName + ")"
	case "uint64":
		return "C.uint64_t(" + argName + ")"
	case "float32":
		return "C.float(" + argName + ")"
	case "float64":
		return "C.double(" + argName + ")"
	case "string":
		cvar := "_cstr_" + argName
		*preambles = append(*preambles,
			cvar+" := C.CString("+argName+")",
			"defer C.free(unsafe.Pointer("+cvar+"))",
		)
		return cvar
	}
	// Pointer types — distinguish ObjC class wrappers (have .Ptr()) from struct
	// pointers (raw Go struct, address taken with unsafe.Pointer directly).
	if strings.HasPrefix(goType, "*") {
		inner := strings.TrimPrefix(goType, "*")
		// BSD package pointer (e.g. *bsd.InAddr): pass pointer value directly.
		if strings.HasPrefix(inner, "bsd.") {
			return "unsafe.Pointer(" + argName + ")"
		}
		// Strip generic suffix like "[T]" or "[runtime.Object]" so the bare
		// type name can be matched against StructIndex.
		bare := inner
		if br := strings.Index(bare, "["); br > 0 {
			bare = bare[:br]
		}
		// Drop "pkg." qualifier if present.
		if dot := strings.LastIndex(bare, "."); dot >= 0 {
			bare = bare[dot+1:]
		}
		if m != nil && isKnownStruct(bare, m) {
			// `desc *AEDesc` → CGo expects void *: pass the Go pointer directly.
			return "unsafe.Pointer(" + argName + ")"
		}
		// Pointer to primitive (*uint8, *int32, etc.) or pointer to enum
		// (*SomeEnum): bridge takes void *, so pass the pointer directly.
		if isPrimitiveGoType(bare) || (m != nil && m.EnumGoIntType(bare) != "") {
			return "unsafe.Pointer(" + argName + ")"
		}
		// ObjC class wrapper — nil-safe extract of raw pointer via Ptr().
		// KeepAlive is handled by the caller (defer cgo.KeepAlive) so the GC cannot
		// finalize the wrapper before the CGo callee retains the pointer —
		// e.g. [NSButton setImage:] must retain the image before our bridge
		// returns, but the compiler may consider the typed pointer dead once
		// its raw Ptr() value has been extracted.
		varName := "_objcPtr_" + argName
		*preambles = append(*preambles,
			"var "+varName+" unsafe.Pointer",
			"if "+argName+" != nil { "+varName+" = "+argName+".Ptr() }",
		)
		*keepAlives = append(*keepAlives, argName)
		return varName
	}
	// Named enum type (e.g. CFNotificationSuspensionBehavior or pkg.SomeEnum):
	// cast directly to the underlying C integer type so the bridge receives the
	// numeric value rather than an invalid unsafe.Pointer conversion.
	if m != nil {
		if intType := m.EnumGoIntType(goType); intType != "" {
			switch intType {
			case "int8":
				return "C.int8_t(" + argName + ")"
			case "int16":
				return "C.int16_t(" + argName + ")"
			case "int32":
				return "C.int32_t(" + argName + ")"
			case "int64":
				return "C.int64_t(" + argName + ")"
			case "uint8":
				return "C.uint8_t(" + argName + ")"
			case "uint16":
				return "C.uint16_t(" + argName + ")"
			case "uint32":
				return "C.uint32_t(" + argName + ")"
			case "uint64":
				return "C.uint64_t(" + argName + ")"
			}
		}
	}
	// Value-type struct (e.g. `CGSize`, `foundation.NSOperatingSystemVersion`):
	// CGo's `void *` bridge expects a pointer; passing the struct by value
	// would be `unsafe.Pointer(arg)` which Go rejects ("cannot convert struct
	// value to unsafe.Pointer"). Take the address and pass that — the bridge
	// .m file already dereferences with `*(CGSize*)arg`.
	if m != nil {
		// BSD package value types (bsd.Timespec, bsd.EtherAddr, etc.):
		// take the address so the bridge can dereference with *(struct T*)ptr.
		if strings.HasPrefix(goType, "bsd.") {
			return "unsafe.Pointer(&" + argName + ")"
		}
		bare := goType
		if br := strings.Index(bare, "["); br > 0 {
			bare = bare[:br]
		}
		if dot := strings.LastIndex(bare, "."); dot >= 0 {
			bare = bare[dot+1:]
		}
		if isKnownStruct(bare, m) {
			return "unsafe.Pointer(&" + argName + ")"
		}
		// Protocol-typed parameter (id<P> resolved to a Go interface). Every
		// generated protocol interface embeds runtime.Object, so calling
		// .Ptr() yields the underlying ObjC pointer the bridge needs. Strip
		// the `Protocol` suffix added by ProtocolGoTypeName when checking
		// against ProtocolIndex, plus the inline `interface { … }` form
		// produced for multi-protocol id<P1, P2>.
		if strings.HasPrefix(goType, "interface {") {
			return "unsafe.Pointer(" + argName + ".Ptr())"
		}
		// Check bare name first (e.g. "MLComputeDeviceProtocol" where Protocol
		// is part of the ObjC protocol name), then with Protocol suffix stripped
		// (e.g. "MPSKernelable" → ProtocolIndex["MPSKernel"]).
		protoName := strings.TrimSuffix(bare, "Protocol")
		if m.ProtocolIndex[bare] != "" || m.ProtocolIndex[protoName] != "" {
			return "unsafe.Pointer(" + argName + ".Ptr())"
		}
	}
	return "unsafe.Pointer(" + argName + ")"
}

// cgoReturnConvert wraps a raw CGo call expression in the cast needed to produce
// the Go return type. m may be nil when enum detection is not required.
func cgoReturnConvert(callExpr, goType string, m *typemap.Mapper) string {
	switch goType {
	case "unsafe.Pointer":
		return "unsafe.Pointer(" + callExpr + ")"
	case "bool":
		return "bool(" + callExpr + ")"
	case "int8":
		return "int8(" + callExpr + ")"
	case "int16":
		return "int16(" + callExpr + ")"
	case "int32":
		return "int32(" + callExpr + ")"
	case "int64":
		return "int64(" + callExpr + ")"
	case "uint8":
		return "uint8(" + callExpr + ")"
	case "uint16":
		return "uint16(" + callExpr + ")"
	case "uint32":
		return "uint32(" + callExpr + ")"
	case "uint64":
		return "uint64(" + callExpr + ")"
	case "float32":
		return "float32(" + callExpr + ")"
	case "float64":
		return "float64(" + callExpr + ")"
	case "string":
		return "C.GoString(" + callExpr + ")"
	}
	// Named enum return type (e.g. "MTLPixelFormat" or "metal.MTLPixelFormat"):
	// cast from the underlying C integer the bridge returned to the Go enum type.
	if m != nil && m.EnumGoIntType(goType) != "" {
		return goType + "(" + callExpr + ")"
	}
	return callExpr
}

// primaryReturnType resolves the non-error return type for a method.
func primaryReturnType(
	method macosplatformmetadata.Method,
	ctx typemap.Context,
	m *typemap.Mapper,
	imports typemap.ImportSet,
) string {
	ret := method.Return
	if ret.IsGeneric && len(ctx.GenericParams) > 0 {
		// Generic element return (e.g. firstObject → ObjectType *).
		// Matches buildGoReturn: both emit cgo.Object so the body is consistent.
		return "cgo.Object"
	}
	if ret.IsInstancetype || typemap.IsInstancetype(ret.ObjCType) {
		if ctx.ClassName != "" {
			retType := "*" + ctx.ClassName
			if len(ctx.GenericParams) > 0 {
				retType = "*" + ctx.ClassName + "[T]"
			} else if m.GenericClasses[ctx.ClassName] {
				retType = "*" + ctx.ClassName + "[cgo.Object]"
			}
			return retType
		}
		return "unsafe.Pointer"
	}
	return m.GoReturnType(ret.ObjCType, ctx, imports)
}

// isKnownStruct reports whether bare (the Go type name with pkg. prefix stripped)
// is a registered C struct. It checks both the Go-capitalized form (e.g. "Decform")
// and the original ObjC-lowercase form (e.g. "decform"), because the StructIndex
// registry stores ObjC names as-is while qualifiedStructType capitalises the first
// letter via naming.GoTypeName.
func isKnownStruct(bare string, m *typemap.Mapper) bool {
	// Preferred: forward-mapped exported Go names (bare is the resolved Go type
	// name, which uses the non-invertible ExportedTypeName mapping).
	if m.IsStructGoName(bare) {
		return true
	}
	// Fallback for single-word names (where ExportedTypeName == the capitalised C
	// name) and callers that did not build the Go-name index: the StructIndex is
	// keyed by the ObjC/C name, so also try the lowercase-first form.
	if m.StructIndex[bare] != "" {
		return true
	}
	if len(bare) > 0 {
		lower := strings.ToLower(string(bare[0])) + bare[1:]
		if m.StructIndex[lower] != "" {
			return true
		}
	}
	return false
}

// isValueStructReturn returns true when the resolved Go return type is a value-type
// struct (CGSize, NSOperatingSystemVersion, ether_addr, …). The bridge malloc's the
// struct and returns void*; the emitter must dereference and free the pointer.
// Enums and primitives share similar names but are excluded via m.EnumGoIntType.
func isValueStructReturn(goType string, m *typemap.Mapper) bool {
	if goType == "" || goType == "unsafe.Pointer" {
		return false
	}
	if strings.HasPrefix(goType, "*") {
		return false
	}
	switch goType {
	case "bool", "int8", "int16", "int32", "int64",
		"uint8", "uint16", "uint32", "uint64", "float32", "float64",
		"string", "error", "cgo.Object":
		return false
	}
	if m != nil && m.EnumGoIntType(goType) != "" {
		return false
	}
	return true
}

// isObjectReturn returns true if the Go return type is an ObjC wrapper pointer.
func isObjectReturn(goType string) bool {
	if !strings.HasPrefix(goType, "*") {
		return false
	}
	inner := goType[1:]
	switch inner {
	case "bool", "int8", "int16", "int32", "int64",
		"uint8", "uint16", "uint32", "uint64", "float32", "float64", "string":
		return false
	}
	return true
}

// nsStringArgIndices returns the indices of args with NSString * ObjCType,
// or nil if no such args exist.
func nsStringArgIndices(args []macosplatformmetadata.Param) []int {
	var idxs []int
	for i, arg := range args {
		if typemap.ClassName(typemap.Normalise(arg.ObjCType)) == "NSString" {
			idxs = append(idxs, i)
		}
	}
	return idxs
}

// nsStringOverloadArgs returns Go argument strings for the overload:
// NSString * args become plain string, all others keep their resolved type.
func nsStringOverloadArgs(
	args []macosplatformmetadata.Param,
	hasNSError bool,
	nsIdxs []int,
	ctx typemap.Context,
	m *typemap.Mapper,
	imports typemap.ImportSet,
) []string {
	// Build a set for O(1) lookup.
	idxSet := make(map[int]bool, len(nsIdxs))
	for _, i := range nsIdxs {
		idxSet[i] = true
	}
	resolved := buildParamNames(args)
	out := []string{}
	for i, arg := range args {
		if idxSet[i] {
			out = append(out, resolved[i]+" string")
		} else {
			goType := m.GoType(arg.ObjCType, ctx, imports)
			out = append(out, resolved[i]+" "+goType)
		}
	}
	if hasNSError {
		out = append(out, "_nsErr_ unsafe.Pointer")
	}
	return out
}

// nsStringConvertArg returns the call-site expression for a single arg in the
// primary method call from a Go-string overload.
// NSString args are wrapped via runtime.GoStringToNSString + the NSString constructor.
func nsStringConvertArg(
	argName string,
	isNS bool,
	ctx typemap.Context,
	m *typemap.Mapper,
	imports typemap.ImportSet,
) string {
	if !isNS {
		return argName
	}
	owner := m.OwnerIndex["NSString"]
	if owner == "" || strings.EqualFold(owner, ctx.Framework) {
		return "NewNSString(cgo.GoStringToNSString(" + argName + "))"
	}
	pkg := strings.ToLower(owner)
	if imports != nil && m.ModulePrefix != "" {
		if _, ok := imports[pkg]; !ok {
			imports[pkg] = m.ModulePrefix + "/" + pkg
		}
	}
	return pkg + ".NewNSString(cgo.GoStringToNSString(" + argName + "))"
}

// writeNSStringInstanceOverload emits a "...Go" suffix instance-method overload
// where all NSString * args are replaced with plain Go string args.
func writeNSStringInstanceOverload(
	w io.Writer,
	goName, receiver string,
	method macosplatformmetadata.Method,
	ctx typemap.Context,
	m *typemap.Mapper,
	imports typemap.ImportSet,
) error {
	nsIdxs := nsStringArgIndices(method.Params)
	if len(nsIdxs) == 0 {
		return nil
	}
	overloadArgs := nsStringOverloadArgs(method.Params, method.IsNSError, nsIdxs, ctx, m, imports)
	idxSet := make(map[int]bool, len(nsIdxs))
	for _, i := range nsIdxs {
		idxSet[i] = true
	}
	resolved := buildParamNames(method.Params)

	goRet := buildGoReturn(method, ctx, m, "", imports)
	retStr := ""
	if goRet != "" {
		retStr = " " + goRet
	}

	var callArgs []string
	for i := range method.Params {
		callArgs = append(callArgs, nsStringConvertArg(resolved[i], idxSet[i], ctx, m, imports))
	}
	return render.Execute(w, "nsstring_overload", view.NsStringOverloadModel{
		Signature: fmt.Sprintf(
			"func (o *%s) %sGo(%s)%s",
			receiver,
			goName,
			strings.Join(overloadArgs, ", "),
			retStr,
		),
		HasReturn: goRet != "",
		CallExpr:  fmt.Sprintf("o.%s(%s)", goName, strings.Join(callArgs, ", ")),
	})
}

// writeNSStringClassOverload emits a "...Go" suffix class-method overload.
func writeNSStringClassOverload(
	w io.Writer,
	goName, className string,
	method macosplatformmetadata.Method,
	ctx typemap.Context,
	m *typemap.Mapper,
	pkgTypeNames map[string]bool,
	imports typemap.ImportSet,
) error {
	nsIdxs := nsStringArgIndices(method.Params)
	if len(nsIdxs) == 0 {
		return nil
	}
	overloadName := className + goName + "Go"
	if pkgTypeNames[overloadName] {
		return nil
	}
	overloadArgs := nsStringOverloadArgs(method.Params, method.IsNSError, nsIdxs, ctx, m, imports)
	idxSet := make(map[int]bool, len(nsIdxs))
	for _, i := range nsIdxs {
		idxSet[i] = true
	}
	resolved := buildParamNames(method.Params)

	goRet := buildGoReturn(method, ctx, m, className, imports)
	retStr := ""
	if goRet != "" {
		retStr = " " + goRet
	}

	var callArgs []string
	for i := range method.Params {
		callArgs = append(callArgs, nsStringConvertArg(resolved[i], idxSet[i], ctx, m, imports))
	}
	return render.Execute(w, "nsstring_overload", view.NsStringOverloadModel{
		Signature: fmt.Sprintf(
			"func %s(%s)%s",
			overloadName,
			strings.Join(overloadArgs, ", "),
			retStr,
		),
		HasReturn: goRet != "",
		CallExpr:  fmt.Sprintf("%s%s(%s)", className, goName, strings.Join(callArgs, ", ")),
	})
}

// receiverType returns the Go receiver type expression for instance methods.
func receiverType(className string, isGeneric bool) string {
	if isGeneric {
		return className + "[T]"
	}
	return className
}
