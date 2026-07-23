//go:build darwin

package idiofw

import (
	"strings"
	"unicode"
)

// dispatch_gen.go contains the small building blocks the generator uses to emit
// a Go method that calls an Objective-C method directly. Given one Go
// parameter or return value, these functions decide how to convert it to or from
// the Objective-C call (for example, a Go string becomes an NSString) and what
// the call's result type should be. They also name the variables that hold
// Objective-C selectors. Each function is a plain string transformation with no
// file or network access, so they are easy to unit-test on their own.

// objKind says how one Go value is converted when calling Objective-C. The
// underlying call layer already handles integers, floats, booleans, and Go enum
// types (which are integers) without help, so only strings and objects need an
// explicit conversion.
type objKind int

const (
	// kindScalar is an integer/float value passed and returned unchanged.
	kindScalar objKind = iota
	// kindBool is a Go bool, passed and returned unchanged (Send[bool]).
	kindBool
	// kindEnum is a named idiomatic enum type, passed/returned unchanged.
	kindEnum
	// kindString is a Go string bridged to/from NSString.
	kindString
	// kindObject is an idiomatic object (a wrapper pointer or obj.Object),
	// bridged to/from objc.ID.
	kindObject
	// kindArray is an Objective-C array bridged to/from a Go slice.
	kindArray
	// kindVoid is the absence of a return value.
	kindVoid
)

// classifyGoType maps a resolved idiomatic Go type (as it appears in a method
// signature) to the objKind that drives dispatch marshaling. isEnum reports
// whether a bare type name is one of the framework's enum types; isObject
// reports whether a bare name is an idiomatic object wrapper (a class). The two
// predicates supply the metadata context this otherwise-pure function needs.
//
// It deliberately handles only the kinds the dispatch primitives marshal today
// (void/string/bool/object/enum/scalar); composite shapes (slices, generics,
// blocks, value structs) are classified elsewhere as they are implemented and
// must not reach here.
func classifyGoType(goType string, isEnum, isObject func(name string) bool) objKind {
	switch goType {
	case "", "void":
		return kindVoid
	case "string":
		return kindString
	case "bool":
		return kindBool
	}
	// Any pointer-to-wrapper or object interface marshals through the bridge.
	if strings.HasPrefix(goType, "*") {
		return kindObject
	}
	if goType == "obj.Object" || strings.HasSuffix(goType, ".Object") {
		return kindObject
	}
	base := goType
	if i := strings.LastIndexByte(base, '.'); i >= 0 {
		base = base[i+1:]
	}
	if isEnum(base) {
		return kindEnum
	}
	if isObject(base) {
		return kindObject
	}
	return kindScalar
}

// marshalArg returns the Go expression that converts an idiomatic argument named
// `name`, classified as `k`, into the value passed to objc.Send. Strings become
// an NSString id; objects yield their underlying id via the sealed bridge;
// everything else (scalars, bools, enums) is passed unchanged because purego
// marshals those kinds directly.
func marshalArg(name string, k objKind) string {
	switch k {
	case kindString:
		return "purego.NSString(" + name + ")"
	case kindObject:
		return "objref.IDOf(" + name + ")"
	default:
		return name
	}
}

// sendReturnType returns the type argument T for objc.Send[T] given the idiomatic
// return classification. Strings and objects come back as a raw objc.ID (then
// marshaled by marshalReturn); scalars/bools/enums use their idiomatic type so
// purego decodes the value directly.
func sendReturnType(k objKind, idiomaticType string) string {
	switch k {
	case kindString, kindObject:
		return "objc.ID"
	default:
		return idiomaticType
	}
}

// marshalReturn renders the body that turns the objc.Send result (bound to the
// local name `recv`, e.g. "_r") into the idiomatic return value, given the
// classification and — for objects — the wrapping expression `wrapExpr` (a
// format with a single %s placeholder for the id local, e.g.
// "stringFromID(%s)" or "obj.Wrap(%s)"). It returns the full statement list
// including the `return`. For kindVoid it returns "".
func marshalReturn(k objKind, recv, idiomaticType, wrapExpr string) string {
	switch k {
	case kindVoid:
		return ""
	case kindString:
		return "if " + recv + " == 0 {\nreturn \"\"\n}\nreturn purego.GoString(" + recv + ")"
	case kindObject, kindArray:
		return "return " + strings.Replace(wrapExpr, "%s", recv, 1)
	default:
		return "return " + recv
	}
}

// zeroValue returns the Go zero value literal for an idiomatic return type,
// used on the error path of a fallible method. Objects and strings have well
// known zeros; everything else falls back to a typed zero via conversion.
func zeroValue(k objKind, idiomaticType string) string {
	switch k {
	case kindVoid:
		return ""
	case kindString:
		return `""`
	case kindObject, kindArray:
		// Object-kind results whose idiomatic type is a Go value (an NSURL
		// surfaced as string, an NSDate as time.Time) need that value's zero;
		// wrapper pointers, slices, and maps zero to nil.
		switch idiomaticType {
		case "string":
			return `""`
		case "time.Time":
			return "time.Time{}"
		}
		return "nil"
	case kindBool:
		return "false"
	case kindEnum:
		return idiomaticType + "(0)"
	default:
		// A bare host pointer result (an unsafe.Pointer surfaced from a raw C
		// pointer or NS_RETURNS_INNER_POINTER return) zeroes to nil, not 0.
		if idiomaticType == "unsafe.Pointer" {
			return "nil"
		}
		// Numeric scalars.
		return "0"
	}
}

// selectorVarName returns a stable, unexported package-level variable name for an
// Objective-C selector, scoped by the class's Go type so two classes that share
// a selector spelling do not collide. e.g. (class "String", selector
// "componentsSeparatedByString:") -> "_selStringComponentsSeparatedByString".
func selectorVarName(goTypeName, selector string) string {
	return "_sel" + goTypeName + selectorIdent(selector)
}

// selectorIdent converts an ObjC selector to a CamelCase identifier fragment by
// upper-casing each colon-delimited keyword and dropping the colons. e.g.
// "initWithURL:readOnly:error:" -> "InitWithURLReadOnly Error" without spaces:
// "InitWithURLReadOnlyError".
func selectorIdent(selector string) string {
	var b strings.Builder
	for _, part := range strings.Split(selector, ":") {
		if part == "" {
			continue
		}
		b.WriteString(capitalizeFirst(part))
	}
	return b.String()
}

// lowerFirst lower-cases the first rune of s, leaving the rest unchanged.
func lowerFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// capitalizeFirst upper-cases the first rune of s, leaving the rest unchanged.
// Selector keywords are already camelCase, so this yields a clean exported
// CamelCase fragment (e.g. "readOnly" -> "ReadOnly").
func capitalizeFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
