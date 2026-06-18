// Package appledocs applies the shared Apple-documentation sidecar schema
// (internal/appledocs) to the purego generator's metadata model
// (internal/codegen/frameworks/meta). The file format, discovery convention,
// and merge policy are identical to internal/appledocs — only the mutated types
// differ, because the frameworks tree mirrors codegen with its own meta package
// (the same split the overrides packages use).
package appledocs

import (
	rootdocs "github.com/deploymenttheory/go-bindings-macosplatform/internal/appledocs"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
)

// ApplyAdjacent looks for a sidecar next to metaPath and applies it to
// framework. A missing sidecar is not an error.
func ApplyAdjacent(metaPath string, framework *meta.FrameworkMeta) error {
	docs, found, err := rootdocs.LoadAdjacent(metaPath)
	if err != nil || !found {
		return err
	}
	Apply(docs, framework)
	return nil
}

// Apply enriches framework's Doc fields from docs (Apple-preferred, header
// fallback). It mutates framework in place.
func Apply(docs *rootdocs.Docs, framework *meta.FrameworkMeta) {
	if docs == nil {
		return
	}

	for name, cls := range framework.Classes {
		changed := false
		if d, ok := docs.Classes[name]; ok && d.Doc != "" {
			cls.Doc = d.Doc
			changed = true
		}
		if methods := docs.Methods[name]; methods != nil {
			for i := range cls.Methods {
				key := rootdocs.MethodKey(cls.Methods[i].Selector, cls.Methods[i].IsClassMethod)
				if d, ok := methods[key]; ok && d.Doc != "" {
					cls.Methods[i].Doc = d.Doc
					changed = true
				}
			}
		}
		if props := docs.Properties[name]; props != nil {
			for i := range cls.Properties {
				if d, ok := props[cls.Properties[i].Name]; ok && d.Doc != "" {
					cls.Properties[i].Doc = d.Doc
					changed = true
				}
			}
		}
		if changed {
			framework.Classes[name] = cls
		}
	}

	for name, enum := range framework.Enums {
		changed := false
		if d, ok := docs.Enums[name]; ok {
			if d.Doc != "" {
				enum.Doc = d.Doc
				changed = true
			}
			for i := range enum.Members {
				if m, ok := d.Members[enum.Members[i].Name]; ok && m.Doc != "" {
					enum.Members[i].Doc = m.Doc
					changed = true
				}
			}
		}
		if changed {
			framework.Enums[name] = enum
		}
	}

	for name, s := range framework.Structs {
		if d, ok := docs.Structs[name]; ok && d.Doc != "" {
			s.Doc = d.Doc
			framework.Structs[name] = s
		}
	}

	for i := range framework.Functions {
		if d, ok := docs.Functions[framework.Functions[i].Name]; ok && d.Doc != "" {
			framework.Functions[i].Doc = d.Doc
		}
	}

	for i := range framework.Externs {
		if d, ok := docs.Externs[framework.Externs[i].Name]; ok && d.Doc != "" {
			framework.Externs[i].Doc = d.Doc
		}
	}
}
