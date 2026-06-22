//go:build darwin

package idiomatic

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// importLines returns the import statements (each `"path"` or `alias "path"`)
// for the alias→path map, sorted by path then alias (the same path can be
// imported under two aliases, e.g. raw and foundation in the trial foundation
// package).
func importLines(imports map[string]string) []string {
	type imp struct{ alias, path string }
	list := make([]imp, 0, len(imports))
	for alias, path := range imports {
		list = append(list, imp{alias, path})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].path != list[j].path {
			return list[i].path < list[j].path
		}
		return list[i].alias < list[j].alias
	})
	lines := make([]string, 0, len(list))
	for _, i := range list {
		segs := strings.Split(i.path, "/")
		defaultAlias := segs[len(segs)-1]
		if i.alias == defaultAlias || i.alias == i.path {
			lines = append(lines, fmt.Sprintf("%q", i.path))
		} else {
			lines = append(lines, fmt.Sprintf("%s %q", i.alias, i.path))
		}
	}
	return lines
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
