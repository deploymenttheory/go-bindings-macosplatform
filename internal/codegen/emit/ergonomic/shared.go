package ergonomic

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/classify"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// argInfo holds the name and Go type for a single method argument.
type argInfo struct {
	name   string
	goType string
}

// resolveOpinionatedArgType maps an ObjC arg type to a Go type, prepending
// "raw." for types owned by the current framework.
func resolveOpinionatedArgType(objcType string, ctx typemap.Context, m *typemap.Mapper, framework *meta.FrameworkMeta, imports typemap.ImportSet) string {
	goType := m.GoType(objcType, ctx, imports)
	if goType == "" {
		goType = "unsafe.Pointer"
	}
	return qualifyWithRawPackage(goType, framework)
}

// qualifyWithRawPackage prepends "raw." to unqualified type names that belong to the
// current framework, so they resolve correctly inside the opinionated package.
// Cross-framework types (already containing a ".") are returned unchanged.
func qualifyWithRawPackage(goType string, framework *meta.FrameworkMeta) string {
	// Handle "interface { P1; P2; ... }" compound protocol types.
	if strings.HasPrefix(goType, "interface {") {
		inner, _, ok := strings.Cut(goType[len("interface {"):], "}")
		if !ok {
			return goType
		}
		inner = strings.TrimSpace(inner)
		parts := strings.Split(inner, ";")
		var out []string
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			out = append(out, qualifyWithRawPackage(part, framework))
		}
		return "interface { " + strings.Join(out, "; ") + " }"
	}

	var prefixBuf strings.Builder
	rest := goType
	for {
		if strings.HasPrefix(rest, "*") {
			prefixBuf.WriteByte('*')
			rest = rest[1:]
		} else if strings.HasPrefix(rest, "[]") {
			prefixBuf.WriteString("[]")
			rest = rest[2:]
		} else {
			break
		}
	}
	baseName := rest
	if idx := strings.IndexByte(rest, '['); idx >= 0 {
		baseName = rest[:idx]
	}
	if strings.ContainsRune(baseName, '.') {
		return goType
	}
	if _, ok := framework.Classes[baseName]; ok {
		return prefixBuf.String() + "raw." + rest
	}
	if _, ok := framework.Protocols[baseName]; ok {
		return prefixBuf.String() + "raw." + rest
	}
	// Bugfix: protocol Go type names have a "Protocol" suffix that the metadata key doesn't have.
	if _, ok := framework.Protocols[strings.TrimSuffix(baseName, "Protocol")]; ok {
		return prefixBuf.String() + "raw." + rest
	}
	if _, ok := framework.Enums[baseName]; ok {
		return prefixBuf.String() + "raw." + rest
	}
	if _, ok := framework.Structs[baseName]; ok {
		return prefixBuf.String() + "raw." + rest
	}
	if _, ok := framework.Typedefs[baseName]; ok {
		return prefixBuf.String() + "raw." + rest
	}
	return goType
}

// recordOpinionatedImports finds cross-framework package alias → path pairs
// inside a Go type string and adds them to usedImports.
func recordOpinionatedImports(goType string, m *typemap.Mapper, usedImports map[string]string) {
	for _, part := range strings.FieldsFunc(goType, func(r rune) bool {
		return r == '*' || r == '[' || r == ']' || r == ',' || r == ' ' || r == '(' || r == ')'
	}) {
		dot := strings.IndexByte(part, '.')
		if dot < 0 {
			continue
		}
		pkg := part[:dot]
		if pkg == "raw" || pkg == "objc" || pkg == "unsafe" {
			continue
		}
		if m.ModulePrefix != "" {
			usedImports[pkg] = m.ModulePrefix + "/" + pkg
		}
	}
}

// sortedKeys returns the sorted keys of a map[string]meta.Class.
func sortedKeys(m map[string]meta.Class) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// buildRawReceiverType returns the Go raw type for a receiver parameter.
// Generic classes (e.g. NSArray[T]) are instantiated with objc.Object in package-level
// ergonomic wrappers, since Go does not allow type parameters on package-level functions
// unless the function itself is generic. Non-generic classes are returned as-is.
func buildRawReceiverType(className string, m *typemap.Mapper) string {
	if m.GenericClasses[className] {
		return "raw." + className + "[objc.Object]"
	}
	return "raw." + className
}

// recordSpecialImports inspects the generated source body and adds "objc" and/or
// "unsafe" to usedImports when those package names appear in type references.
// This is a post-generation safety net; ideally each emitter detects these upfront.
func recordSpecialImports(body []byte, usedImports map[string]string) {
	s := string(body)
	if strings.Contains(s, "objc.") {
		usedImports["objc"] = "github.com/deploymenttheory/go-bindings-macosplatform/internal/objc"
	}
	if strings.Contains(s, "unsafe.") {
		usedImports["unsafe"] = "unsafe"
	}
}

