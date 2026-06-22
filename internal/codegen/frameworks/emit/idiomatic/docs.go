//go:build darwin

package idiomatic

import (
	"bytes"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit"
)

// cleanDoc rewrites Apple's HeaderDoc/Doxygen documentation into plain prose.
// Apple's headers mark up doc comments with tags like "@abstract", "@discussion",
// "@param", "@return" and "@see"; those are noise in a Go doc comment. The
// summary and discussion text is kept (the tag word is dropped); parameter,
// return, see-also, and similar reference tags are removed entirely.
func cleanDoc(doc string) string {
	doc = strings.TrimSpace(doc)
	if !strings.Contains(doc, "@") {
		return doc
	}
	// Drop the doc-comment delimiters Apple sometimes leaves in (/*! ... */).
	doc = strings.ReplaceAll(doc, "/*!", "")
	doc = strings.ReplaceAll(doc, "*/", "")

	segments := strings.Split(doc, "@")
	out := make([]string, 0, len(segments))
	if lead := strings.TrimSpace(segments[0]); lead != "" {
		out = append(out, lead)
	}
	for _, seg := range segments[1:] {
		seg = strings.TrimLeft(seg, " \t\n")
		tag, rest := seg, ""
		if i := strings.IndexAny(seg, " \t\n"); i >= 0 {
			tag, rest = seg[:i], strings.TrimSpace(seg[i:])
		}
		switch tag {
		case "abstract", "discussion", "brief", "details", "summary":
			// Keep the prose, drop the tag word.
			if rest != "" {
				out = append(out, rest)
			}
		default:
			// @param, @return, @returns, @see, @note, @warning, @c, @link,
			// @constant, @field, @function, … — drop the whole reference clause.
		}
	}
	return strings.TrimSpace(strings.Join(out, " "))
}

// commentBlock renders documentation prose as a Go // comment block, one
// "// "-prefixed line per source line, with blank lines preserved as "//".
// Returns "" for empty docs. The result ends in a newline when non-empty, so a
// template can follow it with a "//" separator before the generated summary.
// docLead renders a name-first doc comment block (E2): the exported symbol name
// followed by its cleaned Apple prose, with the prose's first letter lower-cased
// so the sentence reads naturally ("Count" + "returns the count." → "Count
// returns the count."). When there is no Apple documentation it falls back to a
// short generated phrase. An acronym-leading word (URL, NSData) is left as-is.
func docLead(goName, apple, fallback string) string {
	apple = cleanDoc(strings.TrimSpace(apple))
	var sb strings.Builder
	if apple == "" {
		sb.WriteString("// ")
		sb.WriteString(goName)
		sb.WriteString(" ")
		sb.WriteString(fallback)
		sb.WriteString("\n")
		return sb.String()
	}
	lines := strings.Split(apple, "\n")
	lines[0] = lowerLead(lines[0])
	sb.WriteString("// ")
	sb.WriteString(goName)
	sb.WriteString(" ")
	sb.WriteString(strings.TrimRight(lines[0], " "))
	sb.WriteString("\n")
	for _, l := range lines[1:] {
		if strings.TrimSpace(l) == "" {
			sb.WriteString("//\n")
			continue
		}
		sb.WriteString("// ")
		sb.WriteString(strings.TrimRight(l, " "))
		sb.WriteString("\n")
	}
	return sb.String()
}

// lowerLead lower-cases the first letter of s unless its first word is an acronym
// (two or more leading upper-case letters, e.g. "URL", "NSData"), where lowering
// would misread.
func lowerLead(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] < 'A' || r[0] > 'Z' {
		return s
	}
	if len(r) > 1 && r[1] >= 'A' && r[1] <= 'Z' {
		return s // acronym
	}
	r[0] = r[0] - 'A' + 'a'
	return string(r)
}

