//go:build darwin

package idiofw

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	rawfw "github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks"
)

// supportFS embeds the static source of the idiomatic layer's support packages.
// They carry no per-framework data, so they are stored verbatim (as .txt to
// keep them out of this package's own compilation) and written unchanged on
// every generation — making the whole idiomatic tree regenerable from scratch:
// delete the generated tree and a single `generate idiomatic` run restores
// objref, errkit, and rt byte-for-byte.
//
//go:embed support/*.txt
var supportFS embed.FS

// supportFile maps an embedded payload to its destination, relative to the
// idiomatic output root (cfg.OutDir, canonically <repo>/bindings).
type supportFile struct {
	src string // name under support/
	rel string // destination relative to the idiomatic root
}

// The destinations are relative to the bindings root (the parent of the
// idiomatic framework output dir): private support packages under internal/,
// public runtime helpers under runtime/.
var supportFiles = []supportFile{
	{src: "objref.txt", rel: "internal/objref/objref_generated.go"},
	{src: "rt.txt", rel: "runtime/rt/rt_generated.go"},
	{src: "errkit.txt", rel: "runtime/errkit/errkit_generated.go"},
	{src: "obj.txt", rel: "runtime/obj/obj_generated.go"},
	{src: "dispatch.txt", rel: "internal/dispatch/dispatch_generated.go"},
	{src: "shim.txt", rel: "internal/shim/shim_generated.go"},
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
		if err := rawfw.WriteGoFile(dst, content); err != nil {
			return fmt.Errorf("write support %s: %w", f.rel, err)
		}
	}
	return nil
}