// writeErgonomicHeader emits the standard file header for an ergonomic *_generated.go file.
// It always imports "context" and the raw framework alias.
func writeErgonomicHeader(w io.Writer, pkgName, rawImportPath string, usedImports map[string]string, needsObjc bool) {
	fmt.Fprintf(w, "// Code generated by go-bindings-codegen. DO NOT EDIT.\n\n")
	fmt.Fprintf(w, "//go:build darwin\n\n")
	fmt.Fprintf(w, "package %s\n\n", pkgName)

	all := map[string]string{
		"context": "context",
		"raw":     rawImportPath,
	}
	if needsObjc {
		all["objc"] = "github.com/deploymenttheory/go-bindings-macosplatform/internal/objc"
	}
	for alias, path := range usedImports {
		if alias != "raw" {
			all[alias] = path
		}
	}

	type imp struct{ alias, path string }
	var imps []imp
	for alias, path := range all {
		imps = append(imps, imp{alias, path})
	}
	sort.Slice(imps, func(i, j int) bool { return imps[i].path < imps[j].path })

	fmt.Fprintf(w, "import (\n")
	for _, i := range imps {
		if strings.HasSuffix(i.path, "/"+i.alias) || i.alias == i.path {
			fmt.Fprintf(w, "\t%q\n", i.path)
		} else {
			fmt.Fprintf(w, "\t%s %q\n", i.alias, i.path)
		}
	}
	fmt.Fprintf(w, ")\n\n")
}

// containsPatternTag reports whether a PatternTag appears in a tag slice.
func containsPatternTag(tags []classify.PatternTag, want classify.PatternTag) bool {
	return slices.Contains(tags, want)
}

// extractFuncParams returns the comma-separated parameter type strings from a Go
// func type string like "func(*NSData, *NSURL, error)".
func extractFuncParams(goFuncType string) []string {
	_, after, ok := strings.Cut(goFuncType, "(")
	if !ok {
		return nil
	}
	end := strings.LastIndex(after, ")")
	if end < 0 {
		return nil
	}
	inner := strings.TrimSpace(after[:end])
	if inner == "" {
		return nil
	}
	return parseBlockParams(inner)
}

// parseBlockParams splits a comma-separated block param list, respecting nested <>.
func parseBlockParams(params string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, r := range params {
		switch r {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(params[start:i]))
				start = i + 1
			}
		}
	}
	if tail := strings.TrimSpace(params[start:]); tail != "" {
		parts = append(parts, tail)
	}
	return parts
}

// buildZeroValue returns a Go zero-value literal for the given type string.
func buildZeroValue(goType string) string {
	switch {
	case strings.HasPrefix(goType, "*"),
		strings.HasPrefix(goType, "[]"),
		strings.HasPrefix(goType, "func("),
		strings.HasPrefix(goType, "interface {"),
		goType == "error",
		goType == "unsafe.Pointer":
		return "nil"
	case strings.Contains(goType, ".") && !strings.HasPrefix(goType, "*"):
		// Package-qualified non-pointer types (e.g. objc.Object interface) → nil.
		return "nil"
	case goType == "bool":
		return "false"
	case goType == "string":
		return `""`
	default:
		return "0"
	}
}

// isObjcObjectType reports whether a Go type string represents a type that satisfies
// objc.Object (i.e., is a pointer type or is "objc.Object" itself).
// Non-ObjC types like func(), scalars, strings cannot be stored in a typed NSArray.
func isObjcObjectType(goType string) bool {
	t := strings.TrimSpace(goType)
	return strings.HasPrefix(t, "*") || t == "objc.Object"
}

// looksLikeNSArray reports whether an ObjC type is an NSArray or NSMutableArray.
// Block types that happen to reference NSArray in their parameter lists are excluded.
func looksLikeNSArray(objcType string) bool {
	if strings.Contains(objcType, "(^") {
		return false // block type, not a collection return
	}
	t := strings.TrimSpace(objcType)
	// Strip leading nullability annotations (e.g. "_Nullable NSArray<...> *")
	for _, ann := range []string{"_Nullable ", "_Nonnull ", "_Null_unspecified ", "__kindof "} {
		t = strings.TrimPrefix(t, ann)
	}
	return strings.HasPrefix(t, "NSArray") || strings.HasPrefix(t, "NSMutableArray")
}

// extractNSArrayElementGoType returns the Go type for the element of an
// NSArray<ClassName *> ObjC type string. Returns "" if not parsable.
func extractNSArrayElementGoType(objcType string, ctx typemap.Context, m *typemap.Mapper, framework *meta.FrameworkMeta, imports typemap.ImportSet) string {
	// Strip "NSMutableArray" or "NSArray" prefix and look for generic param.
	_, after, ok := strings.Cut(objcType, "<")
	if !ok {
		// Unparameterised NSArray — element type is objc.Object.
		return ""
	}
	end := strings.LastIndex(after, ">")
	if end < 0 {
		return ""
	}
	elemObjC := strings.TrimSpace(after[:end])
	goType := resolveOpinionatedArgType(elemObjC, ctx, m, framework, imports)
	if goType == "" || goType == "unsafe.Pointer" {
		return ""
	}
	return goType
}
