// Package typemap resolves ObjC qualType strings to their Go equivalents.
//
// [Mapper] is the central resolver. Its [Mapper.GoType] method accepts a raw
// ObjC type string (e.g. "NSArray<NSString *> *") and a [Context] describing
// the declaration being emitted, and returns the corresponding Go type string
// (e.g. "*foundation.NSArray[*foundation.NSString]").
//
// Resolution handles:
//   - Primitives and ObjC integer aliases (NSInteger → int64, BOOL → bool).
//   - Class references, with cross-framework import tracking as a side effect
//     (discovered imports are collected in the caller-supplied [ImportSet]).
//   - Generic type parameters (ObjectType on NSArray[T] → T).
//   - Protocol references ("id<Protocol>" → interface type).
//   - Enum and struct references, resolved to their owning framework package.
//   - ObjC block signatures, converted to Go func types.
//   - One-level typedef expansion for otherwise-unrecognised type names.
//   - CoreFoundation opaque pointer types (CFArrayRef, etc.).
//
// When a cross-framework reference cannot be emitted (because the import would
// introduce a cycle), [Mapper] substitutes unsafe.Pointer and records a
// diagnostic. The set of blocked imports is computed by the pipeline before
// generation begins and supplied via [Mapper.BlockedImports].
//
// Results are cached by a key composed of the normalised qualType and the
// relevant Context fields, with side effects (import additions) replayed on
// cache hits.
package typemap
