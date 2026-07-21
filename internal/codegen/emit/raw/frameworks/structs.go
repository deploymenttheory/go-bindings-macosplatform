package rawfw

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks/view"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emitmanifest"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// EmitStructs writes all struct type declarations for the framework to w.
// Returns cross-framework imports discovered during field type resolution.
//
// rec, when non-nil, records every emitted struct, field, and typedef alias into
// the parity manifest keyed on its ObjC/C name. It never affects emitted bytes.
func EmitStructs(
	w io.Writer,
	framework *meta.FrameworkMeta,
	mapper *typemap.Mapper,
	rec *emitmanifest.Recorder,
) (typemap.ImportSet, error) {
	imports := make(typemap.ImportSet)
	pkgName := naming.PackageName(framework.Framework)

	// Build the set of package-level identifiers structs must not collide
	// with: function names (raw C and exported Go forms), class wrapper
	// types, protocol interfaces, and enum types.
	funcNames := make(map[string]bool, len(framework.Functions)*2)
	for _, fn := range framework.Functions {
		funcNames[fn.Name] = true
		if goName := naming.ExportedFunctionName(fn.Name); goName != "" {
			funcNames[goName] = true
		}
	}
	reservedTypeNames := make(map[string]bool)
	for className := range framework.Classes {
		reservedTypeNames[className] = true
	}
	for protocolName := range framework.Protocols {
		reservedTypeNames[naming.GoTypeName(protocolName)] = true
	}
	for enumName := range framework.Enums {
		reservedTypeNames[naming.GoTypeName(enumName)] = true
	}

	names := make([]string, 0, len(framework.Structs))
	for name := range framework.Structs {
		names = append(names, name)
	}
	sort.Strings(names)

	seenGoNames := make(map[string]bool)
	var structs []view.Struct
	for _, name := range names {
		s := framework.Structs[name]
		if s.Availability.IsUnavailable {
			continue
		}
		goName := naming.ExportedTypeName(name)
		if goName == "" || seenGoNames[goName] || reservedTypeNames[goName] {
			continue
		}
		// Skip opaque (zero-field) structs whose name clashes with a function.
		// In Go, a type and function with the same package-level name is a
		// redeclaration error.
		if len(s.Fields) == 0 && funcNames[goName] {
			continue
		}
		seenGoNames[goName] = true
		built := buildStructView(name, s, framework.Framework, mapper, imports)
		structs = append(structs, built)
		rec.Record(emitmanifest.Entry{
			Style:     emitmanifest.StyleRaw,
			Kind:      emitmanifest.KindStruct,
			Framework: framework.Framework,
			MetaKey:   emitmanifest.MetaKey(framework.Framework, emitmanifest.KindStruct, name, ""),
			GoPkg:     pkgName,
			GoSymbol:  built.GoName,
		})
		// Struct-field-level parity is deferred until Phase 2 routes the idiomatic
		// emitter through this same BuildStructView, so both styles derive field
		// Go names identically; recording fields now (with divergent naming) would
		// report every field as a false gap.
	}
	out, err := render.Structs(structs)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(out); err != nil {
		return nil, err
	}

	// Emit typedef aliases (e.g. NSRect = CGRect). Struct Go names and other
	// reserved type names are passed so an alias whose exported name matches
	// its target struct (e.g. typedef NSRange ↔ struct _NSRange, both
	// exporting to NSRange) or another package-level type is skipped.
	for name := range reservedTypeNames {
		seenGoNames[name] = true
	}
	aliases := buildTypedefAliasViews(framework, mapper, imports, seenGoNames)
	for _, alias := range aliases {
		rec.Record(emitmanifest.Entry{
			Style:     emitmanifest.StyleRaw,
			Kind:      emitmanifest.KindTypedefAlias,
			Framework: framework.Framework,
			MetaKey:   emitmanifest.MetaKey(framework.Framework, emitmanifest.KindTypedefAlias, alias.GoName, ""),
			GoPkg:     pkgName,
			GoSymbol:  alias.GoName,
		})
	}
	aliasOut, err := render.TypedefAliases(aliases)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(aliasOut); err != nil {
		return nil, err
	}

	return imports, nil
}

