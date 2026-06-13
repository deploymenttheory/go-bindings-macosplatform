package typemap

import (
	"regexp"
	"strings"
)

// nullabilityRE matches ObjC/Swift nullability qualifiers.
// _Nullable_result must precede _Nullable to avoid the shorter match leaving a
// dangling "_result" suffix (e.g. "CKUserIdentity * _Nullable_result").
var nullabilityRE = regexp.MustCompile(`\s*(_Nullable_result|_Nullable|_Nonnull|_Null_unspecified|__nullable|__nonnull|__null_unspecified)\s*`)

// cvQualRE matches const / volatile qualifiers
var cvQualRE = regexp.MustCompile(`\b(const|volatile|restrict)\s+`)

// cArrayRE normalises C fixed-size array syntax to have a space before the bracket
// so the type mapper's bracket handler can always use " [" as the delimiter.
// E.g. "UInt32[1]" → "UInt32 [1]", "unsigned char[6]" → "unsigned char [6]".
var cArrayRE = regexp.MustCompile(`(\w)\[`)

// objcAttrRE matches ObjC/Swift attribute macros that Clang embeds verbatim in
// qualType strings. All of these are purely advisory annotations with no impact
// on the C ABI; stripping them lets the type mapper see the underlying type.
var objcAttrRE = regexp.MustCompile(`\b(` +
	// Memory-management attributes
	`NS_RETURNS_INNER_POINTER|NS_RETURNS_RETAINED|NS_RETURNS_NOT_RETAINED|NS_NOT_RETAINED|NS_RETAINED|` +
	// Swift interop attributes
	`NS_REFINED_FOR_SWIFT|NS_SWIFT_NAME|NS_SWIFT_UI_ACTOR|WK_SWIFT_UI_ACTOR|NS_SWIFT_UNAVAILABLE_FROM_ASYNC|NS_SWIFT_UNAVAILABLE|` +
	// Availability / deprecation attributes (NS_ prefix)
	`NS_AVAILABLE_MAC|NS_AVAILABLE_IOS|NS_AVAILABLE|NS_DEPRECATED_MAC|NS_DEPRECATED_IOS|NS_DEPRECATED|NS_UNAVAILABLE|` +
	// ARC ownership qualifiers and platform-availability guards
	`NS_REQUIRES_SUPER|__strong|__weak|__unsafe_unretained|__autoreleasing|__kindof|` +
	`__WATCHOS_PROHIBITED|__TVOS_PROHIBITED|__OSX_AVAILABLE_STARTING|__SWIFT_UNAVAILABLE_MSG|` +
	// API availability / deprecation macros (framework-agnostic)
	`API_DEPRECATED_WITH_REPLACEMENT|API_DEPRECATED|API_UNAVAILABLE|API_AVAILABLE|` +
	// CoreFoundation attribute macros
	`CF_RETURNS_RETAINED|CF_RETURNS_NOT_RETAINED|CF_REFINED_FOR_SWIFT|CF_SWIFT_UNAVAILABLE|CF_AVAILABLE_MAC|CF_AVAILABLE|CF_DEPRECATED|` +
	`CF_NOESCAPE|CF_CONSUMED|CF_RELEASES_ARGUMENT|` +
	// ARKit attribute macros
	`AR_OBJECT_RETURNS_RETAINED|AR_REFINED_FOR_SWIFT|` +
	// Framework-specific RETURNS_RETAINED attributes (not covered by NS_/CF_ prefixes)
	`NW_RETURNS_RETAINED|LA_RETURNS_RETAINED|SEC_RETURNS_RETAINED|` +
	`OS_OBJECT_RETURNS_RETAINED|XPC_RETURNS_RETAINED|OS_WORKGROUP_RETURNS_RETAINED|DISPATCH_RETURNS_RETAINED|` +
	// Old-style macOS availability macros (AVAILABLE_MAC_OS_X_VERSION_X_X_AND_LATER)
	`AVAILABLE_MAC_OS_X_VERSION_\w+_AND_LATER|` +
	// Framework-specific visibility / deprecation macros
	`COREMOTION_EXPORT|CI_GL_DEPRECATED_MAC|` +
	`COREVIDEO_GL_DEPRECATED|CS_AVAILABLE_STARTING|CT_AVAILABLE|MODELCOLLECTION_SUNSET|` +
	`OPENGL_DEPRECATED|PDFKIT_DEPRECATED|PDFKIT_AVAILABLE|IC_UNAVAILABLE|IC_AVAILABLE|` +
	`IC_DEPRECATED_WITH_REPLACEMENT|ITLIB_AVAILABLE|MP_DEPRECATED_WITH_REPLACEMENT|` +
	`SCN_GL_DEPRECATED_MAC|` +
	`CS_AVAILABLE|CSSM_GUID|DEPRECATED_ATTRIBUTE` +
	`)\s*`)

