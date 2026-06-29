// Package rawlib contains the per-construct emitters that convert [meta]
// structures into the raw CGo library Go source files and bridge files.
//
// Each top-level function writes one logical output unit:
//
//   - [Classes] — one .go file per ObjC class, using struct embedding to model
//     the superclass chain.
//   - [ClassInterfaces] — one <Name>able Go interface per ObjC class, enabling
//     mock implementations for testing and polymorphic acceptance.
//   - [Bridge] — a C header (.h) and Objective-C implementation (.m) with
//     thin wrapper functions for every bridged method. Bridge files are
//     compiled with -fno-objc-arc; returned objects are +1 retained.
//   - [Enums] — typed Go const blocks; bitmask enums receive a bitwise String
//     method.
//   - [Structs] — value-type wrappers for C structs (CGRect, CGSize, etc.).
//   - [Protocols] — Go interface types for ObjC @protocols.
//   - [Externs] — package-level constants for extern symbols.
//   - [Functions] — Go wrappers for free C functions.
//   - [Blocks] — named Go func types for ObjC block signatures.
//   - [BlockTrampolines] — CGo trampoline header and implementation.
//   - [ForeignExtensions] — package-level functions for ObjC categories that
//     extend a class owned by a different framework.
//   - [FoundationVariadicWrappers] — hand-authored Go variadic overloads for
//     Foundation methods whose ObjC signature is variadic.
//
// Emitters discover their import requirements as a side effect of type
// resolution: they populate a usedImports map during body generation, then
// write the file header (with import block) followed by the body in a single
// pass.
package rawlib
