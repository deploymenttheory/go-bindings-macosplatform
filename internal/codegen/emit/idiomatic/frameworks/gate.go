//go:build darwin

package idiofw

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// rawBindingsImportPath is the import path of the raw bindings. The generated
// idiomatic packages must never import it: they dispatch to Objective-C through
// the shared runtime instead.
const rawBindingsImportPath = "bindings/frameworks"

// verifyHermetic checks that no generated Go file in outDir imports the raw
// bindings. It is run after a package is emitted, turning any accidental
// dependency on the raw layer into a hard generation failure rather than a
// surprise at build time.
func verifyHermetic(outDir string) error {
	entries, err := os.ReadDir(outDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(outDir, e.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(src, []byte(rawBindingsImportPath)) {
			return fmt.Errorf(
				"hermetic check failed: %s references the raw bindings (%s); "+
					"the idiomatic layer must dispatch through the runtime instead",
				path, rawBindingsImportPath,
			)
		}
	}
	return nil
}
