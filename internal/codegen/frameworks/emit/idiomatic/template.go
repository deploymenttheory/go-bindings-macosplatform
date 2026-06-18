//go:build darwin

package idiomatic

import (
	"embed"
	"fmt"
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

// loadedTemplates returns the parsed template set, initialised once on first
// call. All .tmpl files under templates/ are embedded at build time, so ParseFS
// only fails on a template syntax error — caught at test/generation time.
func loadedTemplates() *template.Template {
	tmplOnce.Do(func() {
		var err error
		tmplSet, err = template.ParseFS(templateFS, "templates/*.tmpl")
		if err != nil {
			panic("emit/idiomatic: failed to parse embedded templates: " + err.Error())
		}
	})
	return tmplSet
}

// executeTemplate executes the named template with data, writing to w.
func executeTemplate(w io.Writer, name string, data any) error {
	return loadedTemplates().ExecuteTemplate(w, name, data)
}

// renderTemplate executes the named template, panicking on error. Template
// execution only fails on a codegen bug (bad view-model or template logic), so
// failing loudly is correct for a generator — and it lets the void write*
// renderers stay void. The output still flows through WriteGoFile/gofmt, which
// catches any malformed Go.
func renderTemplate(w io.Writer, name string, data any) {
	if err := executeTemplate(w, name, data); err != nil {
		panic(fmt.Sprintf("emit/idiomatic: template %q: %v", name, err))
	}
}
