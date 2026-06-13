//go:build darwin

// Package scanner drives the first phase of the code-generation pipeline.
//
// It invokes xcrun clang with -ast-dump=json on a framework's umbrella header,
// unmarshals the resulting Clang JSON AST, and walks it to produce a
// [meta.FrameworkMeta] value that captures the framework's complete public API:
// classes, protocols, enums, structs, free functions, extern constants, block
// types, and typedefs.
//
// Extraction is restricted to declarations whose source file belongs to the
// named framework's own headers (filter.go), so re-exported types from other
// frameworks are skipped and attributed to their true owner during the load
// phase.
//
// Key entry points:
//   - [DumpAST] — invokes xcrun clang and returns the parsed AST root node.
//   - [Extract] — walks an AST value and returns a populated [meta.FrameworkMeta].
package scanner