// genericRE matches NSArray<Foo *> — captures the outer class name and the param list
var genericRE = regexp.MustCompile(`^(\w+)\s*<(.+)>\s*\*?`)

// blockRE matches void (^)(args...) style block types, including variants with
// qualifiers after the caret such as void (^ _Nonnull)(args...).
var blockRE = regexp.MustCompile(`^(.+?)\s*\(\^[^)]*\)\s*\((.*)?\)\s*$`)

// gnuAttrRE matches the start of a __attribute__((...)) expression so we can
// locate and remove it with parenthesis-counting (the regex handles the keyword;
// stripGNUAttributes does the balanced-paren removal).
var gnuAttrRE = regexp.MustCompile(`__attribute__\s*\(`)

// stripGNUAttributes removes all __attribute__((...)) constructs from s,
// including those with arbitrarily nested parentheses such as
// __attribute__((__vector_size__(16 * sizeof(signed char)))).
func stripGNUAttributes(s string) string {
	for {
		loc := gnuAttrRE.FindStringIndex(s)
		if loc == nil {
			return s
		}
		// loc[1] points just past the first '(' in "__attribute__("; depth starts at 1.
		start, i, depth := loc[0], loc[1], 1
		for i < len(s) && depth > 0 {
			switch s[i] {
			case '(':
				depth++
			case ')':
				depth--
			}
			i++
		}
		s = s[:start] + s[i:]
	}
}

