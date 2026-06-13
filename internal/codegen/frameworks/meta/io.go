package meta

import (
	"encoding/json"
	"fmt"
	"os"

	rootmeta "github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// Read deserialises a .gometa.json file. The schema-version window is defined
// by internal/meta (the package that writes these files); referencing its
// constants here keeps the two readers from drifting.
func Read(path string) (*FrameworkMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var framework FrameworkMeta
	if err := json.Unmarshal(data, &framework); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if framework.SchemaVersion > rootmeta.CurrentSchemaVersion {
		return nil, fmt.Errorf(
			"%s: %w (file %d, supported %d) — update the generator",
			path,
			rootmeta.ErrSchemaTooNew,
			framework.SchemaVersion,
			rootmeta.CurrentSchemaVersion,
		)
	}
	if framework.SchemaVersion < rootmeta.MinSupportedSchemaVersion {
		return nil, fmt.Errorf(
			"%s: %w (file %d, minimum %d) — re-scan required (go run ./cmd/generate/ scan --framework %s)",
			path,
			rootmeta.ErrSchemaTooOld,
			framework.SchemaVersion,
			rootmeta.MinSupportedSchemaVersion,
			framework.Framework,
		)
	}
	return &framework, nil
}
