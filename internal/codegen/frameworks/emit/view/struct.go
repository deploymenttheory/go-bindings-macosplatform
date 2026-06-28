package view

// Struct is a resolved C/ObjC struct ready to render as a Go type. A struct with
// no fields renders as an opaque empty struct; otherwise it renders its exported
// fields with their resolved Go types.
type Struct struct {
	// CommentBlock is everything rendered before the `type` keyword — the doc
	// comment, any deprecation line, the "// C struct: <name>" note, and (for an
	// opaque struct) the "// X is an opaque type." line — each "// "-prefixed
	// with a trailing newline. Already at column 0, matching the original.
	CommentBlock string
	// GoName is the exported Go type name.
	GoName string
	// IsOpaque marks a zero-field struct, rendered as `struct{}`.
	IsOpaque bool
	// Fields are the exported fields, in declaration order (empty when opaque).
	Fields []StructField
}

// StructField is one exported struct field with its resolved Go type.
type StructField struct {
	GoName string
	GoType string
}

// TypedefAlias is a resolved C typedef rendered as a Go type alias
// (`type X = <RHS>`), for example `NSRect = CGRect` or an opaque-pointer
// `FooRef = *Foo`.
type TypedefAlias struct {
	// CommentBlock is the doc comment before the alias (column 0, trailing
	// newline).
	CommentBlock string
	// GoName is the exported alias name.
	GoName string
	// RHS is the right-hand side of the alias: a type, or "*Type" for an
	// opaque-pointer typedef.
	RHS string
}
