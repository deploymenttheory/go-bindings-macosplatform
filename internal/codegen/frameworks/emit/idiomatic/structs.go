//go:build darwin

package idiomatic

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit"
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
func emitStructTypeAliases(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	fw *meta.FrameworkMeta,
	m *typemap.Mapper,
	takenNames map[string]bool,
) error {
	_ = rawPkgAlias
	_ = rawPkgPath
	ctx := typemap.Context{Framework: fw.Framework}

	type field struct{ name, goType string }
	// Resolve every available struct's field Go types from their Objective-C
	// types. A field that cannot be resolved drops the whole struct.
	resolved := make(map[string][]field)
	docOf := make(map[string]string)
	for name, s := range fw.Structs {
		if s.Availability.IsUnavailable || len(s.Fields) == 0 {
			continue
		}
		goName := naming.ExportedTypeName(name)
		if !isExportedGoIdent(goName) || resolved[goName] != nil {
			continue
		}
		fields := make([]field, 0, len(s.Fields))
		ok := true
		for _, f := range s.Fields {
			gt := m.GoType(f.ObjCType, ctx, make(typemap.ImportSet))
			if gt == "" {
				ok = false
				break
			}
			fields = append(fields, field{name: f.Name, goType: gt})
		}
		if ok && len(fields) > 0 {
			resolved[goName] = fields
			docOf[goName] = s.Doc
		}
	}

	// A struct is emittable when every field is a Go primitive or another
	// emittable struct in this package. Iterate to a fixpoint so a struct can
	// depend on one defined later (e.g. CGRect on CGPoint/CGSize).
	emittable := make(map[string]bool)
	for {
		changed := false
		for goName, fields := range resolved {
			if emittable[goName] {
				continue
			}
			good := true
			for _, f := range fields {
				if goPrimitives[f.goType] || emittable[f.goType] {
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

	names := make([]string, 0, len(resolved))
	for goName := range resolved {
		names = append(names, goName)
	}
	sort.Strings(names)

	var body bytes.Buffer
	for _, goName := range names {
		if !emittable[goName] || takenNames[goName] {
			continue
		}
		takenNames[goName] = true
		if doc := cleanDoc(docOf[goName]); doc != "" {
			fmt.Fprintf(&body, "// %s\n", strings.ReplaceAll(doc, "\n", "\n// "))
		}
		fmt.Fprintf(&body, "type %s struct {\n", goName)
		for _, f := range resolved[goName] {
			fmt.Fprintf(&body, "%s %s\n", capitalizeFirst(f.name), f.goType)
		}
		body.WriteString("}\n\n")
	}
	if body.Len() == 0 {
		return nil
	}

	fname := pkgName + "_type_aliases_generated.go"
	file := assembleFile(pkgName, nil, body.Bytes())
	return emit.WriteGoFile(filepath.Join(outDir, fname), file)
}
