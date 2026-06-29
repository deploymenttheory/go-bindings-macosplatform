//go:build darwin

// Package render executes the idiomatic CGo library templates. It is name-based:
// the idiolib gather phase builds a view.*Model and renders it by template name.
package render

import (
	"embed"
	"io"
	"sync"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var (
	tmplOnce sync.Once
	tmplSet  *template.Template
)

func loadedTemplates() *template.Template {
	tmplOnce.Do(func() {
		var err error
		tmplSet, err = template.ParseFS(templateFS, "templates/*.tmpl")
		if err != nil {
			panic("emit/idiomatic/libraries render: failed to parse embedded templates: " + err.Error())
		}
	})
	return tmplSet
}

// Execute runs the named template with data, writing the result to w.
func Execute(w io.Writer, name string, data any) error {
	return loadedTemplates().ExecuteTemplate(w, name, data)
}
