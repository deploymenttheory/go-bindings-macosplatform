package view

// Extern is a resolved accessor for an extern global symbol, read at runtime via
// purego.Dlsym. The Form selects the body shape:
//   - "raw"    — returns the raw uintptr address
//   - "fromid" — reads an ObjC object reference and wraps it via FromIDCall
//   - "string" — reads a char* as a Go string
//   - "value"  — reads a value type, returning Zero when the symbol is absent
type Extern struct {
	// CommentBlock is the doc + deprecation comment (column 0, trailing newline).
	CommentBlock string
	// GoName is the exported accessor function name.
	GoName string
	// RetType is the accessor's Go return type ("uintptr" for the raw form).
	RetType string
	// DylibVar is the package-level variable holding the loaded dylib handle.
	DylibVar string
	// Symbol is the C symbol name passed to Dlsym.
	Symbol string
	// Form selects the body shape (see above).
	Form string
	// GoType is the value type read for the "value" form.
	GoType string
	// FromIDCall is the typed-wrapper expression for the "fromid" form.
	FromIDCall string
	// Zero is the zero value returned by the "value" form when the symbol is
	// absent.
	Zero string
}
