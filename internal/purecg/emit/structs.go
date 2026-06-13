package emit

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/purecg/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/purecg/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/purecg/typemap"
)

// EmitStructs writes all struct type declarations for the framework to w.
// Returns cross-framework imports discovered during field type resolution.
func EmitStructs(
	w io.Writer,
	framework *meta.FrameworkMeta,
	mapper *typemap.Mapper,
) (typemap.ImportSet, error) {
	imports := make(typemap.ImportSet)

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
		if err := emitStruct(w, name, s, framework.Framework, mapper, imports); err != nil {
			return nil, err
		}
	}

	// Emit typedef aliases (e.g. NSRect = CGRect). Struct Go names and other
	// reserved type names are passed so an alias whose exported name matches
	// its target struct (e.g. typedef NSRange ↔ struct _NSRange, both
	// exporting to NSRange) or another package-level type is skipped.
	for name := range reservedTypeNames {
		seenGoNames[name] = true
	}
	if err := emitTypedefAliases(w, framework, mapper, imports, seenGoNames); err != nil {
		return nil, err
	}

	return imports, nil
}

func emitStruct(
	w io.Writer,
	name string,
	s meta.Struct,
	framework string,
	mapper *typemap.Mapper,
	imports typemap.ImportSet,
) error {
	if s.Doc != "" {
		fmt.Fprintf(w, "// %s\n", s.Doc)
	}
	emitDeprecatedComment(w, s.Availability)

	goName := naming.ExportedTypeName(name)
	if goName != name {
		fmt.Fprintf(w, "// C struct: %s\n", name)
	}

	// Opaque struct (zero fields) — emit as unsafe.Pointer alias.
	if len(s.Fields) == 0 {
		fmt.Fprintf(w, "// %s is an opaque type.\n", goName)
		fmt.Fprintf(w, "type %s struct{}\n\n", goName)
		return nil
	}

	fmt.Fprintf(w, "type %s struct {\n", goName)
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
			// Strip pointer prefix for the check
			rest = strings.TrimPrefix(rest, "*")
			if len(rest) > 0 && rest[0] >= 'a' && rest[0] <= 'z' {
				goType = "unsafe.Pointer"
			}
		}

		fmt.Fprintf(w, "\t%s %s\n", fieldName, goType)
	}
	fmt.Fprintf(w, "}\n\n")
	return nil
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

func emitTypedefAliases(
	w io.Writer,
	framework *meta.FrameworkMeta,
	mapper *typemap.Mapper,
	imports typemap.ImportSet,
	takenGoNames map[string]bool,
) error {
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

	emitted := make(map[string]bool)
	for _, a := range aliases {
		goAliasName := naming.ExportedTypeName(a.name)
		if goAliasName == "" || emitted[goAliasName] || takenGoNames[goAliasName] {
			continue
		}
		emitted[goAliasName] = true

		ctx := typemap.Context{Framework: framework.Framework}
		if a.isPointer {
			targetType := mapper.GoType(a.target, ctx, imports)
			if targetType == "" || targetType == "unsafe.Pointer" ||
				isUnexportedCrossPackage(targetType) {
				continue
			}
			fmt.Fprintf(
				w,
				"// %s is an opaque pointer to %s (C typedef %s).\n",
				goAliasName,
				a.target,
				a.name,
			)
			fmt.Fprintf(w, "type %s = *%s\n\n", goAliasName, targetType)
		} else {
			targetType := mapper.GoType(a.target, ctx, imports)
			if targetType == "" || targetType == "unsafe.Pointer" ||
				isUnexportedCrossPackage(targetType) {
				continue
			}
			fmt.Fprintf(
				w,
				"// %s is an alias for %s (C typedef %s).\n",
				goAliasName,
				a.target,
				a.name,
			)
			fmt.Fprintf(w, "type %s = %s\n\n", goAliasName, targetType)
		}
	}
	return nil
}
