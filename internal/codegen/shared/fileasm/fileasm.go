// Package fileasm assembles a complete generated Go source file — the
// DO-NOT-EDIT header, the build constraint, the package clause, the import
// block, and the rendered body — through a single template. It is shared by
// every emitter so file scaffolding is produced one way, in one place.
//
// The assembler is deliberately parameterised rather than opinionated: the
// caller supplies the exact header text, build tag, and the already-grouped
// import lines. This lets each emitter keep its own byte-for-byte output (the
// raw purego files, the CGo library files, and the idiomatic files use
// different header strings and import groupings) while still flowing through one
// template. gofmt canonicalises whitespace afterwards, so only tokens matter.
//
// SCOPE — shared: used by every emitter (raw and idiomatic, frameworks and
// libraries) so file scaffolding is produced one way, in one place.
package fileasm

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"
)

//go:embed templates/file.tmpl
var templatesFS embed.FS

var templates = template.Must(template.ParseFS(templatesFS, "templates/file.tmpl"))

// File is the data for the file scaffold template.
type File struct {
	// Header is the DO-NOT-EDIT comment block (no trailing newline).
	Header string
	// BuildTag is the build constraint line, e.g. "//go:build darwin" (no
	// trailing newline); empty to omit.
	BuildTag string
	// PkgName is the Go package name.
	PkgName string
	// ImportLines are the rendered import entries (each `"path"` or
	// `alias "path"`, with blank-line group separators already inserted). Empty
	// to omit the import block entirely.
	ImportLines []string
	// Body is the already-rendered file body (declarations).
	Body string
}

// Assemble renders a complete generated file from f. The result is pre-gofmt;
// the caller is expected to run it through go/format.
func Assemble(f File) []byte {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "file", f); err != nil {
		// Template execution only fails on a generator bug (bad data/template),
		// so failing loudly is correct.
		panic(fmt.Sprintf("fileasm: assembling file for package %q: %v", f.PkgName, err))
	}
	return buf.Bytes()
}

// ImportLinesStdlibMod renders import entries for the alias→path map, split into
// the two goimports groups goimports itself produces — standard-library packages
// first, then everything else (third-party and same-module) — separated by a
// blank line. Within each group entries are sorted by path then alias (the same
// path can be imported under two aliases). The blank-line entry survives gofmt as
// the canonical group separator.
func ImportLinesStdlibMod(imports map[string]string) []string {
	var stdlib, other []importEntry
	for alias, path := range imports {
		if IsStdlibImport(path) {
			stdlib = append(stdlib, importEntry{alias, path})
		} else {
			other = append(other, importEntry{alias, path})
		}
	}
	sortByPathThenAlias(stdlib)
	sortByPathThenAlias(other)

	lines := make([]string, 0, len(stdlib)+len(other)+1)
	for _, e := range stdlib {
		lines = append(lines, renderImport(e.alias, e.path))
	}
	if len(stdlib) > 0 && len(other) > 0 {
		lines = append(lines, "")
	}
	for _, e := range other {
		lines = append(lines, renderImport(e.alias, e.path))
	}
	return lines
}

// importEntry is one import: its alias and path.
type importEntry struct{ alias, path string }

func sortByPathThenAlias(group []importEntry) {
	sort.Slice(group, func(i, j int) bool {
		if group[i].path != group[j].path {
			return group[i].path < group[j].path
		}
		return group[i].alias < group[j].alias
	})
}

// renderImport renders one import entry, omitting the alias when it is redundant
// (equal to the path's last segment or to the path itself).
func renderImport(alias, path string) string {
	segments := strings.Split(path, "/")
	defaultAlias := segments[len(segments)-1]
	if alias == defaultAlias || alias == path {
		return fmt.Sprintf("%q", path)
	}
	return fmt.Sprintf("%s %q", alias, path)
}

// IsStdlibImport reports whether an import path names a standard-library package.
// Standard-library paths have no dot in their first segment (e.g. "context",
// "unsafe", "go/token"); module paths begin with a domain such as "github.com".
func IsStdlibImport(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}

// ImportLinesStdlibExternalInternal renders import entries for the alias→path
// map in three groups — standard library, third-party external, then this
// module's own (github.com/deploymenttheory/…) packages — each separated by a
// blank line, paths sorted within a group. An alias is rendered only when it
// differs from the path's last segment. This is the grouping the raw emitters
// use (distinct from the two-group goimports grouping).
func ImportLinesStdlibExternalInternal(imports map[string]string) []string {
	const moduleRoot = "github.com/deploymenttheory"
	var stdlib, external, internal []string
	pathAlias := map[string]string{}
	for alias, path := range imports {
		switch {
		case !strings.Contains(path, "."):
			stdlib = append(stdlib, path)
		case strings.HasPrefix(path, moduleRoot):
			internal = append(internal, path)
		default:
			external = append(external, path)
		}
		segments := strings.Split(path, "/")
		if alias != segments[len(segments)-1] {
			pathAlias[path] = alias
		}
	}
	sort.Strings(stdlib)
	sort.Strings(external)
	sort.Strings(internal)

	render := func(path string) string {
		if alias, ok := pathAlias[path]; ok {
			return fmt.Sprintf("%s %q", alias, path)
		}
		return fmt.Sprintf("%q", path)
	}

	lines := make([]string, 0, len(stdlib)+len(external)+len(internal)+2)
	for _, path := range stdlib {
		lines = append(lines, render(path))
	}
	if len(stdlib) > 0 && len(external)+len(internal) > 0 {
		lines = append(lines, "")
	}
	for _, path := range external {
		lines = append(lines, render(path))
	}
	if len(external) > 0 && len(internal) > 0 {
		lines = append(lines, "")
	}
	for _, path := range internal {
		lines = append(lines, render(path))
	}
	return lines
}
