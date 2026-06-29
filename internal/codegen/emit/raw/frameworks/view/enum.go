// Package view holds the pure-data intermediate representation for the raw
// purego framework emitter. A gather phase in package emit resolves metadata and
// type information into these structs; package render turns them into Go source
// through templates only. No struct here carries behaviour or makes a type
// decision — every value is already resolved.
package view

// Enum is a resolved ObjC enum ready to render as a Go type. A named enum emits
// a `type X <underlying>` declaration, a typed const block, and a String method;
// an anonymous enum (IsAnon) emits only an untyped const block.
type Enum struct {
	// GoName is the exported Go type name (empty for an anonymous enum).
	GoName string
	// GoType is the underlying Go integer type (empty for an anonymous enum).
	GoType string
	// CommentBlock is the rendered doc + deprecation comment for the type, each
	// line "// "-prefixed with a trailing newline, or empty.
	CommentBlock string
	// IsBitmask selects the flag-combining String form over the switch form.
	IsBitmask bool
	// IsAnon marks an anonymous enum: only Members is used, rendered as an
	// untyped const block.
	IsAnon bool
	// HasConstBlock is true when the named enum had at least one (deduplicated)
	// member, so a `const (...)` block is emitted even if every member was
	// filtered out as unavailable — matching the original emitter exactly.
	HasConstBlock bool
	// Members are the constant declarations, in source order, already
	// deduplicated and filtered to available members.
	Members []EnumMember
	// StringMembers are the members the String method dispatches on: for the
	// switch form, deduplicated by value; for the bitmask form, the non-zero
	// members. Empty for an anonymous enum.
	StringMembers []EnumMember
}

// EnumMember is one constant of an enum.
type EnumMember struct {
	// ConstName is the exported Go constant name.
	ConstName string
	// Value is the literal integer value as it appears in source.
	Value string
	// CommentBlock is the rendered doc comment for the member ("\t// …\n"), or
	// empty.
	CommentBlock string
}
