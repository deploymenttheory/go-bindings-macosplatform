//go:build darwin

package idiofw

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"slices"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// classHeaderView is the template data for class_header.tmpl.
type classHeaderView struct {
	DocComment string
	GoTypeName string
	RecvVar    string // receiver variable for the promoted root methods, e.g. "vm"
	ClassName  string // Objective-C class name, for the doc line
	AdoptName  string // unexported +1-adopt helper, e.g. "virtualMachineAdopt"
	// BaseType is the same-framework wrapper type this class embeds, promoting
	// its methods; empty when this class is a framework root.
	BaseType string
	// EmbedsRoot is true when the class is a framework root: it embeds
	// objref.Handle directly and carries the obj.Object methods + String that
	// every subclass then promotes.
	EmbedsRoot bool
	// HasOwnString is true when the class declares a genuine method named String
	// (a real `string` property, e.g. VNRecognizedText.string). It overrides the
	// synthesized fmt stringer, which is then suppressed to avoid a duplicate.
	HasOwnString bool
}

// classSealView is the template data for class_seal.tmpl.
type classSealView struct {
	GoTypeName string
	// RecvVar is the receiver variable for the sealing marker method, e.g. "bl".
	RecvVar string
	// MarkerName is the sealing method this class defines when it is itself a
	// provider base, or "" when it is not.
	MarkerName string
	// ProviderIfaces are the provider interfaces this class satisfies, for the
	// compile-time var _ Iface = (*T)(nil) assertions (E11).
	ProviderIfaces []string
}

// classProviderIfaces returns the provider interface names a class satisfies
// through its embed chain — one per provider-base ancestor it actually embeds
// (transitively), so the marker method is genuinely promoted and the assertion
// compiles. The class's own provider (when it is itself a base) is added by the
// caller. Walking the embed chain rather than the raw super chain means a chain
// broken by a root or a de-prefix collision stops the promotion here too.
func classProviderIfaces(
	class meta.Class,
	goTypeName string,
	fc *frameworkContext,
	prefix string,
	abstractBases abstractBaseIndex,
) []string {
	var ifaces []string
	cur, curGo := class, goTypeName
	for {
		base := sameFrameworkBase(cur, curGo, fc, prefix)
		if base == "" {
			break
		}
		// cur.Super is a provider base (something subclasses it — at least cur);
		// the embedded base type promotes its marker to this class.
		if _, isBase := abstractBases[cur.Super]; isBase {
			ifaces = append(ifaces, providerInterfaceName(base))
		}
		sup, ok := fc.framework.Classes[cur.Super]
		if !ok {
			break
		}
		cur, curGo = sup, base
	}
	return ifaces
}

// sameFrameworkBase returns the Go wrapper type a class should embed: its
// Objective-C superclass when that superclass is an available, distinct,
// actually-emitted class in the same framework (so the subclass promotes the
// base's methods through Go embedding), or "" when the class is a root — which
// embeds objref.Handle and defines the obj.Object surface the subclasses promote.
//
// NSObject and NSProxy are the universal roots: a class derived from one of them
// is treated as a root (it embeds objref.Handle), never as embedding the
// Foundation Object/Proxy wrapper — that wrapper's bare de-prefixed name would
// also collide with common members like an `object` property. The collision and
// emitted-set guards keep a de-prefix clash (two ObjC classes mapping to one Go
// name) from producing an embed of an undefined or self type.
func sameFrameworkBase(
	class meta.Class,
	goTypeName string,
	fc *frameworkContext,
	prefix string,
) string {
	switch class.Super {
	case "", "NSObject", "NSProxy":
		return ""
	}
	super, ok := fc.framework.Classes[class.Super]
	if !ok || super.Availability.IsUnavailable {
		return ""
	}
	baseGo := trialTypeName(class.Super, prefix)
	if baseGo == goTypeName || !fc.classGoNames[baseGo] {
		return ""
	}
	return baseGo
}

