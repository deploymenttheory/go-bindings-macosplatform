// Package typemap resolves ObjC qualType strings to their Go equivalents for the
// LIBRARIES pipeline (cgo/C) only.
//
// SCOPE — pipeline-specific by design. This package is deliberately NOT shared
// with its frameworks counterpart, internal/codegen/frameworks/typemap: the
// purego/cgo split is a non-negotiable of the architecture, and the two mappers
// diverge in BEHAVIOUR, not merely in structure. Same-named helpers on the two
// sides (IsCoreFoundationOpaqueRef, Normalise, splitArgs, the block/fn-ptr parser,
// the primitive tables) have different implementations and different outputs and
// MUST NOT be unified — doing so would change generated code in at least one
// pipeline. This side additionally carries a CType() dimension (structs_ctype.go)
// the frameworks side has no use for. Logic that is genuinely identical across
// both pipelines lives in the shared packages instead: internal/codegen/emit/
// structlayout (ABI/layout), internal/codegen/naming/core (naming), internal/
// codegen/pipeline/structindex (struct ownership), internal/codegen/shared/
// fileasm (file assembly).
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
