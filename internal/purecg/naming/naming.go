package naming

import (
	"strings"
	"unicode"
)

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

// MethodName converts an ObjC selector to an exported Go method name.
func MethodName(selector string) string {
	parts := strings.Split(selector, ":")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}

	var sb strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		sb.WriteString(capitalise(part))
	}

	name := sb.String()

	if strings.HasSuffix(name, "UsingBlock") {
		name = name[:len(name)-len("UsingBlock")] + "Using"
	}
	if strings.HasSuffix(name, "WithBlock") {
		name = name[:len(name)-len("WithBlock")] + "With"
	}

	return name
}

// ParamName sanitises an ObjC parameter name for use as a Go argument name.
func ParamName(objcName string) string {
	if objcName == "" {
		return "arg"
	}
	r := []rune(objcName)
	r[0] = unicode.ToLower(r[0])
	name := string(r)
	if goReservedWords[name] {
		name += "_"
	}
	return name
}

// PackageName converts a framework name to a Go package name (lowercase).
func PackageName(framework string) string {
	return strings.ToLower(framework)
}

// ExportedFunctionName maps a C function symbol to its exported Go name.
// Symbols that are already exported (first byte 'A'–'Z') are returned
// byte-identical so existing consumers of CFArrayCreate / SecItemAdd /
// CSSM_CL_CertSign never see a rename. Unexported symbols are converted to
// PascalCase: split on '_', drop empty segments, capitalise each segment's
// first letter, join.
//
//	vmnet_start_interface      → VmnetStartInterface
//	vImageBoxConvolve_ARGB8888 → VImageBoxConvolveARGB8888
//	_MPIsFullyInitialized      → MPIsFullyInitialized
//	CFArrayCreate              → CFArrayCreate (unchanged)
//
// Returns "" when no exported Go identifier can be derived.
func ExportedFunctionName(symbol string) string {
	if symbol == "" {
		return ""
	}
	if symbol[0] >= 'A' && symbol[0] <= 'Z' {
		return symbol
	}
	var sb strings.Builder
	for segment := range strings.SplitSeq(symbol, "_") {
		if segment == "" {
			continue
		}
		sb.WriteString(capitalise(segment))
	}
	name := sb.String()
	if name == "" || !unicode.IsUpper([]rune(name)[0]) {
		return ""
	}
	return name
}

// ExportedTypeName maps a C type name to an exported Go type name using the
// same rules as ExportedFunctionName (already-exported names byte-identical,
// snake_case → PascalCase).
func ExportedTypeName(name string) string {
	return ExportedFunctionName(name)
}

// GoTypeName ensures a type name is exported (first letter uppercase).
func GoTypeName(name string) string {
	if name == "" {
		return name
	}
	return capitalise(name)
}

// ProtocolGoTypeName returns the Go interface type name for an ObjC protocol.
func ProtocolGoTypeName(protoName string, classNameOwner map[string]string) string {
	bare := GoTypeName(protoName)
	if classNameOwner != nil {
		if _, isClass := classNameOwner[protoName]; isClass {
			return bare + "Protocol"
		}
	}
	return bare
}

// SelectorVarName returns a lowercase Go identifier for a selector variable
// that is safe to use as a package-level var name within a class file.
// e.g. ("NSString", "uppercaseString") → "_nsstrSelUppercaseString"
func SelectorVarName(className, selector string) string {
	prefix := "_" + strings.ToLower(className) + "Sel"
	return prefix + capitalise(MethodName(selector))
}

// ClassVarName returns the package-level var name for a class reference.
// e.g. "NSString" → "_clsNSString"
func ClassVarName(className string) string {
	return "_cls" + className
}

// capitalise returns s with its first Unicode letter uppercased.
func capitalise(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
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
