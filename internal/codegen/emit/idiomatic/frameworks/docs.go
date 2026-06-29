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

// docKind classifies the construct a doc comment is attached to, so docLeadKind
// can pick a verb that makes the first sentence read as proper Go documentation
// ("Name verb …"). Apple's header prose for a property is a bare noun phrase
// ("the URL of the kernel"), which is not a sentence on its own; the verb the
// kind implies ("returns"/"sets"/"reports whether") turns it into one.
type docKind int

const (
	// docMethod is an action method (or constructor); Apple's prose is normally
	// already verb-led ("Starts the VM…"), so no verb is injected.
	docMethod docKind = iota
	// docGetter is a value accessor; the prose is a noun phrase, so "returns" is
	// injected unless the prose already begins with its own verb.
	docGetter
	// docBoolGetter is a boolean accessor; the lead becomes "reports whether" and
	// any "true if …"/"indicates whether …" preamble in the prose is stripped.
	docBoolGetter
	// docSetter is a fluent With* setter; the lead becomes "sets".
	docSetter
)

// docLead renders a name-first doc comment block for an action method or
// constructor (docMethod): the exported symbol name followed by its cleaned
// Apple prose, lower-cased so the sentence reads naturally. It is the historical
// entry point; docLeadKind handles accessors and setters.
func docLead(goName, apple, fallback string) string {
	return docLeadKind(goName, apple, fallback, docMethod)
}

// docMethodFiller is the historical post-name clause used when a method has no
// Apple/header prose and no confident phrase can be synthesised from its name.
// It is retained as the safe fallback so a generated comment never regresses to
// something misleading.
const docMethodFiller = "wraps the corresponding Objective-C method."

// thirdPersonVerbs maps the lower-cased leading word of an action method's name
// to its third-person singular form, so a method with no documentation gets a
// readable sentence synthesised from its own name ("RemoveAllObjects" →
// "RemoveAllObjects removes all objects."). Only the well-known Objective-C
// method verbs are listed; an unrecognised lead word leaves the method on the
// safe filler (docMethodFiller) rather than risk an ungrammatical sentence. The
// table is a plain lookup — no regular expressions, no inflection rules.
var thirdPersonVerbs = map[string]string{
	"add": "adds", "append": "appends", "apply": "applies", "attach": "attaches",
	"begin": "begins", "bind": "binds", "cancel": "cancels", "clear": "clears",
	"close": "closes", "commit": "commits", "configure": "configures",
	"connect": "connects", "copy": "copies", "create": "creates",
	"deactivate": "deactivates", "decode": "decodes", "delete": "deletes",
	"deselect": "deselects", "detach": "detaches", "disable": "disables",
	"disconnect": "disconnects", "dismiss": "dismisses", "draw": "draws",
	"enable": "enables", "encode": "encodes", "end": "ends", "enumerate": "enumerates",
	"fetch": "fetches", "flush": "flushes", "hide": "hides", "insert": "inserts",
	"invalidate": "invalidates", "load": "loads", "lock": "locks", "make": "makes",
	"move": "moves", "open": "opens", "pause": "pauses", "perform": "performs",
	"pop": "pops", "post": "posts", "prepare": "prepares", "present": "presents",
	"push": "pushes", "read": "reads", "register": "registers", "reload": "reloads",
	"remove": "removes", "replace": "replaces", "request": "requests",
	"reset": "resets", "resume": "resumes", "run": "runs", "save": "saves",
	"scroll": "scrolls", "select": "selects", "send": "sends", "show": "shows",
	"start": "starts", "stop": "stops", "synchronize": "synchronizes",
	"toggle": "toggles", "unbind": "unbinds", "unlock": "unlocks",
	"unregister": "unregisters", "update": "updates", "validate": "validates",
	"write": "writes",
}