// buildStructView resolves one struct into its renderable view: the comment
// block, whether it is opaque, and the exported fields with their resolved Go
// types. Cross-framework field types are accumulated into imports as a side
// effect, exactly as the original emitter did.
func buildStructView(
	name string,
	s meta.Struct,
	framework string,
	mapper *typemap.Mapper,
	imports typemap.ImportSet,
) view.Struct {
	goName := naming.ExportedTypeName(name)

	var comment strings.Builder
	if s.Doc != "" {
		fmt.Fprintf(&comment, "// %s\n", s.Doc)
	}
	comment.WriteString(deprecatedComment(s.Availability))
	if goName != name {
		fmt.Fprintf(&comment, "// C struct: %s\n", name)
	}

	if len(s.Fields) == 0 {
		fmt.Fprintf(&comment, "// %s is an opaque type.\n", goName)
		return view.Struct{CommentBlock: comment.String(), GoName: goName, IsOpaque: true}
	}

	built := view.Struct{CommentBlock: comment.String(), GoName: goName}
	for i, field := range s.Fields {
		fieldName := field.Name
		if fieldName == "" {
			fieldName = fmt.Sprintf("Field%d", i)
		}
		// Export first letter.
		if len(fieldName) > 0 && fieldName[0] >= 'a' && fieldName[0] <= 'z' {
			fieldName = strings.ToUpper(fieldName[:1]) + fieldName[1:]
		}
		// Skip underscore-prefixed private fields.
		if strings.HasPrefix(field.Name, "_") {
			continue
		}

		var goType string
		if field.GoType != "" {
			goType = field.GoType
		} else {
			ctx := typemap.Context{Framework: framework}
			goType = mapper.GoType(field.ObjCType, ctx, imports)
			if goType == "" {
				goType = "unsafe.Pointer"
			}
		}

		// If the resolved type references a cross-package unexported identifier
		// (starts with lowercase after the dot), fall back to unsafe.Pointer.
		if dotIdx := strings.LastIndex(goType, "."); dotIdx >= 0 {
			rest := goType[dotIdx+1:]
			rest = strings.TrimPrefix(rest, "*")
			if len(rest) > 0 && rest[0] >= 'a' && rest[0] <= 'z' {
				goType = "unsafe.Pointer"
			}
		}

		built.Fields = append(built.Fields, view.StructField{GoName: fieldName, GoType: goType})
	}
	return built
}

// isUnexportedCrossPackage reports whether a Go type string refers to an
// unexported (lowercase) identifier from another package, which would be a
// compile error.
func isUnexportedCrossPackage(goType string) bool {
	// Strip leading * for pointer types.
	t := strings.TrimPrefix(goType, "*")
	dotIdx := strings.LastIndex(t, ".")
	if dotIdx < 0 {
		return false // same package or primitive
	}
	after := t[dotIdx+1:]
	// Strip generic params
	if brIdx := strings.Index(after, "["); brIdx >= 0 {
		after = after[:brIdx]
	}
	return len(after) > 0 && after[0] >= 'a' && after[0] <= 'z'
}

func buildTypedefAliasViews(
	framework *meta.FrameworkMeta,
	mapper *typemap.Mapper,
	imports typemap.ImportSet,
	takenGoNames map[string]bool,
) []view.TypedefAlias {
	type alias struct {
		name, target string
		isPointer    bool // true for "typedef struct Foo *FooRef" patterns
	}
	var aliases []alias

	for tName, tTarget := range framework.Typedefs {
		t := strings.TrimSpace(tTarget)
		tConst := strings.TrimPrefix(t, "const ")
		tConst = strings.TrimSpace(tConst)

		if strings.HasPrefix(tConst, "struct ") {
			// "struct Foo" → plain struct alias (no pointer)
			bare := strings.TrimPrefix(tConst, "struct ")
			bare = strings.TrimSpace(bare)
			if _, ok := framework.Structs[bare]; ok && bare != tName {
				aliases = append(aliases, alias{tName, bare, false})
			}
			// "struct Foo *" → opaque pointer typedef (FooRef = *Foo)
			bareStar := strings.TrimSuffix(bare, "*")
			bareStar = strings.TrimSpace(bareStar)
			if strings.HasSuffix(bare, "*") {
				if _, ok := framework.Structs[bareStar]; ok {
					aliases = append(aliases, alias{tName, bareStar, true})
				}
			}
		}
	}
	sort.Slice(aliases, func(i, j int) bool { return aliases[i].name < aliases[j].name })

	var views []view.TypedefAlias
	emitted := make(map[string]bool)
	for _, a := range aliases {
		goAliasName := naming.ExportedTypeName(a.name)
		if goAliasName == "" || emitted[goAliasName] || takenGoNames[goAliasName] {
			continue
		}
		emitted[goAliasName] = true

		ctx := typemap.Context{Framework: framework.Framework}
		targetType := mapper.GoType(a.target, ctx, imports)
		if targetType == "" || targetType == "unsafe.Pointer" ||
			isUnexportedCrossPackage(targetType) {
			continue
		}
		if a.isPointer {
			views = append(views, view.TypedefAlias{
				CommentBlock: fmt.Sprintf("// %s is an opaque pointer to %s (C typedef %s).\n", goAliasName, a.target, a.name),
				GoName:       goAliasName,
				RHS:          "*" + targetType,
			})
		} else {
			views = append(views, view.TypedefAlias{
				CommentBlock: fmt.Sprintf("// %s is an alias for %s (C typedef %s).\n", goAliasName, a.target, a.name),
				GoName:       goAliasName,
				RHS:          targetType,
			})
		}
	}
	return views
}
