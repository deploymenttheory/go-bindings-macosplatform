package view

// Protocol is a resolved ObjC protocol rendered as a Go interface: the embedded
// parent interfaces followed by the required method set.
type Protocol struct {
	// CommentBlock is the "// X wraps the ObjC protocol Y." comment (column 0,
	// trailing newline).
	CommentBlock string
	// GoName is the exported Go interface name.
	GoName string
	// Embeds are the embedded parent interface names, already qualified with a
	// package selector when cross-framework.
	Embeds []string
	// Methods are the required interface methods.
	Methods []ProtocolMethod
}

// ProtocolMethod is one method of a protocol interface: its Go name and its
// already-resolved signature (the "(params) ret" fragment).
type ProtocolMethod struct {
	GoName    string
	Signature string
}
