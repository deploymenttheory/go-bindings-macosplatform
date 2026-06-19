//go:build darwin

package idiomatic

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

// emitStructTypeAliases writes <pkgname>_type_aliases_generated.go: a Go type
// alias (type X = raw.X) for every value-type struct the framework defines (e.g.
// CGSize, CGRect, CGPoint), so callers can name them through the idiomatic
// package without importing the raw bindings. The alias *is* the raw type, so it
// still satisfies raw-typed by-value constructor/function parameters while
// keeping the caller's import on opinionated/idiomatic/...
//
// Only structs with fields are aliased: opaque (zero-field) structs are runtime
// handles of no use to idiomatic callers and can clash with C function names in
// the raw package. Runs after emitEnums so takenNames already covers every other
// package-level identifier emitted for this framework, preventing a redeclare.
func emitStructTypeAliases(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	fw *meta.FrameworkMeta,
	takenNames map[string]bool,
) error {
	names := make([]string, 0, len(fw.Structs))
	for name := range fw.Structs {
		names = append(names, name)
	}
	sort.Strings(names)

	var body bytes.Buffer
	seen := make(map[string]bool)
	for _, name := range names {
		s := fw.Structs[name]
		if s.Availability.IsUnavailable || len(s.Fields) == 0 {
			continue
		}
		goName := naming.ExportedTypeName(name)
		if goName == "" || seen[goName] || takenNames[goName] {
			continue
		}
		seen[goName] = true
		takenNames[goName] = true
		fmt.Fprintf(&body, "// %s is a type alias for the raw %s value-type struct.\n", goName, goName)
		fmt.Fprintf(&body, "type %s = %s.%s\n\n", goName, rawPkgAlias, goName)
	}
	if body.Len() == 0 {
		return nil
	}

	imports := map[string]string{rawPkgAlias: rawPkgPath}
	fname := pkgName + "_type_aliases_generated.go"
	file := assembleFile(pkgName, usedImports(body.Bytes(), imports), body.Bytes())
	return emit.WriteGoFile(filepath.Join(outDir, fname), file)
}
