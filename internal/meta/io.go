package meta

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Schema-window sentinel errors, matchable with errors.Is by callers and by
// the purecg reader (which mirrors this check on its own model).
var (
	ErrSchemaTooNew = errors.New("metadata schema version is newer than this generator supports")
	ErrSchemaTooOld = errors.New("metadata schema version predates the minimum supported version")
)

// Write serialises framework to outDir/<framework>-<arch>-<sdk>.gometa.json.
// The file is stamped with CurrentSchemaVersion so Read can detect metadata
// written by an incompatible generator generation.
func Write(framework *FrameworkMeta, outDir string) error {
	framework.SchemaVersion = CurrentSchemaVersion
	NormaliseEnumGoTypes(framework)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	name := fmt.Sprintf("%s-%s-%s.gometa.json",
		framework.Framework, framework.Arch, framework.SDKVersion)
	path := filepath.Join(outDir, name)

	data, err := json.MarshalIndent(framework, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling %s: %w", name, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// Read deserialises a .gometa.json file.
func Read(path string) (*FrameworkMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var framework FrameworkMeta
	if err := json.Unmarshal(data, &framework); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if framework.SchemaVersion > CurrentSchemaVersion {
		return nil, fmt.Errorf(
			"%s: %w (file %d, supported %d) — update the generator",
			path,
			ErrSchemaTooNew,
			framework.SchemaVersion,
			CurrentSchemaVersion,
		)
	}
	if framework.SchemaVersion < MinSupportedSchemaVersion {
		return nil, fmt.Errorf(
			"%s: %w (file %d, minimum %d) — re-scan required (go run ./cmd/generate/ scan --framework %s)",
			path,
			ErrSchemaTooOld,
			framework.SchemaVersion,
			MinSupportedSchemaVersion,
			framework.Framework,
		)
	}
	NormaliseEnumGoTypes(&framework)
	return &framework, nil
}
