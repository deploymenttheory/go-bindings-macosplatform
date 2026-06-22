//go:build darwin

package idiomatic

import (
	"path/filepath"
	"sort"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/idiomatic/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/idiomatic/view"
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
	m *typemap.Mapper,
) (names, goTypes []string, ok bool) {
	for _, f := range s.Fields {
		gt := m.GoType(f.ObjCType, ctx, make(typemap.ImportSet))
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
// skipped.
func ComputeEmittableStructs(frameworks []*meta.FrameworkMeta, m *typemap.Mapper) map[string]bool {
	resolved := make(map[string][]string) // struct Go name → field Go types
	for _, fw := range frameworks {
		ctx := typemap.Context{Framework: fw.Framework}
		for name, s := range fw.Structs {
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
			if _, goTypes, ok := resolveStructFields(s, ctx, m); ok {
				resolved[goName] = goTypes
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
			good := true
			for _, gt := range goTypes {
				if goPrimitives[gt] || emittable[gt] {
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

func emitStructTypeAliases(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	fw *meta.FrameworkMeta,
	m *typemap.Mapper,
	takenNames map[string]bool,
) error {
	_ = rawPkgAlias
	_ = rawPkgPath
	ctx := typemap.Context{Framework: fw.Framework}

	type structDef struct {
		fieldNames, fieldTypes []string
		doc                    string
	}
	defs := make(map[string]structDef)
	for name, s := range fw.Structs {
		if s.Availability.IsUnavailable || len(s.Fields) == 0 {
			continue
		}
		goName := naming.ExportedTypeName(name)
		if !isExportedGoIdent(goName) {
			continue
		}
		if _, dup := defs[goName]; dup {
			continue
		}
		if names, goTypes, ok := resolveStructFields(s, ctx, m); ok {
			defs[goName] = structDef{fieldNames: names, fieldTypes: goTypes, doc: s.Doc}
		}
	}

	names := make([]string, 0, len(defs))
	for goName := range defs {
		names = append(names, goName)
	}
	sort.Strings(names)

	// Build the view, then render it through a template (render owns the Go
	// syntax; this function only resolves the data).
	var structs []view.Struct
	for _, goName := range names {
		// Emit only structs in the global emittable set, so this definition and
		// any cross-framework reference to it agree.
		if !m.EmittableStructs[goName] || takenNames[goName] {
			continue
		}
		takenNames[goName] = true
		def := defs[goName]
		fields := make([]view.Field, len(def.fieldNames))
		for i, fname := range def.fieldNames {
			fields[i] = view.Field{GoName: capitalizeFirst(fname), GoType: def.fieldTypes[i]}
		}
		structs = append(structs, view.Struct{
			GoName: goName,
			Doc:    cleanDoc(def.doc),
			Fields: fields,
		})
	}
	if len(structs) == 0 {
		return nil
	}

	body, err := render.Structs(structs)
	if err != nil {
		return err
	}
	fname := pkgName + "_type_aliases_generated.go"
	file := assembleFile(pkgName, nil, body)
	return emit.WriteGoFile(filepath.Join(outDir, fname), file)
}
