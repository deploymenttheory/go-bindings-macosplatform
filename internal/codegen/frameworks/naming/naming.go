package naming

import (
	"strings"
	"unicode"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/naming/core"
)

// Shared naming helpers (identical across the purego/cgo pipelines) live in
// internal/codegen/naming/core and are re-exported here so existing callers keep
// using naming.MethodName / naming.ExportedTypeName / … unchanged.
var (
	MethodName           = core.MethodName
	PackageName          = core.PackageName
	GoTypeName           = core.GoTypeName
	ProtocolGoTypeName   = core.ProtocolGoTypeName
	ExportedFunctionName = core.ExportedFunctionName
	ExportedTypeName     = core.ExportedTypeName
)

// goReservedWords is the frameworks pipeline's escape set. It carries stdlib
// package names (strings, fmt, …) the purego generated files import, which the
// libraries set does not — so it stays pipeline-local.
var goReservedWords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true,
	"continue": true, "default": true, "defer": true, "else": true,
	"fallthrough": true, "for": true, "func": true, "go": true,
	"goto": true, "if": true, "import": true, "interface": true,
	"map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true,
	"var": true,
	"len": true, "cap": true, "copy": true, "delete": true,
	"make": true, "new": true, "panic": true, "print": true,
	"println": true, "error": true, "string": true, "close": true,
	"append":  true,
	"id":      true,
	"context": true,
	"unsafe":  true,
	"runtime": true,
	// Common stdlib package names that would shadow the import.
	"strings": true, "fmt": true, "errors": true, "sync": true,
	"bytes": true, "io": true, "os": true, "log": true,
	"math": true, "sort": true, "time": true, "net": true,
}

// ParamName sanitises an ObjC or C parameter name for use as a Go argument
// name: snake_case becomes camelCase (distributor_base_address →
// distributorBaseAddress) and a leading upper-case word is lowered as a unit
// (CPUCount → cpuCount, URLString → urlString, URLs → urls) rather than one
// letter at a time (never cPUCount). A name that would collide with a Go
// keyword, builtin, or common package name gets a trailing underscore.
func ParamName(objcName string) string {
	if objcName == "" {
		return "arg"
	}
	name := objcName
	if strings.Contains(name, "_") {
		name = snakeToCamel(name)
	}
	name = lowerLeadingWord(name)
	if name == "" {
		return "arg"
	}
	if goReservedWords[name] {
		name += "_"
	}
	return name
}

// snakeToCamel converts a snake_case parameter name to camelCase: segments
// after the first have their first letter capitalised; the first segment is
// left for lowerLeadingWord to case. An underscore-only name collapses to "".
func snakeToCamel(name string) string {
	var sb strings.Builder
	first := true
	for segment := range strings.SplitSeq(name, "_") {
		if segment == "" {
			continue
		}
		if first {
			sb.WriteString(segment)
			first = false
			continue
		}
		sb.WriteString(core.Capitalise(segment))
	}
	return sb.String()
}

// lowerLeadingWord lowers a name's leading upper-case word as a unit. The word
// is the leading run of capitals; when the run is followed by a lower-case
// letter its last capital starts the next word and stays upper (URLString →
// urlString, AVAsset → avAsset), except for a plural "s" directly after the
// run (URLs → urls, IDs → ids).
func lowerLeadingWord(name string) string {
	runes := []rune(name)
	if len(runes) == 0 || !unicode.IsUpper(runes[0]) {
		return name
	}
	run := 1
	for run < len(runes) && unicode.IsUpper(runes[run]) {
		run++
	}
	if run > 1 && run < len(runes) && unicode.IsLetter(runes[run]) {
		isPluralS := runes[run] == 's' &&
			(run+1 == len(runes) || !unicode.IsLower(runes[run+1]))
		if !isPluralS {
			run-- // the run's last capital starts the following word
		}
	}
	for i := range run {
		runes[i] = unicode.ToLower(runes[i])
	}
	return string(runes)
}

// SelectorVarName returns a lowercase Go identifier for a selector variable
// that is safe to use as a package-level var name within a class file.
// e.g. ("NSString", "uppercaseString") → "_nsstrSelUppercaseString"
func SelectorVarName(className, selector string) string {
	prefix := "_" + strings.ToLower(className) + "Sel"
	return prefix + core.Capitalise(MethodName(selector))
}

// ClassVarName returns the package-level var name for a class reference.
// e.g. "NSString" → "_clsNSString"
func ClassVarName(className string) string {
	return "_cls" + className
}

// lowerFirst returns s with its first Unicode letter lowercased.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// LowerFirst is the exported form of lowerFirst.
func LowerFirst(s string) string { return lowerFirst(s) }
