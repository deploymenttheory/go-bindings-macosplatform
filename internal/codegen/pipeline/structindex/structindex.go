// Package structindex builds the struct-name → owning-framework registry shared
// by the frameworks (purego) and libraries (cgo) pipeline loaders. Both derived
// the same map with the same conflict-resolution and typedef-backfill rules; this
// is the single implementation.
//
// SCOPE — shared: the single struct-ownership index for both the frameworks
// (purego) and libraries (cgo) pipeline loaders.
package structindex

import (
	"strings"

	mpm "github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// Build populates structIndex (struct name → owning framework) from frameworks,
// then backfills clean typedef-alias names using typedefIndex. Both maps are
// mutated in place.
//
// Ownership rules:
//   - Only a struct with at least one field can own a name; opaque (zero-field)
//     C types must stay unsafe.Pointer, not resolve to a typed Go struct.
//   - A field-bearing definition replaces a prior zero-field placeholder.
//   - When two frameworks both define the struct with fields, the
//     lexicographically-smallest framework name wins, for deterministic output.
//
// Typedef backfill: for a struct stored under an underscore-prefixed tag (the C
// idiom `typedef struct _NSRange NSRange;`), the clean public typedef name is
// registered to the same owner, so the mapper resolves "NSRange" directly.
func Build(frameworks []*mpm.FrameworkMeta, structIndex, typedefIndex map[string]string) {
	for _, framework := range frameworks {
		for name, s := range framework.Structs {
			existingOwner, has := structIndex[name]
			if !has {
				if len(s.Fields) > 0 {
					structIndex[name] = framework.Framework
				}
				continue
			}
			if len(s.Fields) > 0 {
				prior := lookupStruct(frameworks, existingOwner, name)
				if prior == nil || len(prior.Fields) == 0 {
					structIndex[name] = framework.Framework
					continue
				}
				if framework.Framework < existingOwner {
					structIndex[name] = framework.Framework
				}
			}
		}
	}

	for name, owner := range structIndex {
		if !strings.HasPrefix(name, "_") {
			continue
		}
		target := "struct " + name
		for tName, tTarget := range typedefIndex {
			if tTarget == target && !strings.HasPrefix(tName, "_") {
				if _, already := structIndex[tName]; !already {
					structIndex[tName] = owner
				}
			}
		}
	}
}

// lookupStruct returns the struct definition for name in the framework fwName,
// or nil if that framework does not define it.
func lookupStruct(frameworks []*mpm.FrameworkMeta, fwName, name string) *mpm.Struct {
	for _, framework := range frameworks {
		if framework.Framework != fwName {
			continue
		}
		if s, ok := framework.Structs[name]; ok {
			return &s
		}
	}
	return nil
}
