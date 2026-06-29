//go:build darwin

// Package view holds the pure-data IR structs for the idiomatic CGo library
// emitter (idiolib). No metadata or type-mapping imports — every value is
// resolved by the gather phase before render.
package view

// OpinionatedHeaderModel is template data for an idiomatic library file header
// (generated banner, build tag, package clause, and sorted import block).
type OpinionatedHeaderModel struct {
	PkgName string
	Imports []ImportLineModel
}

// ImportLineModel is one import line; Bare omits the alias (path's last element
// already equals the alias, or alias == path).
type ImportLineModel struct {
	Alias string
	Path  string
	Bare  bool
}

// HandleWrapperModel is template data for an opaque C-handle wrapper type with
// its Wrap adopt-constructor and Ptr accessor, followed by its pre-rendered
// methods.
type HandleWrapperModel struct {
	GoName  string
	Base    string
	Methods string // pre-rendered wrapper-func method bodies (may be empty)
}

// WrapperFuncModel is template data for one idiomatic forwarding function or
// method. RetKind matches the idiolib constants: 0 void, 1 plain, 2 error
// (OSStatus), 3 handle (Wrap).
type WrapperFuncModel struct {
	RecvPrefix string // "" for a free function, "(h T) " for a method
	GoName     string
	Params     string
	Call       string // raw.Foo(args)
	RetKind    int
	RetType    string
}
