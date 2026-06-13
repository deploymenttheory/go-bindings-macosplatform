// Package naming converts ObjC identifiers to idiomatic Go names.
//
// ObjC uses a keyword-argument selector syntax ("objectAtIndex:") that must be
// flattened into a single Go method name ("ObjectAtIndex"). This package
// provides the mapping functions used consistently across all emitters:
//
//   - [MethodName] converts an ObjC selector string to an exported Go method
//     name, handling common suffixes ("UsingBlock:" → drop "Block") and
//     acronym casing.
//   - [ParamName] lowercases the first letter of a parameter name and escapes
//     any identifier that collides with a Go keyword or common builtin.
//   - [PackageName] lowercases the framework name to produce the Go package
//     name ("Foundation" → "foundation").
//   - [BridgeFuncName] produces the C symbol for a bridge function
//     ("foundation_NSArray_objectAtIndex_").
//   - [ProtocolGoTypeName] disambiguates protocol names from same-named
//     classes (NSObject class vs NSObjectProtocol interface).
package naming