// recaseWords renders a slice of identifier words as a lower-cased,
// space-separated phrase, preserving the all-caps form of any word that names a
// known initialism ("USB", "URL"). It is the building block for synthesising a
// human-readable doc phrase from an exported Go name, and uses only the
// initialisms table — no regular expressions.
func recaseWords(words []string) string {
	out := make([]string, 0, len(words))
	for _, w := range words {
		if w == "" {
			continue
		}
		if up := strings.ToUpper(w); commonInitialisms[up] {
			out = append(out, up)
			continue
		}
		out = append(out, strings.ToLower(w))
	}
	return strings.Join(out, " ")
}

// synthFallback builds the post-name clause for a doc comment when a construct
// has no Apple/header prose, deriving a readable phrase from the exported Go name
// so the generated documentation says something rather than restating that the
// method exists (E2). docLeadKind prepends the name, so the returned text begins
// with a lower-case verb and ends with a period ("returns the title."). Synthesis
// uses splitWords + the initialisms table (and, for action methods, the
// thirdPersonVerbs lookup) — no regular expressions. When the name does not yield
// a confident phrase, the historical filler is returned so a comment never
// regresses to something misleading.
func synthFallback(goName string, kind docKind) string {
	words := splitWords(goName)
	switch kind {
	case docSetter:
		if len(words) > 0 && words[0] == "With" {
			words = words[1:]
		}
		phrase := recaseWords(words)
		if phrase == "" {
			return "sets the property."
		}
		return "sets the " + phrase + "."
	case docGetter:
		phrase := recaseWords(words)
		if phrase == "" {
			return docMethodFiller
		}
		return "returns the " + phrase + "."
	case docBoolGetter:
		// Only the Is*/Has* shapes synthesise cleanly into a "reports whether"
		// predicate; any other boolean accessor name would read ungrammatically,
		// so it stays on the safe filler.
		if len(words) > 1 && words[0] == "Is" {
			return "reports whether the object is " + recaseWords(words[1:]) + "."
		}
		if len(words) > 1 && words[0] == "Has" {
			return "reports whether the object has " + recaseWords(words[1:]) + "."
		}
		return docMethodFiller
	default: // docMethod
		if len(words) == 0 {
			return docMethodFiller
		}
		verb, ok := thirdPersonVerbs[strings.ToLower(words[0])]
		if !ok {
			return docMethodFiller
		}
		rest := recaseWords(words[1:])
		if rest == "" {
			return docMethodFiller
		}
		return verb + " " + rest + "."
	}
}

// docLeadKind renders a name-first doc comment block (E2) whose first sentence
// starts with the exported symbol name followed by a kind-appropriate verb and
// the cleaned Apple prose. When there is no Apple documentation it falls back to
// a short generated phrase (already verb-led by its caller). Verb selection is
// driven entirely by kind (resolved data), never by pattern-matching the prose,
// and uses only token/prefix string operations — no regular expressions.
func docLeadKind(goName, apple, fallback string, kind docKind) string {
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
	sb.WriteString("// ")
	sb.WriteString(strings.TrimRight(composeLead(goName, lines[0], kind), " "))
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

// composeLead builds the first doc-comment sentence "Name verb prose" for the
// given kind. prose is Apple's cleaned first line.
func composeLead(goName, prose string, kind docKind) string {
	switch kind {
	case docSetter:
		return goName + " sets " + lowerLead(prose)
	case docBoolGetter:
		return goName + " reports whether " + boolClause(prose)
	case docGetter:
		// When Apple's prose already opens with "return(s)", normalise it to the
		// third-person "returns" ("return the list of …" → "Name returns the list
		// of …").
		if w := firstWord(prose); w == "return" || w == "returns" {
			return goName + " returns " + lowerLead(stripFirstWord(prose))
		}
		// Apple sometimes phrases an accessor's prose with another leading verb
		// ("define the command-line parameters"); injecting "returns" there would
		// read as "returns define …", so the prose is left as-is in that case.
		if startsWithLeadingVerb(prose) {
			return goName + " " + lowerLead(prose)
		}
		return goName + " returns " + lowerLead(prose)
	default:
		return goName + " " + lowerLead(prose)
	}
}

// firstWord returns the first whitespace-delimited word of s, lower-cased and
// stripped of trailing punctuation, for set membership tests.
func firstWord(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(strings.TrimRight(fields[0], ".,;:"))
}

// stripFirstWord returns s with its first whitespace-delimited word removed and
// surrounding whitespace trimmed.
func stripFirstWord(s string) string {
	s = strings.TrimLeft(s, " \t")
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return strings.TrimSpace(s[i:])
	}
	return ""
}

