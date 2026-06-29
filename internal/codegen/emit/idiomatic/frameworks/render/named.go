// Package render: name-based execution for the construct emitters that render an
// ad-hoc view by template name rather than through a typed Render* function.
package render

import (
	"fmt"
	"io"
)

// Execute runs a template by name against the single embedded template set
// shared with the typed renderers.
func Execute(w io.Writer, name string, data any) error {
	return templates.ExecuteTemplate(w, name, data)
}

// Must is Execute, panicking on error — a template failure here is a codegen bug
// (bad view model or template). Output still flows through gofmt, which catches
// any malformed Go.
func Must(w io.Writer, name string, data any) {
	if err := Execute(w, name, data); err != nil {
		panic(fmt.Sprintf("emit/idiomatic render: template %q: %v", name, err))
	}
}
