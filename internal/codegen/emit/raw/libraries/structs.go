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
)

// Structs writes all C struct type definitions and struct typedef aliases to w.
// It returns a map of Go package alias → import path for any cross-framework
// imports required by struct fields (e.g. corefoundation for CF-typed fields).
func EmitStructs(w io.Writer, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) (typemap.ImportSet, error) {
	usedImports := make(typemap.ImportSet)
	cfOpaqueStructs := buildCFOpaqueSet(framework)

	// ── Regular structs ─────────────────────────────────────────────────────────
	for _, name := range sortedStringKeys(framework.Structs) {
		if cfOpaqueStructs[name] {
			continue
		}
		model := buildStructModel(name, framework.Structs[name], framework.Framework, m, knownClasses, usedImports)
		if err := render.Execute(w, "struct", model); err != nil {
			return nil, err
		}
	}

	// ── CF opaque wrapper types ─────────────────────────────────────────────────
	cfOrdered := computeCFOrdered(framework.Typedefs, cfOpaqueStructs)
	for _, structName := range sortedStringKeys(cfOrdered) {
		typedefNames := cfOrdered[structName]
		primaryName := typedefNames[0]
		goPrimary := naming.GoTypeName(primaryName)

		if err := render.Execute(w, "cf_wrapper", view.CfWrapperModel{GoName: goPrimary}); err != nil {
			return nil, err
		}
		for _, alias := range typedefNames[1:] {
			model := view.CfAliasModel{GoAlias: naming.GoTypeName(alias), GoPrimary: goPrimary}
			if err := render.Execute(w, "cf_alias", model); err != nil {
				return nil, err
			}
		}
	}

	// ── Non-CF struct typedef aliases ───────────────────────────────────────────
	for _, name := range sortedStringKeys(framework.Typedefs) {
		target := framework.Typedefs[name]
		if cfStructName := cfPointerStructName(target); cfStructName != "" && cfOpaqueStructs[cfStructName] {
			continue
		}
		if !strings.HasPrefix(target, "struct ") {
			continue
		}
		structName := strings.TrimSpace(strings.TrimPrefix(target, "struct "))
		if _, ok := framework.Structs[structName]; !ok {
			continue
		}
		if cfOpaqueStructs[structName] {
			continue
		}
		goAlias := naming.GoTypeName(name)
		goTarget := naming.GoTypeName(structName)
		if goAlias == goTarget {
			continue
		}
		if err := render.Execute(w, "struct_typedef", view.StructTypedefModel{GoAlias: goAlias, GoTarget: goTarget}); err != nil {
			return nil, err
		}
	}

	return usedImports, nil
}

func buildStructModel(name string, s macosplatformmetadata.Struct, framework string, m *typemap.Mapper, knownClasses map[string]bool, usedImports typemap.ImportSet) view.StructModel {
	ctx := m.BaseContext(framework, knownClasses)

	fields := make([]view.StructFieldModel, 0, len(s.Fields))
	for i, f := range s.Fields {
		fieldName := f.Name
		if fieldName == "" {
			fieldName = fmt.Sprintf("Field%d", i)
		} else {
			fieldName = naming.GoTypeName(f.Name)
		}
		goType := f.GoType
		if goType == "" {
			goType = m.GoType(f.ObjCType, ctx, usedImports)
		}
		if goType == "" {
			goType = "unsafe.Pointer"
		}
		fields = append(fields, view.StructFieldModel{Name: fieldName, GoType: goType})
	}
	return view.StructModel{
		GoName:       naming.GoTypeName(name),
		CommentBlock: renderCommentBlock(s.Doc, s.SDKFile, s.SDKLine, s.Availability, ""),
		Fields:       fields,
	}
}

// sortedStringKeys returns the keys of a string-keyed map in sorted order.
func sortedStringKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// buildCFOpaqueSet returns the set of zero-field struct names that are emitted
// as opaque pointer wrappers via their pointer typedefs rather than as plain structs.
func buildCFOpaqueSet(framework *macosplatformmetadata.FrameworkMeta) map[string]bool {
	zeroFieldStructs := make(map[string]bool)
	for name, s := range framework.Structs {
		if len(s.Fields) == 0 {
			zeroFieldStructs[name] = true
		}
	}
	referencedByTypedef := make(map[string]bool)
	for _, target := range framework.Typedefs {
		if sname := opaquePointerStructName(target); sname != "" && zeroFieldStructs[sname] {
			referencedByTypedef[sname] = true
		}
	}
	return referencedByTypedef
}

// computeCFOrdered builds a map from opaque struct name to an ordered list of
// typedef names: the primary (immutable, non-Mutable) type is first, aliases follow.
func computeCFOrdered(typedefs map[string]string, cfOpaqueStructs map[string]bool) map[string][]string {
	grouped := make(map[string][]string)
	for tdName, target := range typedefs {
		structName := cfPointerStructName(target)
		if structName == "" || !cfOpaqueStructs[structName] {
			continue
		}
		grouped[structName] = append(grouped[structName], tdName)
	}
	for structName := range grouped {
		names := grouped[structName]
		sort.Strings(names)
		for i, name := range names {
			if !strings.Contains(name, "Mutable") {
				names[0], names[i] = names[i], names[0]
				break
			}
		}
		grouped[structName] = names
	}
	return grouped
}

// opaquePointerStructName extracts the struct name from a pointer typedef target
// like "const struct __CFString *" → "__CFString", "struct CGColor *" → "CGColor".
// Returns "" if the target is not a simple pointer-to-struct form.
func opaquePointerStructName(target string) string {
	s := strings.TrimSpace(target)
	s = strings.TrimPrefix(s, "const ")
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "struct ") {
		return ""
	}
	s = strings.TrimPrefix(s, "struct ")
	if !strings.HasSuffix(s, " *") {
		return ""
	}
	s = strings.TrimSuffix(s, " *")
	s = strings.TrimSpace(s)
	if strings.ContainsAny(s, " *<>^()") {
		return ""
	}
	return s
}

// cfPointerStructName is an alias for opaquePointerStructName retained for
// backward compatibility with callers that reference it by the old name.
func cfPointerStructName(target string) string {
	return opaquePointerStructName(target)
}
