package render

import (
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/idiomatic/view"
)

// join concatenates parts with sep. A thin wrapper over strings.Join so
// templates can build comma-separated lists without a pipeline of helpers.
func join(parts []string, sep string) string {
	return strings.Join(parts, sep)
}

// wrap applies a conversion template (one %s placeholder) to expr. The gather
// phase decides the template (for example "FooFromID(%s)" or "obj.Wrap(%s)");
// this helper only substitutes, so render makes no type decision.
func wrap(tmpl, expr string) string {
	if tmpl == "" {
		return expr
	}
	return strings.Replace(tmpl, "%s", expr, 1)
}

// retOutValues renders the success return list for a method that lifts value
// out-parameters: the method's own result (if any), then each out-parameter,
// then a trailing nil when the method also returns an error.
func retOutValues(d view.Dispatch) string {
	var vals []string
	switch d.RetKind {
	case view.RetString, view.RetObject:
		vals = append(vals, "_v")
	case view.RetScalar:
		vals = append(vals, "_r")
	}
	for _, o := range d.Outs {
		vals = append(vals, o.GoName)
	}
	if d.Error {
		vals = append(vals, "nil")
	}
	return strings.Join(vals, ", ")
}

// retOutZeros renders the error-path zero prefix for a method that lifts value
// out-parameters: the result's zero (if any) then each out-parameter's zero,
// ending in ", " so the caller can append the error expression. It returns "" if
// there is nothing to zero (so the error stands alone).
func retOutZeros(d view.Dispatch) string {
	var zeros []string
	switch d.RetKind {
	case view.RetString:
		zeros = append(zeros, `""`)
	case view.RetObject, view.RetScalar:
		zeros = append(zeros, d.RetZero)
	}
	for _, o := range d.Outs {
		zeros = append(zeros, o.Zero)
	}
	if len(zeros) == 0 {
		return ""
	}
	return strings.Join(zeros, ", ") + ", "
}
