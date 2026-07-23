// Package core holds the naming helpers that are identical across the frameworks
// (purego) and libraries (cgo) pipelines, so both pipeline-specific naming
// packages re-export them from a single source instead of keeping divergent
// copies.
//
// Only behavior-identical helpers live here. Functions that genuinely differ
// between the pipelines — ParamName (the frameworks pipeline camel-cases
// snake_case and lowers leading acronyms as a unit; the libraries pipeline only
// lowercases the first letter) and each pipeline's goReservedWords set — stay in
// the pipeline packages. Bridge-symbol helpers (libraries) and selector/class
// var-name helpers (frameworks) also stay pipeline-local.
package core

import (
	"strings"
	"unicode"
)

// Capitalise returns s with its first Unicode letter uppercased.
func Capitalise(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// MethodName converts an ObjC selector to an exported Go method name.
//
//	"objectAtIndex:"                        → "ObjectAtIndex"
//	"writeToURL:error:"                     → "WriteToURL"  (NSError** arg elided by scanner)
//	"enumerateObjectsUsingBlock:"           → "EnumerateObjectsUsing"
//	"count"                                 → "Count"
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
		sb.WriteString(Capitalise(part))
	}

	name := sb.String()

	// Drop trailing "Block" and replace with "Using"/"With" (idiomatic Go).
	if strings.HasSuffix(name, "UsingBlock") {
		name = name[:len(name)-len("UsingBlock")] + "Using"
	}
	if strings.HasSuffix(name, "WithBlock") {
		name = name[:len(name)-len("WithBlock")] + "With"
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
		sb.WriteString(Capitalise(segment))
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
	return Capitalise(name)
}

// ProtocolGoTypeName returns the Go interface type name for an ObjC protocol,
// disambiguating from a class of the same name by appending "Protocol" (Apple's
// NSObject exists as both a class and a protocol; the class takes the bare name).
// classNameOwner maps class names → owning framework; when the protocol's bare
// Go name appears there the suffix is applied, otherwise the bare name is used.
func ProtocolGoTypeName(protoName string, classNameOwner map[string]string) string {
	bare := GoTypeName(protoName)
	if classNameOwner != nil {
		if _, isClass := classNameOwner[protoName]; isClass {
			return bare + "Protocol"
		}
	}
	return bare
}
