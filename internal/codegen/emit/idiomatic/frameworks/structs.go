//go:build darwin

package idiofw

import (
	"fmt"
	"maps"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/view"
	rawfw "github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emitmanifest"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// goPrimitives is the set of Go primitive types a value-struct field may use
// directly. A field of any other bare type must be another emittable struct in
// the same package (checked via the emittable fixpoint below).
var goPrimitives = map[string]bool{
	"bool": true, "byte": true, "rune": true, "uintptr": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
}

// emitStructTypeAliases writes <pkgname>_type_aliases_generated.go: a Go
// definition for every plain value-type struct the framework defines (e.g.
// CGSize, CGRect, CGPoint, NSRange), so callers can name and build them through
// the idiomatic package. The struct is re-declared with the same field types and
// order as the system framework expects, so it can be passed to Objective-C by
// value.
//
// Field Go types are resolved from each field's Objective-C type (the scanner
// records only that), so CGFloat becomes float64 and CGPoint stays CGPoint. A
// struct is emitted only when every field is a Go primitive or another emittable
// struct in this same package; structs with pointer, runtime, or cross-framework
// fields are skipped. Runs after emitEnums so takenNames already covers every
// other package-level identifier, preventing a redeclare.
// resolveStructFields resolves a struct's field names and Go types from their
// Objective-C types. ok is false when any field type cannot be resolved.
func resolveStructFields(
	s meta.Struct,
	ctx typemap.Context,
	mapper *typemap.Mapper,
) (names, goTypes []string, ok bool) {
	for _, f := range s.Fields {
		gt := mapper.GoType(f.ObjCType, ctx, make(typemap.ImportSet))
		if gt == "" {
			return nil, nil, false
		}
		names = append(names, f.Name)
		goTypes = append(goTypes, gt)
	}
	return names, goTypes, len(names) > 0
}

// ComputeEmittableStructs returns the set of value-struct Go names (across every
// framework) the idiomatic layer can emit a definition for: a struct is
// emittable when every field is a Go primitive or another emittable struct.
// Iterates to a fixpoint so a struct may depend on one resolved later (CGRect on
// CGPoint/CGSize). The same set gates both the per-framework struct emission and
// cross-framework references, so a reference never names a struct that was
// skipped. An enum-typed field is allowed when the enum is one the same framework
// emits locally (see ComputeEmittableStructs).
func ComputeEmittableStructs(
	frameworks []*meta.FrameworkMeta,
	mapper *typemap.Mapper,
) map[string]bool {
	// Per-framework own-enum names (bare), so an enum-typed field is accepted only
	// when the enum belongs to the same framework as the struct — the only case the
	// idiomatic package emits a local definition for (registerLocalStructEnumRefs).
	ownEnumsByFw := make(map[string]map[string]bool, len(frameworks))
	resolved := make(map[string][]string) // struct Go name → field Go types
	structFw := make(map[string]string)   // struct Go name → owning framework
	for _, framework := range frameworks {
		ownEnumsByFw[framework.Framework] = buildOwnEnumNames(framework)
		ctx := typemap.Context{Framework: framework.Framework}
		for name, s := range framework.Structs {
			if s.Availability.IsUnavailable || len(s.Fields) == 0 {
				continue
			}
			goName := naming.ExportedTypeName(name)
			if !isExportedGoIdent(goName) {
				continue
			}
			if _, dup := resolved[goName]; dup {
				continue
			}
			if _, goTypes, ok := resolveStructFields(s, ctx, mapper); ok {
				resolved[goName] = goTypes
				structFw[goName] = framework.Framework
			}
		}
	}

	emittable := make(map[string]bool)
	for {
		changed := false
		for goName, goTypes := range resolved {
			if emittable[goName] {
				continue
			}
			ownEnums := ownEnumsByFw[structFw[goName]]
			good := true
			for _, gt := range goTypes {
				// A field is acceptable when it is a Go primitive, another emittable
				// struct, or one of this struct's own framework's emitted enums (a
				// bare name; cross-framework or non-emitted enums are rejected so the
				// struct never names a type this package does not define).
				if goPrimitives[gt] || emittable[gt] || ownEnums[gt] {
					continue
				}
				good = false
				break
			}
			if good {
				emittable[goName] = true
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return emittable
}

// ComputeAllEmittedStructNames returns every value-struct Go name the idiomatic
// layer physically writes across all frameworks — the broad, degrade-don't-drop
// set emitStructs actually emits (available, exported, first-seen), INCLUDING
// opaque structs and ones with degraded fields that ComputeEmittableStructs
// (the clean subset) omits. A cross-framework typedef alias needs its target to
// merely exist, so it consults this set. Name collisions with a class/enum in
// the owning framework are resolved per-framework at emit time via `taken`,
// which is not known globally here; the small over-approximation is harmless —
// a genuinely skipped target would dangle and be caught by the build.
func ComputeAllEmittedStructNames(frameworks []*meta.FrameworkMeta) map[string]bool {
	all := make(map[string]bool)
	for _, framework := range frameworks {
		for name, s := range framework.Structs {
			if s.Availability.IsUnavailable {
				continue
			}
			goName := naming.ExportedTypeName(name)
			if !isExportedGoIdent(goName) {
				continue
			}
			all[goName] = true
		}
	}
	return all
}

// registerLocalStructEnumRefs marks the framework's own enums that appear as
// fields of a value struct this package emits (e.g. hv_vcpu_exit_t's reason field
// of type Hv_exit_reason_t) as referenced, so emitEnums — which runs before
// emitStructs — emits a local definition and the struct field names a hermetic
// local type rather than the raw package's. It must cover every struct emitStructs
// will emit (not just the "simple" emittable set), or an own-enum field would name
// a type emitEnums never wrote. takenNames is already fully populated at call time.
func registerLocalStructEnumRefs(
	fc *frameworkContext,
	mapper *typemap.Mapper,
	takenNames map[string]bool,
) {
	ctx := typemap.Context{Framework: fc.framework.Framework}
	_, _, structOf := emittableStructNames(fc.framework, takenNames)
	for _, s := range structOf {
		for _, f := range s.Fields {
			if strings.HasPrefix(f.Name, "_") {
				continue
			}
			gt := f.GoType
			if gt == "" {
				gt = mapper.GoType(f.ObjCType, ctx, make(typemap.ImportSet))
			}
			if fc.ownEnums[gt] {
				fc.referenced[gt] = true
			}
		}
	}
}

// structFieldGoName derives the exported Go field name for a C struct field:
// snake_case becomes PascalCase with known initialisms re-cased
// (virtual_address → VirtualAddress), plain names get their first letter
// capitalised. Names only — the field order and types are ABI and untouched.
func structFieldGoName(fieldName string) string {
	if exported := naming.ExportedTypeName(fieldName); exported != "" {
		return applyInitialisms(exported)
	}
	return capitalizeFirst(fieldName)
}

// emittableStructNames returns the value structs this framework will emit: every
// available, exported, non-duplicate struct whose Go name is not already claimed
// by another package-level identifier (a class wrapper, enum, function, provider,
// …). Both field-bearing and opaque (zero-member) structs qualify. The result
// gates same-package field references so a field never names a struct that was
// skipped. taken is the fully-populated set of claimed names at struct-emission
// time (structs are emitted last, after every other construct).
func emittableStructNames(
	framework *meta.FrameworkMeta,
	taken map[string]bool,
) (goNames []string, keyOf map[string]string, structOf map[string]meta.Struct) {
	keyOf = make(map[string]string)
	structOf = make(map[string]meta.Struct)
	seen := make(map[string]bool)
	for name, s := range framework.Structs {
		if s.Availability.IsUnavailable {
			continue
		}
		goName := naming.ExportedTypeName(name)
		if !isExportedGoIdent(goName) || seen[goName] || taken[goName] {
			continue
		}
		seen[goName] = true
		keyOf[goName] = name
		structOf[goName] = s
		goNames = append(goNames, goName)
	}
	sort.Strings(goNames)
	return goNames, keyOf, structOf
}

// emitStructs writes <pkgname>_structs_generated.go: a Go definition for every
// value-type struct the framework declares, so callers can name and build them
// through the idiomatic package and pass them to Objective-C by value. It mirrors
// the raw layer's inclusion policy (degrade-don't-drop): a struct is always
// emitted, an opaque one as `struct{}`, and a field whose type cannot be named
// hermetically degrades to unsafe.Pointer rather than dropping the whole struct.
//
// Field Go types are resolved from each field's Objective-C type: a Go primitive
// or array-of-primitive is kept (correct ABI), an own-framework enum names the
// local idiomatic enum type, another value struct (this framework's or a sibling
// idiomatic package's) is named directly, and anything else — pointers, ObjC
// objects, function pointers, cross-package unexported types — becomes
// unsafe.Pointer. Runs last, after every other package-level identifier is
// claimed, so a struct name never redeclares one.
func emitStructs(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	fc *frameworkContext,
	mapper *typemap.Mapper,
	takenNames map[string]bool,
) error {
	_ = rawPkgPath
	framework := fc.framework
	ctx := typemap.Context{Framework: framework.Framework}

	goNames, keyOf, structOf := emittableStructNames(framework, takenNames)
	willEmit := make(map[string]bool, len(goNames))
	for _, goName := range goNames {
		willEmit[goName] = true
	}

	imports := map[string]string{}
	var structs []view.Struct
	for _, goName := range goNames {
		takenNames[goName] = true
		s := structOf[goName]
		if fc.manifest != nil {
			fc.manifest.Record(emitmanifest.Entry{
				Style:     emitmanifest.StyleIdiomatic,
				Kind:      emitmanifest.KindStruct,
				Framework: framework.Framework,
				MetaKey: emitmanifest.MetaKey(
					framework.Framework,
					emitmanifest.KindStruct,
					keyOf[goName],
					"",
				),
				GoPkg:    pkgName,
				GoSymbol: goName,
			})
		}
		if len(s.Fields) == 0 {
			structs = append(
				structs,
				view.Struct{GoName: goName, Doc: cleanDoc(s.Doc), IsOpaque: true},
			)
			continue
		}
		var fields []view.Field
		for i, f := range s.Fields {
			// Skip underscore-prefixed private members (bitfields, padding),
			// exactly as the raw emitter does — a struct whose members are all
			// skipped renders with an empty body, not as opaque.
			if strings.HasPrefix(f.Name, "_") {
				continue
			}
			fieldName := f.Name
			if fieldName == "" {
				fieldName = fmt.Sprintf("Field%d", i)
			}
			gt := hermeticFieldType(f, ctx, mapper, fc, rawPkgAlias, willEmit, imports)
			fields = append(fields, view.Field{GoName: structFieldGoName(fieldName), GoType: gt})
		}
		structs = append(structs, view.Struct{GoName: goName, Doc: cleanDoc(s.Doc), Fields: fields})
	}
	// Typedef aliases (NSRect = CGRect, opaque-pointer FooRef = *Foo) share the
	// file, matching the raw layer's single _structs.go. willEmit lets an alias
	// reference a struct emitted just above; taken names avoid a redeclaration.
	aliases := buildTypedefAliasViews(
		framework,
		mapper,
		fc,
		rawPkgAlias,
		willEmit,
		takenNames,
		imports,
	)

	if len(structs) == 0 && len(aliases) == 0 {
		return nil
	}

	body, err := render.Structs(structs)
	if err != nil {
		return err
	}
	if len(aliases) > 0 {
		aliasBody, err := render.TypedefAliases(aliases)
		if err != nil {
			return err
		}
		body = append(body, aliasBody...)
	}
	fname := pkgName + "_structs_generated.go"
	file := assembleFile(pkgName, imports, body)
	return rawfw.WriteGoFile(filepath.Join(outDir, fname), file)
}

// buildTypedefAliasViews resolves the framework's C struct typedefs into idiomatic
// Go type aliases, mirroring the raw emitter's selection (a `struct Foo` typedef
// becomes `Name = Foo`, a `struct Foo *` typedef the opaque-pointer `Name = *Foo`)
// but localizing the right-hand side so the alias stays hermetic: the aliased
// struct must be one this package emits (willEmit) or a sibling idiomatic package's
// value struct. An alias whose name is already claimed, or whose target cannot be
// named hermetically, is skipped. Names it does emit are added to takenNames, and
// any sibling-package import is accumulated into imports.
func buildTypedefAliasViews(
	framework *meta.FrameworkMeta,
	mapper *typemap.Mapper,
	fc *frameworkContext,
	rawPkgAlias string,
	willEmit, takenNames map[string]bool,
	imports map[string]string,
) []view.TypedefAlias {
	type alias struct {
		name, target string
		isPointer    bool
	}
	var candidates []alias
	for tName, tTarget := range framework.Typedefs {
		t := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(tTarget), "const "))
		if !strings.HasPrefix(t, "struct ") {
			continue
		}
		bare := strings.TrimSpace(strings.TrimPrefix(t, "struct "))
		if base, isPtr := strings.CutSuffix(bare, "*"); isPtr {
			base = strings.TrimSpace(base)
			// The target struct need not live in this framework: an opaque-pointer
			// typedef often aliases a struct owned by another framework (e.g.
			// AudioComponentInstance → CarbonCore's ComponentInstanceRecord, the
			// Kerberos GSS typedefs → the gss package's structs). The resolution
			// pass below localizes and existence-checks the target, so a
			// non-nameable one is dropped there rather than here.
			candidates = append(candidates, alias{tName, base, true})
			continue
		}
		if bare != tName {
			candidates = append(candidates, alias{tName, bare, false})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].name < candidates[j].name })

	ctx := typemap.Context{Framework: framework.Framework}
	var out []view.TypedefAlias
	for _, a := range candidates {
		goName := naming.ExportedTypeName(a.name)
		if !isExportedGoIdent(goName) || takenNames[goName] {
			continue
		}
		// The target is a struct name; resolve it to the Go type this package (or a
		// sibling) names, then confirm it is actually emitted so the alias never
		// dangles.
		targetType := mapper.GoType(a.target, ctx, make(typemap.ImportSet))
		if targetType == "" {
			continue
		}
		var rhsCore string
		if !strings.ContainsAny(targetType, ".*[] ") && willEmit[targetType] {
			// Same-framework value struct emitted just above.
			rhsCore = targetType
		} else if resolved, imps, ok := crossFrameworkEmittedStruct(targetType, mapper, rawPkgAlias); ok {
			// Cross-framework struct the idiomatic layer emits (opaque/degraded
			// targets are fine — an alias names the type, never its fields).
			maps.Copy(imports, imps)
			rhsCore = resolved
		} else {
			continue // not hermetically nameable
		}
		takenNames[goName] = true
		rhs := rhsCore
		if a.isPointer {
			rhs = "*" + rhsCore
		}
		if fc.manifest != nil {
			fc.manifest.Record(emitmanifest.Entry{
				Style:     emitmanifest.StyleIdiomatic,
				Kind:      emitmanifest.KindTypedefAlias,
				Framework: framework.Framework,
				MetaKey: emitmanifest.MetaKey(
					framework.Framework,
					emitmanifest.KindTypedefAlias,
					goName,
					"",
				),
				GoPkg:    naming.PackageName(framework.Framework),
				GoSymbol: goName,
			})
		}
		out = append(out, view.TypedefAlias{
			Doc:    fmt.Sprintf("%s is an alias for the %s value type.", goName, a.target),
			GoName: goName,
			RHS:    rhs,
		})
	}
	return out
}

// hermeticFieldType resolves one struct field to a Go type the idiomatic package
// can name without importing the raw bindings. It keeps primitives and arrays of
// primitives (correct ABI), names an own-framework enum by its local idiomatic
// spelling, names a same-package or sibling-package value struct directly, and
// degrades everything else to unsafe.Pointer. Imports needed to name a type
// (unsafe, or a sibling idiomatic package) are accumulated into imports.
func hermeticFieldType(
	f meta.StructField,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	fc *frameworkContext,
	rawPkgAlias string,
	willEmit map[string]bool,
	imports map[string]string,
) string {
	gt := f.GoType
	if gt == "" {
		gt = mapper.GoType(f.ObjCType, ctx, make(typemap.ImportSet))
	}
	degrade := func() string {
		imports["unsafe"] = "unsafe"
		return "unsafe.Pointer"
	}
	if gt == "" {
		return degrade()
	}
	if gt == "unsafe.Pointer" {
		return degrade()
	}
	// An own-framework enum field names the local idiomatic enum type (emitted by
	// emitEnums, which registerLocalStructEnumRefs primed to include it).
	if fc.ownEnums[gt] {
		return fc.localEnumTypeName(gt)
	}
	// A sibling framework's value struct (e.g. corefoundation.CGRect): name it
	// through the sibling idiomatic package.
	if resolved, imps, ok := crossFrameworkValueStruct(gt, mapper, rawPkgAlias); ok {
		maps.Copy(imports, imps)
		return resolved
	}
	// A Go primitive, or an array of one, keeps its ABI and needs no import.
	if isPrimitiveOrArrayOf(gt, func(elem string) bool { return goPrimitives[elem] }) {
		return gt
	}
	// A same-package value struct that will be emitted (bare name, or an array of
	// one) can be named directly.
	if isPrimitiveOrArrayOf(gt, func(elem string) bool { return willEmit[elem] }) {
		return gt
	}
	return degrade()
}

// isPrimitiveOrArrayOf reports whether goType is a bare identifier accepted by
// ok, or a (possibly multi-dimensional) fixed-size array whose element type is.
// It rejects pointers, slices, and qualified (dotted) names.
func isPrimitiveOrArrayOf(goType string, ok func(elem string) bool) bool {
	elem := goType
	for strings.HasPrefix(elem, "[") {
		close := strings.IndexByte(elem, ']')
		if close < 0 {
			return false
		}
		inner := elem[1:close]
		if inner == "" { // a slice "[]T", not a fixed array — not ABI-safe
			return false
		}
		elem = elem[close+1:]
	}
	if elem == "" || strings.ContainsAny(elem, ".*[] ") {
		return false
	}
	return ok(elem)
}

// ComputeIdiomaticClassIndex maps every ObjC class emitted by the idiomatic
// layer to its owning idiomatic package and wrapper type name ("NSProgress" →
// {foundation, Progress}). Only each class's canonical owner (ownerIndex)
// contributes, and only frameworks the idiomatic generator actually emits
// (callers pass the already-filtered framework list), so a reference built
// from this index always names a type that exists. Every available class in an
// emitted framework gets a wrapper file — a non-abstract class always carries
// at least its +new constructor and an abstract base is emitted for embedding
// — so no per-class emission check is needed.
func ComputeIdiomaticClassIndex(
	frameworks []*meta.FrameworkMeta,
	ownerIndex map[string]string,
) map[string]typemap.IdiomaticClassRef {
	index := make(map[string]typemap.IdiomaticClassRef)
	for _, framework := range frameworks {
		prefix := detectClassPrefix(framework)
		packageName := naming.PackageName(framework.Framework)
		for className, class := range framework.Classes {
			if class.Availability.IsUnavailable {
				continue
			}
			if ownerIndex[className] != framework.Framework {
				continue
			}
			typeName := trialTypeName(className, prefix)
			if !isExportedGoIdent(typeName) {
				continue
			}
			index[className] = typemap.IdiomaticClassRef{
				Package:  packageName,
				TypeName: typeName,
			}
		}
	}
	return index
}
