package raw

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

// loadedTemplates returns the parsed template set, initialised once on first call.
// All .tmpl files under templates/ are embedded at build time, so ParseFS only
// fails if a template contains a syntax error — which is caught at test time.
func loadedTemplates() *template.Template {
	tmplOnce.Do(func() {
		var err error
		tmplSet, err = template.ParseFS(templateFS, "templates/*.tmpl")
		if err != nil {
			panic("emit/raw: failed to parse embedded templates: " + err.Error())
		}
	})
	return tmplSet
}

// executeTemplate executes the named template with data, writing the result to w.
func executeTemplate(w io.Writer, name string, data any) error {
	return loadedTemplates().ExecuteTemplate(w, name, data)
}
