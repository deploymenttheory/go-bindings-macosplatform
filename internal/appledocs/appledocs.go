// Package appledocs applies Apple Developer documentation to scanned metadata
// at load time, mirroring the overrides package.
//
// The prose comes from Apple's DocC render API, harvested by the
// scripts/tools/appledeveloperdocs tool into a sidecar file that sits next to
// the .gometa.json it enriches (metadata/frameworks/<name>/appledocs.json).
// Like overrides, it is applied after macosplatformmetadata.Read so the
// committed .gometa.json stays pure scanned Clang output and a re-scan never
// clobbers the web docs.
//
// Merge policy is Apple-preferred with header fallback: a non-empty Apple
// abstract overwrites the sparse header doc comment captured by the scanner;
// when Apple has nothing, the header comment is left in place.
package appledocs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// FileName is the sidecar file name expected next to a framework's
// .gometa.json (e.g. metadata/frameworks/foundation/appledocs.json).
const FileName = "appledocs.json"

// SchemaVersion is the current generation of the sidecar schema.
const SchemaVersion = 1

// Docs is one framework's Apple-documentation sidecar. Keys are the exact
// metadata identities: class/enum/struct/protocol names, function/extern names,
// the ±selector method key (see MethodKey), property names, and enum member
// names.
type Docs struct {
	SchemaVersion int    `json:"schema_version"`
	Framework     string `json:"framework"`
	Source        string `json:"source,omitempty"`
	FetchedAt     string `json:"fetched_at,omitempty"`

	Classes   map[string]SymbolDoc `json:"classes,omitempty"`
	Protocols map[string]SymbolDoc `json:"protocols,omitempty"`
	Enums     map[string]EnumDoc   `json:"enums,omitempty"`
	Structs   map[string]SymbolDoc `json:"structs,omitempty"`
	Functions map[string]SymbolDoc `json:"functions,omitempty"`
	Externs   map[string]SymbolDoc `json:"externs,omitempty"`

	// Methods is keyed by class name, then by the ±selector key from MethodKey.
	Methods map[string]map[string]SymbolDoc `json:"methods,omitempty"`
	// Properties is keyed by class name, then by property name.
	Properties map[string]map[string]SymbolDoc `json:"properties,omitempty"`
}

// SymbolDoc is the documentation for one symbol.
type SymbolDoc struct {
	Doc string `json:"doc,omitempty"`
	URL string `json:"url,omitempty"`
}

// EnumDoc is the documentation for an enum and its members.
type EnumDoc struct {
	Doc     string               `json:"doc,omitempty"`
	URL     string               `json:"url,omitempty"`
	Members map[string]SymbolDoc `json:"members,omitempty"`
}

// MethodKey builds the method map key: "-selector" for an instance method,
// "+selector" for a class method. This matches the ±selector convention used
// across the codegen naming layer so instance and class methods that share a
// selector never collide.
func MethodKey(selector string, isClassMethod bool) string {
	if isClassMethod {
		return "+" + selector
	}
	return "-" + selector
}

// LoadAdjacent loads the sidecar file that sits next to metaPath. found is
// false when no sidecar exists — not an error.
func LoadAdjacent(metaPath string) (docs *Docs, found bool, err error) {
	path := filepath.Join(filepath.Dir(metaPath), FileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("reading appledocs %s: %w", path, err)
	}
	var parsed Docs
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, false, fmt.Errorf("parsing appledocs %s: %w", path, err)
	}
	return &parsed, true, nil
}

// ApplyAdjacent looks for a sidecar next to metaPath and applies it to
// framework. A missing sidecar is not an error.
func ApplyAdjacent(metaPath string, framework *macosplatformmetadata.FrameworkMeta) error {
	docs, found, err := LoadAdjacent(metaPath)
	if err != nil || !found {
		return err
	}
	Apply(docs, framework)
	return nil
}

// Apply enriches framework's Doc fields from docs (Apple-preferred, header
// fallback). It mutates framework in place.
func Apply(docs *Docs, framework *macosplatformmetadata.FrameworkMeta) {
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
				key := MethodKey(cls.Methods[i].Selector, cls.Methods[i].IsClassMethod)
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
