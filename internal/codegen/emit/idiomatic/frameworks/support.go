//go:build darwin

package idiomatic

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit"
)

// supportFS embeds the static source of the idiomatic layer's support packages.
// They carry no per-framework data, so they are stored verbatim (as .txt to
// keep them out of this package's own compilation) and written unchanged on
// every generation — making the whole idiomatic tree regenerable from scratch:
// delete opinionated/idiomatic and a single `generate idiomatic` run restores
// objref, errkit, and rt byte-for-byte.
//
//go:embed support/*.txt
var supportFS embed.FS

// supportFile maps an embedded payload to its destination, relative to the
// idiomatic output root (cfg.OutDir, canonically <repo>/opinionated/idiomatic).
type supportFile struct {
	src string // name under support/
	rel string // destination relative to the idiomatic root
}

var supportFiles = []supportFile{
	{src: "objref.txt", rel: "internal/objref/objref_generated.go"},
	{src: "rt.txt", rel: "rt/rt_generated.go"},
	{src: "errkit.txt", rel: "errkit/errkit_generated.go"},
	{src: "obj.txt", rel: "obj/obj_generated.go"},
	{src: "dispatch.txt", rel: "internal/dispatch/dispatch_generated.go"},
}

// EmitSupportPackages writes the idiomatic layer's hand-independent support
// packages (objref, errkit, rt) under rootDir. It is idempotent and
// deterministic: it only writes the known support files (never cleans a
// directory), so co-located hand-authored or generated framework packages are
// untouched. Call once per generation, before the per-framework loop.
func EmitSupportPackages(rootDir string) error {
	for _, f := range supportFiles {
		content, err := supportFS.ReadFile("support/" + f.src)
		if err != nil {
			return fmt.Errorf("read embedded support %s: %w", f.src, err)
		}
		dst := filepath.Join(rootDir, filepath.FromSlash(f.rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("mkdir for %s: %w", f.rel, err)
		}
		if err := emit.WriteGoFile(dst, content); err != nil {
			return fmt.Errorf("write support %s: %w", f.rel, err)
		}
	}
	return nil
}
