// Package render turns a fully-resolved view (the IR built by the gather phase)
// into Go source. It does so through templates only: render makes no resolution
// decisions and imports no metadata or type-mapping packages — every value it
// needs is already present on the view. Its FuncMap holds pure formatting
// helpers, never type logic.
package render

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/idiomatic/view"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// funcMap holds the template helpers. Each is a pure formatting function; none
// decides a Go type (that is the gather phase's job).
var funcMap = template.FuncMap{
	"comment":      comment,
	"wrap":         wrap,
	"retOutValues": retOutValues,
	"retOutZeros":  retOutZeros,
	"join":         join,
}

var templates = template.Must(
	template.New("render").Funcs(funcMap).ParseFS(templatesFS, "templates/*.tmpl"),
)

// comment renders documentation prose as a Go "// " comment block: the first
// line is prefixed with "// " and every subsequent line likewise. It returns ""
// for empty prose so callers can omit the comment entirely.
func comment(doc string) string {
	if doc == "" {
		return ""
	}
	return "// " + strings.ReplaceAll(doc, "\n", "\n// ")
}

// Structs renders the value-struct definitions for a package as a Go source
// fragment (a package body, before file assembly and gofmt).
func Structs(structs []view.Struct) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "struct", structs); err != nil {
		return nil, fmt.Errorf("render structs: %w", err)
	}
	return buf.Bytes(), nil
}

// Enums renders the concrete enum definitions for a package as a Go source
// fragment (a package body, before file assembly and gofmt). Each enum becomes a
// `type X <underlying>` declaration with a typed const block and a String method.
func Enums(enums []view.Enum) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "enums", enums); err != nil {
		return nil, fmt.Errorf("render enums: %w", err)
	}
	return buf.Bytes(), nil
}

// Method renders one Go method (or package-level function) — signature, doc, and
// body — from a resolved view.Method. The body comes from the method's Dispatch,
// so no Go syntax is assembled by string concatenation in the caller.
func Method(m view.Method) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "method", m); err != nil {
		return nil, fmt.Errorf("render method %s: %w", m.GoName, err)
	}
	return buf.Bytes(), nil
}

// AsyncMethod renders a completion-handler method as a blocking, ctx-aware call.
func AsyncMethod(m view.AsyncMethod) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "async", m); err != nil {
		return nil, fmt.Errorf("render async method %s: %w", m.GoName, err)
	}
	return buf.Bytes(), nil
}

// SliceMethod renders an array-returning getter as a slice-returning method.
func SliceMethod(m view.SliceMethod) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "slice", m); err != nil {
		return nil, fmt.Errorf("render slice method %s: %w", m.GoName, err)
	}
	return buf.Bytes(), nil
}

// BoolNSErrorMethod renders a BOOL+NSError method as an error-returning method.
func BoolNSErrorMethod(m view.BoolNSErrorMethod) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "boolnserror", m); err != nil {
		return nil, fmt.Errorf("render boolnserror method %s: %w", m.GoName, err)
	}
	return buf.Bytes(), nil
}

// Constants renders the global-constant accessor functions for a package as a Go
// source fragment (before file assembly and gofmt).
func Constants(constants []view.Constant) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "constants", constants); err != nil {
		return nil, fmt.Errorf("render constants: %w", err)
	}
	return buf.Bytes(), nil
}

// Sentinels renders the error-sentinel variable declarations for a package as a
// Go source fragment (before file assembly and gofmt).
func Sentinels(sentinels []view.ErrorSentinel) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "sentinels", sentinels); err != nil {
		return nil, fmt.Errorf("render sentinels: %w", err)
	}
	return buf.Bytes(), nil
}

// Funcs renders the C-function wrappers for a package as a Go source fragment
// (before file assembly and gofmt).
func Funcs(funcs []view.Func) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "cfuncs", funcs); err != nil {
		return nil, fmt.Errorf("render cfuncs: %w", err)
	}
	return buf.Bytes(), nil
}
