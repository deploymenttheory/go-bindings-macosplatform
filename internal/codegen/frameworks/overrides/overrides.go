// Package overrides applies the shared declarative override schema
// (internal/overrides) to the purego generator's metadata model
// (internal/codegen/frameworks/meta). The file format, discovery convention, and
// semantics are identical to internal/overrides; this package preserves the
// framework pipeline import path while using the canonical metadata model.
package overrides

import (
	"fmt"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	rootoverrides "github.com/deploymenttheory/go-bindings-macosplatform/internal/overrides"
)

// ApplyAdjacent looks for an override file next to metaPath and applies it to
// framework. A missing file is not an error. Returned warnings list override
// entries that matched nothing — stale after an SDK re-scan.
func ApplyAdjacent(metaPath string, framework *meta.FrameworkMeta) ([]string, error) {
	file, found, err := rootoverrides.LoadAdjacent(metaPath)
	if err != nil {
		return nil, fmt.Errorf("loading overrides for %s: %w", framework.Framework, err)
	}
	if !found {
		return nil, nil
	}
	return Apply(file, framework), nil
}

// Apply delegates to the shared implementation: the framework metadata types
// alias the canonical model, so both loaders must apply identical corrections.
func Apply(file *rootoverrides.File, framework *meta.FrameworkMeta) []string {
	return rootoverrides.Apply(file, framework)
}
