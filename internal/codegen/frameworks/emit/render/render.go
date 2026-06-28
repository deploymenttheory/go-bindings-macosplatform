// Package render turns the raw purego emitter's resolved view (package view)
// into Go source through templates only. It makes no resolution decisions and
// imports no metadata or type-mapping packages — every value it needs is already
// present on the view. The output is pre-gofmt; the caller canonicalises it.
package render

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/view"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

var templates = template.Must(template.ParseFS(templatesFS, "templates/*.tmpl"))

// Enums renders all enum declarations for a package as a Go source fragment.
func Enums(enums []view.Enum) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "enums", enums); err != nil {
		return nil, fmt.Errorf("render enums: %w", err)
	}
	return buf.Bytes(), nil
}

// Structs renders all struct type declarations for a package as a Go source
// fragment.
func Structs(structs []view.Struct) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "structs", structs); err != nil {
		return nil, fmt.Errorf("render structs: %w", err)
	}
	return buf.Bytes(), nil
}

// TypedefAliases renders all C-typedef Go type aliases for a package as a Go
// source fragment.
func TypedefAliases(aliases []view.TypedefAlias) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "typedefs", aliases); err != nil {
		return nil, fmt.Errorf("render typedef aliases: %w", err)
	}
	return buf.Bytes(), nil
}

// Externs renders all extern-symbol accessor functions for a package as a Go
// source fragment.
func Externs(externs []view.Extern) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "externs", externs); err != nil {
		return nil, fmt.Errorf("render externs: %w", err)
	}
	return buf.Bytes(), nil
}

// Protocols renders all protocol interface declarations for a package as a Go
// source fragment.
func Protocols(protocols []view.Protocol) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "protocols", protocols); err != nil {
		return nil, fmt.Errorf("render protocols: %w", err)
	}
	return buf.Bytes(), nil
}
