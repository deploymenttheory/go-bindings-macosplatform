//go:build darwin

package idiomatic

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// importLines returns the import statements (each `"path"` or `alias "path"`)
// for the alias→path map, split into the two idiomatic goimports groups —
// standard-library packages first, then third-party/module packages — separated
// by a blank line. Within each group the entries are sorted by path then alias
// (the same path can be imported under two aliases, e.g. raw and foundation in
// the trial foundation package). The blank-line entry survives gofmt as the
// canonical import-group separator.
func importLines(imports map[string]string) []string {
	type imp struct{ alias, path string }
	var std, mod []imp
	for alias, path := range imports {
		if isStdlibImport(path) {
			std = append(std, imp{alias, path})
		} else {
			mod = append(mod, imp{alias, path})
		}
	}
	byPathThenAlias := func(group []imp) {
		sort.Slice(group, func(i, j int) bool {
			if group[i].path != group[j].path {
				return group[i].path < group[j].path
			}
			return group[i].alias < group[j].alias
		})
	}
	byPathThenAlias(std)
	byPathThenAlias(mod)

	render := func(i imp) string {
		segs := strings.Split(i.path, "/")
		defaultAlias := segs[len(segs)-1]
		if i.alias == defaultAlias || i.alias == i.path {
			return fmt.Sprintf("%q", i.path)
		}
		return fmt.Sprintf("%s %q", i.alias, i.path)
	}

	lines := make([]string, 0, len(std)+len(mod)+1)
	for _, i := range std {
		lines = append(lines, render(i))
	}
	if len(std) > 0 && len(mod) > 0 {
		lines = append(lines, "")
	}
	for _, i := range mod {
		lines = append(lines, render(i))
	}
	return lines
}

// isStdlibImport reports whether an import path names a standard-library package.
// Standard-library paths have no dot in their first segment (e.g. "context",
// "unsafe", "go/token"); module paths begin with a domain such as "github.com".
func isStdlibImport(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}

// fileView is the template data for file.tmpl.
type fileView struct {
	Header      string
	BuildTag    string
	PkgName     string
	ImportLines []string
	Body        string
}

// assembleFile renders a complete generated file — the scaffold (header, build
// tag, package clause, import block) plus the already-rendered body — through
// file.tmpl.
func assembleFile(pkgName string, imports map[string]string, body []byte) []byte {
	var buf bytes.Buffer
	renderTemplate(&buf, "file", fileView{
		Header:      strings.TrimRight(generatedHeader, "\n"),
		BuildTag:    strings.TrimRight(buildTag, "\n"),
		PkgName:     pkgName,
		ImportLines: importLines(imports),
		Body:        string(body),
	})
	return buf.Bytes()
}