func emitClassFile(
	w io.Writer,
	pkgName, rawPkgAlias, rawPkgPath string,
	className, goTypeName string,
	class meta.Class,
	fc *frameworkContext,
	mapper *typemap.Mapper,
	trialNames trialNameMap,
	prefix string,
	abstractBases abstractBaseIndex,
	handFuncs map[string]bool,
	handMethods map[string]bool,
) error {
	framework := fc.framework
	ctx := typemap.Context{
		ClassName:     className,
		Framework:     framework.Framework,
		GenericParams: class.GenericParams,
	}

	ctors := buildConstructors(
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
	withMethods := buildWithSetters(
		class,
		goTypeName,
		fc,
		ctx,
		mapper,
		rawPkgAlias,
		trialNames,
		prefix,
		abstractBases,
	)
	methods := buildMethods(class, fc, ctx, mapper, rawPkgAlias, trialNames, abstractBases)
	providerMethods, providerImports := buildProviderMethods(
		class, className, goTypeName, framework, mapper, rawPkgAlias, abstractBases)

	// Flatten inherited With* setters onto the subclass wrapper. A wrapper is
	// otherwise built from only its own class's properties, so anything configured
	// through an inherited setter (e.g. VZNetworkDeviceConfiguration.attachment on
	// a VZVirtio… subclass) is missing — the gap that forced consumers back to the
	// raw bindings. Walk the superclass chain within this framework and re-emit the
	// base classes' fluent setters on the subclass; they dispatch through x.inner,
	// which has the underlying setter via raw struct embedding. Subclass-own and
	// nearer-ancestor setters win on name collision. Only setters are flattened:
	// flattening arbitrary methods reintroduces base/override arity clashes through
	// embedding and out-of-package return types.
	withMethods = append(
		withMethods,
		buildInheritedSetters(
			class,
			className,
			fc,
			mapper,
			rawPkgAlias,
			trialNames,
			prefix,
			abstractBases,
			withMethods,
		)...)

	// The wrapper struct already provides Description/IsEqual/IsKind (so it can be
	// used as an obj.Object) and embeds objref.Handle (which promotes a field
	// named Handle and the lifecycle method Release). Drop any generated method
	// or setter that would redeclare one of those names.
	objMethodNames := map[string]bool{
		"Description": true, "IsEqual": true, "IsKind": true, "Handle": true,
		"Release": true,
	}
	methods = slices.DeleteFunc(
		methods,
		func(e methodModel) bool { return objMethodNames[e.goName] },
	)
	withMethods = slices.DeleteFunc(
		withMethods,
		func(e withSetterModel) bool { return objMethodNames[e.goName] },
	)

	// Drop the plain void Set* accessor for any property already exposed through a
	// fluent With* setter: the two send the same Objective-C setter selector, so
	// the void form is redundant surface. Matching is by selector (exact), not by
	// name. The chainable With* form is kept as the single mutator.
	if len(withMethods) > 0 {
		coveredSetters := make(map[string]bool, len(withMethods))
		for _, setter := range withMethods {
			if setter.setterSelector != "" {
				coveredSetters[setter.setterSelector] = true
			}
		}
		methods = slices.DeleteFunc(
			methods,
			func(e methodModel) bool { return coveredSetters[e.selector] },
		)
	}

	// Drop anything a hand-authored file in this package already declares, so the
	// human's version wins (no duplicate-method compile error).
	if len(handFuncs) > 0 {
		ctors = slices.DeleteFunc(
			ctors,
			func(c constructorModel) bool { return handFuncs[c.goName] },
		)
	}
	if len(handMethods) > 0 {
		withMethods = slices.DeleteFunc(
			withMethods,
			func(e withSetterModel) bool { return handMethods[e.goName] },
		)
		methods = slices.DeleteFunc(
			methods,
			func(e methodModel) bool { return handMethods[e.goName] },
		)
		providerMethods = slices.DeleteFunc(
			providerMethods,
			func(e providerMethodEntry) bool { return handMethods[e.methodName] },
		)
	}

	// WS3 (P10): an abstract base — a class some other same-framework class
	// subclasses — is never constructed directly via a bare +new (New<Base>() is
	// meaningless), so that constructor is dropped. A class cluster's base
	// (NSNumber, NSData, NSString — all subclassed by Mutable/Decimal variants) is
	// still concretely constructible through its designated initializers, so those
	// parameterized constructors are kept. A pure abstract base (VZBootLoader) has
	// no such inits and so ends up with no constructors. The wrapper is still
	// emitted either way: subclasses embed it and it carries the sealing marker.
	isAbstract := abstractBases[className] != ""
	if isAbstract {
		kept := ctors[:0]
		for _, c := range ctors {
			if c.rawInitGoName == "" {
				continue // drop the bare +new constructor
			}
			kept = append(kept, c)
		}
		ctors = kept
	}

	// Ensure a +new constructor exists for provider-only classes (no other content).
	if !isAbstract && len(ctors) == 0 && len(providerMethods) > 0 {
		ctors = []constructorModel{buildNewConstructor(className, goTypeName, rawPkgAlias)}
	}

	// Skip classes that have nothing useful to emit. An abstract base is always
	// emitted (it is embedded and sealed). When a hand-authored file declares
	// methods on this type, still emit the wrapper struct so those files have a
	// type to hang from.
	if !isAbstract && len(ctors) == 0 && len(withMethods) == 0 && len(methods) == 0 &&
		len(providerMethods) == 0 && len(handMethods) == 0 {
		return nil
	}

	// ── Render body, then derive imports from what the code actually uses ──────

	var body bytes.Buffer

	// Hermetic wrapper struct: a framework root embeds objref.Handle and carries
	// the obj.Object surface (+ String); a subclass embeds its same-framework base
	// and promotes that surface. No raw inner, no Unwrap/ID — dispatch goes
	// straight to the runtime.
	baseType := sameFrameworkBase(class, goTypeName, fc, prefix)
	embedsRoot := baseType == ""
	subLinks := ""
	if isAbstract {
		subLinks = directSubclassLinks(className, fc, prefix)
	}
	hasOwnString := false
	for i := range methods {
		if methods[i].goName == "String" {
			hasOwnString = true
			break
		}
	}
	render.Must(&body, "class_header", classHeaderView{
		DocComment: buildClassDoc(
			goTypeName,
			className,
			isAbstract,
			subLinks,
			baseType,
			class.Doc,
		),
		GoTypeName:   goTypeName,
		RecvVar:      receiverName(goTypeName),
		ClassName:    className,
		AdoptName:    adoptHelperName(goTypeName),
		BaseType:     baseType,
		EmbedsRoot:   embedsRoot,
		HasOwnString: hasOwnString,
	})

	for _, c := range ctors {
		writeConstructor(
			&body,
			goTypeName,
			rawPkgAlias,
			className,
			genericInstantiation(len(class.GenericParams)),
			c,
		)
	}
	for _, setter := range withMethods {
		writeWithMethod(&body, goTypeName, setter)
	}
	for _, method := range methods {
		writeMethod(&body, goTypeName, method)
	}
	provItems := make([]providerMethodItem, len(providerMethods))
	for i, pm := range providerMethods {
		provItems[i] = providerMethodItem{
			MethodName: pm.methodName,
			RawType:    pm.rawType,
			BodyExpr:   pm.bodyExpr,
		}
	}
	render.Must(
		&body,
		"provider_method",
		providerMethodView{GoTypeName: goTypeName, Items: provItems},
	)

	emitDictionaryAugment(
		&body,
		className,
		goTypeName,
		rawPkgAlias,
		genericInstantiation(len(class.GenericParams)),
	)

	// Sealing: a class that is itself a provider base defines the unexported
	// marker method; every class also gets a compile-time assertion that it
	// satisfies each provider interface its embed chain grants it (E11).
	seal := classSealView{
		GoTypeName:     goTypeName,
		RecvVar:        receiverName(goTypeName),
		ProviderIfaces: classProviderIfaces(class, goTypeName, fc, prefix, abstractBases),
	}
	if _, isBase := abstractBases[className]; isBase {
		seal.MarkerName = markerMethodName(goTypeName)
		seal.ProviderIfaces = append(
			[]string{providerInterfaceName(goTypeName)},
			seal.ProviderIfaces...)
	}
	if seal.MarkerName != "" || len(seal.ProviderIfaces) > 0 {
		render.Must(&body, "class_seal", seal)
	}

	imports := classFileImports(ctors, withMethods, methods, providerImports, embedsRoot)
	if className == "NSMutableDictionary" {
		// The dictionary augment helpers (Set/SetString/Get) use obj (which a
		// non-root class would not otherwise import) and keep their object
		// arguments alive across the send via runtime.KeepAlive.
		imports["obj"] = objImportPath
		imports["runtime"] = "runtime"
	}

	// ── Write output ───────────────────────────────────────────────────────────

	if _, err := w.Write(assembleFile(pkgName, imports, body.Bytes())); err != nil {
		return fmt.Errorf("write class body: %w", err)
	}

	return nil
}

// classFileImports computes the import set for a class file from the resolved
// constructs, rather than by scanning the rendered text. The class header always
// dispatches through the runtime (objref/objc/purego); a framework root also
// carries the obj.Object surface (obj/rt), while a subclass promotes that surface
// from its embedded base and so names obj/rt only when one of its own constructs
// does. Every other import is contributed by a construct that uses it: each
// construct's extraImports already records its type and runtime imports, and an
// error-returning constructor additionally needs errkit and unsafe (its template,
// not its builder, references them). The raw bindings package is never imported
// (the hermetic invariant), so it is intentionally absent.
func classFileImports(
	ctors []constructorModel,
	withMethods []withSetterModel,
	methods []methodModel,
	providerImports map[string]string,
	embedsRoot bool,
) map[string]string {
	imports := map[string]string{
		"objref": objrefImportPath,
		"objc":   objcImportPath,
		"purego": pureobjcImportPath,
	}
	if embedsRoot {
		// Only a root defines Description/IsEqual/IsKind/String, which use rt (and
		// keep their receiver alive across the send via runtime.KeepAlive); its
		// IsEqual also names obj.Object. A subclass promotes that surface from its
		// embedded base and names obj only when one of its own constructs does
		// (recorded in that construct's extraImports below).
		imports["rt"] = rtImportPath
		imports["obj"] = objImportPath
		imports["runtime"] = "runtime"
	}
	for _, c := range ctors {
		maps.Copy(imports, c.extraImports)
		if c.hasNSError {
			imports["unsafe"] = "unsafe"
			imports["errkit"] = errkitImportPath
		}
	}
	for _, setter := range withMethods {
		maps.Copy(imports, setter.extraImports)
	}
	for _, method := range methods {
		maps.Copy(imports, method.extraImports)
	}
	maps.Copy(imports, providerImports)
	return imports
}

// isReservedMemberName reports whether a generated Go method or setter name
// collides with a name the wrapper already provides through embedding: the
// obj.Object surface (Description/IsEqual/IsKind), the objref.Handle field, or
// the lifecycle method Release the Handle promotes. Such a member is dropped —
// the embedded base supplies it, promoted to every subclass (spec edge case
// #14, E13).
//
// "String" is deliberately NOT reserved here: the synthesized fmt stringer only
// returns -description, but a genuine same-named property carries distinct data
// (e.g. VNRecognizedText.string is the recognized text). Such a property is
// emitted and overrides the stringer — for a framework root the synthesized
// String() is suppressed via classHeaderView.HasOwnString to avoid a duplicate;
// a subclass's emitted String() simply shadows the promoted one.
func isReservedMemberName(name string) bool {
	switch name {
	case "Description", "IsEqual", "IsKind", "Handle", "Release":
		return true
	}
	return false
}
