package naming

import (
	"strings"
	"unicode"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/naming/core"
)

// Shared naming helpers (identical across the purego/cgo pipelines) live in
// internal/codegen/naming/core and are re-exported here. This is also where the
// libraries pipeline gains ExportedFunctionName / ExportedTypeName (previously
// frameworks-only), so library type names can be unified onto the PascalCase
// ExportedTypeName form (audit_token_t → AuditTokenT) instead of the
// underscore-preserving GoTypeName.
var (
	MethodName           = core.MethodName
	PackageName          = core.PackageName
	GoTypeName           = core.GoTypeName
	ProtocolGoTypeName   = core.ProtocolGoTypeName
	ExportedFunctionName = core.ExportedFunctionName
	ExportedTypeName     = core.ExportedTypeName
)

// goReservedWords is the libraries pipeline's escape set. Unlike the frameworks
// set it carries `inv` and `ctx` (shadowed by generated cgo method bodies) but
// not the stdlib package names, so it stays pipeline-local.
var goReservedWords = map[string]bool{
	// keywords
	"break": true, "case": true, "chan": true, "const": true,
	"continue": true, "default": true, "defer": true, "else": true,
	"fallthrough": true, "for": true, "func": true, "go": true,
	"goto": true, "if": true, "import": true, "interface": true,
	"map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true,
	"var": true,
	// common builtins that cause confusion when shadowed
	"len": true, "cap": true, "copy": true, "delete": true,
	"make": true, "new": true, "panic": true, "print": true,
	"println": true, "error": true, "string": true, "close": true,
	"append": true,
	// ObjC keyword that clashes with the 'id' dynamic-object type in C bridge .m files
	"id": true,
	// Go stdlib package names commonly imported by generated files. An ObjC
	// param named e.g. "context" (KVO observers, addObserver:context:) would
	// otherwise shadow the `context` package and break references to
	// context.Context in the same scope.
	"context": true,
	"unsafe":  true,
	"runtime": true,
	// Generated method bodies use `inv runtime.Invocation` as a closure
	// parameter. An ObjC selector with an argument named "inv" (e.g.
	// -[NSInvocationOperation initWithInvocation:]) would shadow it and
	// cause inv.Ptr() to resolve against runtime.Invocation, not the user's
	// NSInvocation argument.
	"inv": true,
	// Every emitted Go method starts with `ctx context.Context`. An ObjC
	// param named "ctx" (e.g. -[CAOpenGLLayer drawInCGLContext:..ctx..]
	// or -[AVCaptionRenderer renderInContext:..ctx..]) would otherwise
	// redeclare ctx and shadow the context.Context value.
	"ctx": true,
}

// ParamName sanitises an ObjC parameter name for use as a Go argument name.
// It lowercases the first letter and escapes Go reserved words.
func ParamName(objcName string) string {
	if objcName == "" {
		return "arg"
	}
	// Lowercase the first letter
	r := []rune(objcName)
	r[0] = unicode.ToLower(r[0])
	name := string(r)

	if goReservedWords[name] {
		name += "_"
	}
	return name
}

// BridgeFuncName produces the C bridge function name for a given class+selector.
// Class methods get a "_cls" suffix to avoid conflicts with identically-named
// instance methods (e.g. +[NSDate timeIntervalSinceReferenceDate] vs
// -[NSDate timeIntervalSinceReferenceDate]).
//
//	("Foundation", "NSArray", "objectAtIndex:", false)              → "foundation_NSArray_objectAtIndex"
//	("Foundation", "NSBundle", "URLForAuxiliaryExecutable:", false) → "foundation_NSBundle_urlForAuxiliaryExecutable"
//	("Foundation", "NSDate", "timeIntervalSinceReferenceDate", true) → "foundation_NSDate_timeIntervalSinceReferenceDate_cls"
func BridgeFuncName(framework, className, selector string, isClassMethod bool) string {
	method := lowerLeadingAcronym(MethodName(selector))
	name := strings.ToLower(framework) + "_" + className + "_" + method
	if isClassMethod {
		name += "_cls"
	}
	return name
}

// lowerLeadingAcronym lowercases the leading acronym in a PascalCase identifier,
// producing a camelCase C name. It handles the three patterns that arise from
// ObjC method names:
//
//   - Single leading uppercase:   "ObjectAtIndex" → "objectAtIndex"
//   - Leading acronym + word:     "URLForResource" → "urlForResource"
//   - Leading acronym + suffix:   "URLsForResources" → "urlsForResources"
//   - Leading acronym only:       "URL" → "url"
//
// The distinction between "acronym + word" and "acronym + suffix" is made by
// checking whether the lowercase letter that ends the uppercase run is itself
// followed by an uppercase letter. If so (e.g. URLs·For) the lowercase letter
// is a single-character suffix and the entire uppercase run is lowercased.
// If not (e.g. URLF·or) the last uppercase letter starts the next word and
// is preserved, so only the preceding uppercase letters are lowercased.
func lowerLeadingAcronym(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)

	// Find the length of the leading all-uppercase run.
	end := 0
	for end < len(runes) && unicode.IsUpper(runes[end]) {
		end++
	}

	if end == 0 {
		return s // already starts with lowercase
	}

	// How many characters to lowercase.
	stop := end
	if end < len(runes) && end > 1 {
		// There is a lowercase character at position end. Check whether it is
		// a single-character suffix (followed immediately by uppercase, e.g. the
		// 's' in "URLsFor") or the genuine start of the next word (e.g. the 'o'
		// in "urlFor" — nothing uppercase follows immediately).
		nextIsUpper := end+1 < len(runes) && unicode.IsUpper(runes[end+1])
		if !nextIsUpper {
			// The last uppercase char (at end-1) starts the next camelCase word;
			// leave it uppercase. E.g. "URLFor" → stop=3, giving "urlFor".
			stop = end - 1
		}
		// If nextIsUpper: the char at end is a suffix (e.g. 's' in "URLs");
		// lowercase the entire run. stop stays at end.
	}

	out := make([]rune, len(runes))
	copy(out, runes)
	for i := 0; i < stop; i++ {
		out[i] = unicode.ToLower(out[i])
	}
	return string(out)
}

// BlockTypeName produces a named Go type for an ObjC block used in a framework.
// Example: ("Foundation", "NSArray", "enumerateObjectsUsingBlock:") → "NSArrayEnumerationBlock"
func BlockTypeName(className, selector string) string {
	method := MethodName(selector)
	// Strip common suffixes to get the concept name
	method = strings.TrimSuffix(method, "Using")
	method = strings.TrimSuffix(method, "With")
	method = strings.TrimSuffix(method, "Block")
	return className + method + "Block"
}

// MethodBridgeID returns the structured-path ID comment for an ObjC method binding.
//
//	("Foundation", "NSArray", "objectAtIndex:", false) → "ID: objc-sym Foundation.NSArray.-[objectAtIndex:]"
//	("Foundation", "NSDate", "date", true)             → "ID: objc-sym Foundation.NSDate.+[date]"
func MethodBridgeID(framework, class, selector string, isClassMethod bool) string {
	sign := "-"
	if isClassMethod {
		sign = "+"
	}
	return "ID: objc-sym " + framework + "." + class + "." + sign + "[" + selector + "]"
}

// FunctionBridgeID returns the structured-path ID comment for a free C function binding.
//
//	("CoreFoundation", "CFArrayCreateMutable") → "ID: objc-sym CoreFoundation.CFArrayCreateMutable"
func FunctionBridgeID(framework, funcName string) string {
	return "ID: objc-sym " + framework + "." + funcName
}