// getterLeadingVerbs are verbs Apple uses to open an accessor's prose; when the
// prose already starts with one, docGetter does not prepend "returns".
var getterLeadingVerbs = map[string]bool{
	"define": true, "defines": true, "indicate": true, "indicates": true,
	"specify": true, "specifies": true, "return": true, "returns": true,
	"get": true, "gets": true, "determine": true, "determines": true,
	"contain": true, "contains": true, "represent": true, "represents": true,
	"describe": true, "describes": true, "hold": true, "holds": true,
	"provide": true, "provides": true, "create": true, "creates": true,
	"enable": true, "enables": true, "set": true, "sets": true,
}

func startsWithLeadingVerb(prose string) bool {
	return getterLeadingVerbs[firstWord(prose)]
}

// boolClause turns Apple's boolean-property prose into the predicate that follows
// "reports whether". It maps the Objective-C literals YES/NO to true/false, then
// strips a leading "true if …"/"indicates whether …"-style preamble and a
// trailing ", false otherwise"-style coda so the clause reads as a single
// condition. All matching is exact-token/prefix based, never a regular
// expression.
func boolClause(prose string) string {
	prose = replaceYesNo(prose)
	lower := strings.ToLower(prose)
	for _, lead := range []string{
		"a boolean value that indicates whether ", "a boolean value indicating whether ",
		"a value that indicates whether ", "a value indicating whether ",
		"return true if ", "returns true if ", "true if ",
		"return true when ", "returns true when ", "true when ",
		"indicates whether or not ", "indicate whether or not ",
		"indicates whether ", "indicate whether ", "indicating whether ",
		"represents whether ", "reflects whether ", "specifies whether ", "specifying whether ",
		"determines whether ", "describes whether ",
		"returns whether or not ", "return whether or not ",
		"returns whether ", "return whether ",
		"that indicates whether ", "that indicate whether ",
		"whether or not ", "whether ",
	} {
		if strings.HasPrefix(lower, lead) {
			prose = prose[len(lead):]
			break
		}
	}
	for _, coda := range []string{
		", false otherwise.", ", false otherwise", ", otherwise false.", ", otherwise false",
		", no otherwise.", ", no otherwise",
	} {
		if strings.HasSuffix(strings.ToLower(prose), coda) {
			prose = prose[:len(prose)-len(coda)]
			if !strings.HasSuffix(prose, ".") {
				prose += "."
			}
			break
		}
	}
	return lowerLead(strings.TrimSpace(prose))
}

// replaceYesNo replaces the standalone Objective-C boolean literals YES and NO
// with Go's true and false, leaving words that merely contain those letters
// untouched (it operates on whitespace-split tokens, not substrings).
func replaceYesNo(s string) string {
	fields := strings.Fields(s)
	for i, f := range fields {
		trail := ""
		core := f
		for len(core) > 0 && strings.ContainsRune(".,;:)", rune(core[len(core)-1])) {
			trail = string(core[len(core)-1]) + trail
			core = core[:len(core)-1]
		}
		switch core {
		case "YES":
			fields[i] = "true" + trail
		case "NO":
			fields[i] = "false" + trail
		}
	}
	return strings.Join(fields, " ")
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
	for name, class := range fc.framework.Classes {
		if class.Availability.IsUnavailable || class.Super != className {
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
	for className, class := range fc.framework.Classes {
		if class.Availability.IsUnavailable || class.Super == "" {
			continue
		}
		if _, isBase := abstractBases[class.Super]; isBase {
			childrenByBase[class.Super] = append(
				childrenByBase[class.Super],
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
		Framework: fc.framework.Framework,
		Groups:    buildDocGroups(fc, abstractBases, prefix),
	})
	return emit.WriteGoFile(filepath.Join(outDir, "doc.go"), buf.Bytes())
}
