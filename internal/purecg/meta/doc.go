// Package meta defines the data model consumed by the purego code generator.
//
// [FrameworkMeta] is the serialisable representation of a single macOS
// framework's public API surface. The scanner writes one
// <framework>-<arch>-<sdk>.gometa.json file per framework; the purego
// generator reads them back to emit CGo-free Go packages.
package meta
