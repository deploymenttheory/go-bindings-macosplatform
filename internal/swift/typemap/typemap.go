// Package typemap maps Swift type strings (from .swiftinterface) to Go type strings.
package typemap

import "strings"

// swiftRawTypes are Swift types that can be enum raw-value backing types.
// The map value is the corresponding Go type.
var swiftRawTypes = map[string]string{
	"Int":     "int64",
	"Int8":    "int8",
	"Int16":   "int16",
	"Int32":   "int32",
	"Int64":   "int64",
	"UInt":    "uint64",
	"UInt8":   "uint8",
	"UInt16":  "uint16",
	"UInt32":  "uint32",
	"UInt64":  "uint64",
	"Float":   "float32",
	"Float16": "float32",
	"Double":  "float64",
	"String":  "string",
	"Bool":    "bool",
}

// fieldTypes maps (stripped) Swift type names to Go type strings for struct fields.
var fieldTypes = map[string]string{
	"String":           "string",
	"Bool":             "bool",
	"Int":              "int64",
	"Int8":             "int8",
	"Int16":            "int16",
	"Int32":            "int32",
	"Int64":            "int64",
	"UInt":             "uint64",
	"UInt8":            "uint8",
	"UInt16":           "uint16",
	"UInt32":           "uint32",
	"UInt64":           "uint64",
	"Float":            "float32",
	"Float16":          "float32",
	"Double":           "float64",
	"Data":             "[]byte",
	"Date":             "int64", // Unix timestamp
	"URL":              "string",
	"UUID":             "string",
	"Locale":           "string",
	"Locale.Language":  "string",
	"AttributedString": "string",
	"Decimal":          "float64",
	"TimeInterval":     "float64",
	"Void":             "",
	// Common Foundation types
	"NSString": "string",
	"NSData":   "[]byte",
	"NSDate":   "int64",
	"NSURL":    "string",
	"NSUUID":   "string",
	"NSError":  "error",
}

// stripModulePrefix removes well-known module qualifiers from a Swift type string.
// "Swift.String" → "String", "Foundation.Locale.Language" → "Locale.Language"
func stripModulePrefix(s string) string {
	for _, pfx := range []string{"Swift.", "Foundation.", "CoreFoundation.", "Darwin."} {
		if strings.HasPrefix(s, pfx) {
			return s[len(pfx):]
		}
	}
	return s
}

// Map resolves a Swift type string to a Go type string.
// optional=true means the Swift type had a trailing ? and the result should be pointer-typed
// (e.g. "*string" instead of "string") where applicable.
// Returns "" if the type is unknown or should be skipped.
func Map(swiftType string, optional bool) string {
	s := strings.TrimSpace(swiftType)
	s = stripModulePrefix(s)

	goType := fieldTypes[s]
	if goType == "" {
		// Fallback: any remaining single-identifier type that looks like a Foundation
		// string-like name gets string. Unknown compound types are skipped.
		if !strings.ContainsAny(s, ".<>[]()") {
			// Single identifier we don't recognise — skip (leave as empty).
			return ""
		}
		return ""
	}

	if optional && goType != "" && goType != "[]byte" && !strings.HasPrefix(goType, "[]") {
		return "*" + goType
	}
	return goType
}

// RawEnumGoType resolves the first conformance token of a Swift enum to its Go
// backing type for raw-value enums. Returns "" if the token is not a Swift scalar.
func RawEnumGoType(firstConformance string) string {
	s := stripModulePrefix(strings.TrimSpace(firstConformance))
	return swiftRawTypes[s]
}

// IsRawType returns true when the stripped Swift type name is a known raw-value type.
func IsRawType(swiftType string) bool {
	s := stripModulePrefix(strings.TrimSpace(swiftType))
	_, ok := swiftRawTypes[s]
	return ok
}
