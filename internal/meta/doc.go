// Package meta defines the data model that connects the scanner and the code
// generator.
//
// [FrameworkMeta] is the serialisable representation of a single macOS
// framework's public API surface. The scanner writes one
// <framework>-<arch>-<sdk>.gometa.json file per framework; the generator reads
// them back without needing Xcode or a live SDK (--skip-scan mode).
//
// The sub-types — [Class], [Protocol], [Method], [Arg], [Return], [Enum],
// [Struct], [Function], [Extern], [BlockType], [Availability] — mirror the
// ObjC declaration model closely enough to drive accurate code generation while
// discarding Clang-internal detail that is irrelevant to Go.
//
// Serialisation is handled by [Read] and [Write] in io.go using the standard
// encoding/json package. No external dependencies are required.
package meta
