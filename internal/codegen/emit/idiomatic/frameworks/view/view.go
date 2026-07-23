// Package view is the intermediate representation (IR) of a fully-resolved
// idiomatic framework package. It is pure data: every decision (type mapping,
// hierarchy, documentation, marshaling) is made by the gather phase and recorded
// here, and the render phase turns it into Go source through templates only.
//
// view imports nothing from the rest of the emitter, so it cannot smuggle in
// resolution logic — the single-purpose split that keeps gather and render
// honest.
package view

// Struct is a value-type struct re-emitted in an idiomatic package (for example
// CGRect, CGPoint, NSRange). It is passed to Objective-C by value, so the field
// names, Go types, and order match what the framework expects.
type Struct struct {
	// GoName is the exported Go type name.
	GoName string
	// Doc is the documentation prose, already cleaned of HeaderDoc/Doxygen tags;
	// empty when the symbol has no documentation.
	Doc string
	// Fields are the struct's fields, in declaration order.
	Fields []Field
	// IsOpaque marks a struct the C headers declare with no members (e.g. NSZone).
	// It renders as `struct{}`, matching the raw layer, so callers can still name
	// and pass a pointer to it. A struct whose members all happen to be skipped
	// (private bitfields) is NOT opaque — it renders with an empty body.
	IsOpaque bool
}

// Field is one field of a value Struct.
type Field struct {
	// GoName is the exported field name.
	GoName string
	// GoType is the field's resolved Go type (a primitive or another value
	// struct in the same package).
	GoType string
}

// Protocol is an Objective-C protocol re-emitted as a Go interface — the
// idiomatic duck-typed counterpart of the raw layer's protocol interface. A Go
// value satisfies it by declaring the listed methods.
type Protocol struct {
	// Doc is the one-line interface comment; empty when none.
	Doc string
	// GoName is the exported Go interface name.
	GoName string
	// Methods are the interface methods (name + full signature).
	Methods []ProtocolMethod
}

// ProtocolMethod is one method of a Protocol interface.
type ProtocolMethod struct {
	// GoName is the exported method name.
	GoName string
	// Signature is the method's parenthesised parameter list plus return clause,
	// e.g. "(index uint) obj.Object".
	Signature string
}

// TypedefAlias is a Go type alias re-emitted for a C typedef (e.g. NSRect =
// CGRect, or the opaque-pointer form Id = *ObjcObject), so callers can name the
// alias through the idiomatic package. RHS is the fully-resolved right-hand side,
// already localized to a hermetic type (a same-package type or a sibling
// idiomatic package's).
type TypedefAlias struct {
	// Doc is the one-line comment describing the alias; empty when none.
	Doc string
	// GoName is the exported alias name.
	GoName string
	// RHS is the aliased type expression.
	RHS string
}

// HandleType is a distinct named handle type the idiomatic layer emits for an
// opaque CoreFoundation / handle typedef: `type CFArrayRef struct{ obj.Object }`.
// It embeds obj.Object (so it still satisfies Object and carries the lifecycle
// methods) while being its own Go type, giving callers the compile-time type
// distinction a generic obj.Object cannot. Its zero value wraps no object; the
// generated IsNil method reports that NULL state.
type HandleType struct {
	// Doc is the one-line comment describing the handle type; empty when none.
	Doc string
	// GoName is the exported handle type name (e.g. "CFArrayRef").
	GoName string
	// ImmutableType, when non-empty, is the immutable counterpart type name of a
	// CFMutable<X>Ref handle (e.g. "CFArrayRef" for "CFMutableArrayRef"); the
	// generated ImmutableMethod converts to it. In C a mutable ref IS-A immutable
	// ref, a subtyping the distinct Go types lose, so this restores the widening.
	ImmutableType string
	// ImmutableMethod is the conversion method name (e.g. "AsArray").
	ImmutableMethod string
}