// NormaliseBlock strips ObjC attribute macros, GNU __attribute__((...))
// constructs, and nullability qualifiers but preserves const/volatile, so that
// block casts remain compatible with parameters that have const-qualified
// argument types.
func NormaliseBlock(qt string) string {
	s := stripGNUAttributes(qt)
	s = objcAttrRE.ReplaceAllString(s, " ")
	s = nullabilityRE.ReplaceAllString(s, " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

// Normalise strips GNU __attribute__((...)) constructs, ObjC attribute macros,
// nullability qualifiers, leading const/volatile, and normalises whitespace.
func Normalise(qt string) string {
	s := stripGNUAttributes(qt)
	s = objcAttrRE.ReplaceAllString(s, " ")
	s = nullabilityRE.ReplaceAllString(s, " ")
	s = cvQualRE.ReplaceAllString(s, "")
	s = strings.Join(strings.Fields(s), " ")
	// Strip trailing pointer qualifier: "NSString * const" → "NSString *"
	// (the const applies to the pointer itself, not the pointee; irrelevant for type mapping).
	s = strings.ReplaceAll(s, "* const", "*")
	s = strings.ReplaceAll(s, "*const", "*")
	// Normalise C fixed-size array syntax to always have a space before the bracket
	// so the array handler can reliably detect " [size]": "UInt32[1]" → "UInt32 [1]".
	s = cArrayRE.ReplaceAllString(s, "$1 [")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

// IsVoid returns true for the void type (no return value).
func IsVoid(qt string) bool {
	return Normalise(qt) == "void"
}

// IsPointer returns true if the (normalised) type is a pointer.
func IsPointer(qt string) bool {
	return strings.Contains(qt, "*")
}

// IsDoublePointer returns true for NSError ** and similar out-param patterns.
// Generic type parameters like NSArray<NSString *> are NOT double pointers — the
// inner asterisk is part of the element type, not an extra indirection level.
// We only count asterisks that appear after the last closing angle bracket.
func IsDoublePointer(qt string) bool {
	n := Normalise(qt)
	suffix := n
	if i := strings.LastIndex(n, ">"); i >= 0 {
		suffix = n[i+1:]
	}
	return strings.Count(suffix, "*") >= 2
}

// IsBlock returns true for ObjC block types: returnType (^)(args) or
// returnType (^ _Nonnull)(args) etc. Checks for the caret introducer only;
// qualifiers between ^ and ) are ignored.
func IsBlock(qt string) bool {
	return strings.Contains(qt, "(^")
}

// IsInstancetype returns true for the special ObjC instancetype return.
func IsInstancetype(qt string) bool {
	return strings.TrimSpace(Normalise(qt)) == "instancetype"
}

// IsID returns true for bare `id` (any object).
func IsID(qt string) bool {
	n := strings.TrimSpace(Normalise(qt))
	return n == "id"
}

// IDProtocols extracts the ordered list of protocol names from a `id<P1, P2>`
// type expression. Returns nil for any input that is not exactly an `id<...>`
// form (bare `id`, `NSString *`, blocks, etc.). The returned names are
// trimmed of whitespace.
//
//	"id<VZGraphicsDisplayObserver>"           → ["VZGraphicsDisplayObserver"]
//	"id<NSCopying, NSSecureCoding> _Nullable" → ["NSCopying", "NSSecureCoding"]
//	"id"                                      → nil
//	"NSString *"                              → nil
func IDProtocols(qt string) []string {
	n := strings.TrimSpace(Normalise(qt))
	if !strings.HasPrefix(n, "id<") {
		return nil
	}
	// Find the matching closing '>'. The contents are flat protocol names
	// separated by commas — no nested angle brackets.
	open := strings.Index(n, "<")
	close := strings.Index(n[open:], ">")
	if open < 0 || close < 0 {
		return nil
	}
	inner := n[open+1 : open+close]
	parts := strings.Split(inner, ",")
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		names = append(names, p)
	}
	if len(names) == 0 {
		return nil
	}
	return names
}

// IsClass returns true for the ObjC `Class` meta-object type.
func IsClass(qt string) bool {
	n := strings.TrimSpace(Normalise(qt))
	return n == "Class"
}

// IsSEL returns true for ObjC selectors.
func IsSEL(qt string) bool {
	n := strings.TrimSpace(Normalise(qt))
	return n == "SEL"
}

// IsBOOL returns true for the ObjC BOOL type.
func IsBOOL(qt string) bool {
	n := strings.TrimSpace(Normalise(qt))
	return n == "BOOL"
}

// IsBOOLPointer returns true for "BOOL *" — a pointer-to-BOOL used as an
// in-out stop flag in ObjC enumeration block callbacks.
func IsBOOLPointer(qt string) bool {
	n := strings.TrimSpace(Normalise(qt))
	return n == "BOOL *"
}

// ClassName extracts "NSString" from "NSString *" or "NSString * _Nullable".
// Returns empty string if the type does not look like a named ObjC object.
func ClassName(qt string) string {
	n := Normalise(qt)
	n = strings.TrimSuffix(strings.TrimSpace(n), "*")
	n = strings.TrimSpace(n)
	// strip generic params
	if idx := strings.Index(n, "<"); idx > 0 {
		n = n[:idx]
	}
	n = strings.TrimSpace(n)
	// A class name is a single identifier: no spaces, starts uppercase.
	if len(n) == 0 || n[0] < 'A' || n[0] > 'Z' || strings.ContainsAny(n, " \t") {
		return ""
	}
	return n
}

// GenericParam extracts the first type parameter from a generic type string.
// E.g. "NSArray<ObjectType> *" → "ObjectType".
// Returns empty if no generic param is present.
// Use GenericParams for multi-parameter generics like NSDictionary<KeyType, ObjectType>.
func GenericParam(qt string) string {
	params := GenericParams(qt)
	if len(params) == 0 {
		return ""
	}
	return params[0]
}

// GenericParams extracts all type parameters from a generic type string.
// E.g. "NSDictionary<KeyType, ObjectType> *" → ["KeyType", "ObjectType"].
// Returns nil if no generic params are present.
func GenericParams(qt string) []string {
	m := genericRE.FindStringSubmatch(Normalise(qt))
	if len(m) < 3 {
		return nil
	}
	raw := strings.TrimSpace(m[2])
	parts := splitArgs(raw)
	params := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimSuffix(p, "*")
		p = strings.TrimSpace(p)
		if p != "" {
			params = append(params, p)
		}
	}
	if len(params) == 0 {
		return nil
	}
	return params
}

// ParseBlock parses a block type string into its return type and argument types.
// Returns (returnType, []argTypes, ok).
func ParseBlock(qt string) (string, []string, bool) {
	m := blockRE.FindStringSubmatch(Normalise(qt))
	if len(m) < 3 {
		return "", nil, false
	}
	retType := strings.TrimSpace(m[1])
	argStr := strings.TrimSpace(m[2])
	var args []string
	if argStr != "" && argStr != "void" {
		for _, a := range splitArgs(argStr) {
			args = append(args, strings.TrimSpace(a))
		}
	}
	return retType, args, true
}

// splitArgs splits a comma-separated argument list, respecting nested angle brackets.
func splitArgs(s string) []string {
	var args []string
	depth := 0
	start := 0
	for i, ch := range s {
		switch ch {
		case '<', '(':
			depth++
		case '>', ')':
			depth--
		case ',':
			if depth == 0 {
				args = append(args, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		args = append(args, s[start:])
	}
	return args
}
