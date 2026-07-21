//go:build darwin

package idiofw

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// TestEmitSupportPackages verifies the idiomatic support packages (objref,
// errkit, rt) are emitted deterministically from embedded payloads: a single
// call recreates every file, and each is syntactically valid Go. This guards
// the "regenerable from scratch, nothing loseable" invariant.
func TestEmitSupportPackages(t *testing.T) {
	root := t.TempDir()
	if err := EmitSupportPackages(root); err != nil {
		t.Fatalf("EmitSupportPackages: %v", err)
	}

	want := []string{
		"internal/objref/objref_generated.go",
		"runtime/rt/rt_generated.go",
		"runtime/errkit/errkit_generated.go",
		"runtime/obj/obj_generated.go",
		"internal/dispatch/dispatch_generated.go",
	}
	fset := token.NewFileSet()
	for _, rel := range want {
		path := filepath.Join(root, filepath.FromSlash(rel))
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected emitted file %s: %v", rel, err)
		}
		if len(src) == 0 {
			t.Fatalf("emitted file %s is empty", rel)
		}
		if _, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution); err != nil {
			t.Fatalf("emitted file %s does not parse: %v", rel, err)
		}
	}

	// Idempotent: a second emission produces byte-identical output.
	first, _ := os.ReadFile(filepath.Join(root, "runtime/rt/rt_generated.go"))
	if err := EmitSupportPackages(root); err != nil {
		t.Fatalf("second EmitSupportPackages: %v", err)
	}
	second, _ := os.ReadFile(filepath.Join(root, "runtime/rt/rt_generated.go"))
	if string(first) != string(second) {
		t.Fatal("EmitSupportPackages is not deterministic across runs")
	}
}