// buildClassDoc assembles the full godoc comment block for a wrapper type: the
// name-leading summary (E2), a Usage paragraph (an abstract base points at its
// concrete subtypes; a subclass notes the base it embeds), then the cleaned Apple
// prose. It returns a "// …\n"-prefixed block ending in a newline, so the type
// declaration follows immediately with no detaching blank line.
func buildClassDoc(
	goTypeName, className string,
	isAbstract bool,
	subLinks, baseType, appleDoc string,
) string {
	lines := []string{
		goTypeName + " is an idiomatic wrapper over the Objective-C class " + className + ".",
	}
	switch {
	case isAbstract && subLinks != "":
		lines = append(
			lines,
			"",
			goTypeName+" is an abstract base — you do not construct it directly. Construct one of "+subLinks+" and pass it where a "+goTypeName+" is accepted.",
		)
	case isAbstract:
		lines = append(
			lines,
			"",
			goTypeName+" is an abstract base — you do not construct it directly. Construct a concrete subtype and pass it where a "+goTypeName+" is accepted.",
		)
	case baseType != "":
		lines = append(lines, "", "It embeds ["+baseType+"], promoting that type's methods.")
	}
	if ad := cleanDoc(strings.TrimSpace(appleDoc)); ad != "" {
		lines = append(lines, "")
		lines = append(lines, strings.Split(ad, "\n")...)
	}
	var sb strings.Builder
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			sb.WriteString("//\n")
			continue
		}
		sb.WriteString("// ")
		sb.WriteString(strings.TrimRight(l, " "))
		sb.WriteString("\n")
	}
	return sb.String()
}

// directSubclassLinks returns the godoc links ("[A], [B]") for a class's direct,
// available, same-framework concrete subclasses, sorted — used in an abstract
// base's documentation to point the reader at what to construct.
func directSubclassLinks(className string, fc *frameworkContext, prefix string) string {
	var subs []string
	for name, cls := range fc.fw.Classes {
		if cls.Availability.IsUnavailable || cls.Super != className {
			continue
		}
		subs = append(subs, "["+trialTypeName(name, prefix)+"]")
	}
	sort.Strings(subs)
	return strings.Join(subs, ", ")
}

// docGroup is one line of the package doc.go type index: a provider base and the
// godoc-linked concrete subtypes a caller chooses from.
type docGroup struct {
	Title string // the base wrapper type name
	Links string // "[SubA], [SubB], …" — godoc links to the concrete subtypes
}

// docFileView is the template data for docfile.tmpl.
type docFileView struct {
	Header, BuildTag, PkgName, Framework string
	Groups                               []docGroup
}

// buildDocGroups assembles the package type index: one entry per provider base,
// listing its direct concrete subclasses as godoc [Type] links, so the package
// doc reads as a map of "which type do I build for this role".
func buildDocGroups(
	fc *frameworkContext,
	abstractBases abstractBaseIndex,
	prefix string,
) []docGroup {
	childrenByBase := map[string][]string{} // base raw name → subclass Go names
	for className, cls := range fc.fw.Classes {
		if cls.Availability.IsUnavailable || cls.Super == "" {
			continue
		}
		if _, isBase := abstractBases[cls.Super]; isBase {
			childrenByBase[cls.Super] = append(
				childrenByBase[cls.Super],
				trialTypeName(className, prefix),
			)
		}
	}
	baseRaws := make([]string, 0, len(abstractBases))
	for raw := range abstractBases {
		baseRaws = append(baseRaws, raw)
	}
	sort.Slice(
		baseRaws,
		func(i, j int) bool { return abstractBases[baseRaws[i]] < abstractBases[baseRaws[j]] },
	)

	var groups []docGroup
	for _, raw := range baseRaws {
		subs := childrenByBase[raw]
		if len(subs) == 0 {
			continue
		}
		sort.Strings(subs)
		links := make([]string, len(subs))
		for i, s := range subs {
			links[i] = "[" + s + "]"
		}
		groups = append(
			groups,
			docGroup{Title: abstractBases[raw], Links: strings.Join(links, ", ")},
		)
	}
	return groups
}

func emitDocGo(
	outDir, pkgName string,
	fc *frameworkContext,
	abstractBases abstractBaseIndex,
	prefix string,
) error {
	var buf bytes.Buffer
	renderTemplate(&buf, "docfile", docFileView{
		Header:    strings.TrimRight(generatedHeader, "\n"),
		BuildTag:  strings.TrimRight(buildTag, "\n"),
		PkgName:   pkgName,
		Framework: fc.fw.Framework,
		Groups:    buildDocGroups(fc, abstractBases, prefix),
	})
	return emit.WriteGoFile(filepath.Join(outDir, "doc.go"), buf.Bytes())
}
