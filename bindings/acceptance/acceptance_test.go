//go:build darwin

// Package acceptance_test validates the code generators against the real
// framework metadata checked into the repository. These tests load pre-scanned
// .gometa.json files from the metadata directory (or the path given by
// GO_BINDINGS_METADATA_DIR), run the purego frameworks pipeline — the pipeline
// that generates bindings/internal/raw/frameworks — and verify the output is
// syntactically valid Go with the expected shape.
package acceptance_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pipeline "github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/pipeline"
)

const (
	// defaultMetaDir is the metadata directory relative to this package.
	// Tests run from bindings/acceptance/, so step up two levels to the repo root.
	defaultMetaDir = "../../metadata"

	frameworksModPrefix = "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/frameworks"
	librariesModPrefix  = "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries"
)

// metaDir returns the metadata directory, honouring GO_BINDINGS_METADATA_DIR.
func metaDir() string {
	if dir := os.Getenv("GO_BINDINGS_METADATA_DIR"); dir != "" {
		return dir
	}
	return defaultMetaDir
}

// requireFrameworkMeta returns the metadata directory for one framework, or
// skips when the corpus is not present (e.g. a checkout without metadata/).
func requireFrameworkMeta(t *testing.T, framework string) string {
	t.Helper()
	dir := filepath.Join(metaDir(), "frameworks", strings.ToLower(framework))
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("metadata for %s not found at %s: %v", framework, dir, err)
	}
	return dir
}

// generateFrameworks loads the given metadata dirs and emits raw framework
// packages into a temp dir, returning that dir.
func generateFrameworks(t *testing.T, metaPaths ...string) string {
	t.Helper()
	reg, err := pipeline.LoadAll(metaPaths, frameworksModPrefix, librariesModPrefix)
	if err != nil {
		t.Fatalf("LoadAll(%v): %v", metaPaths, err)
	}
	if len(reg.Frameworks) == 0 {
		t.Fatalf("LoadAll(%v) returned zero frameworks", metaPaths)
	}
	outDir := t.TempDir()
	if err := pipeline.GenerateBindings(pipeline.BindingsConfig{
		Registry:         reg,
		FrameworksOutDir: outDir,
	}); err != nil {
		t.Fatalf("GenerateBindings: %v", err)
	}
	return outDir
}

// assertPackageParses fails if any .go file in pkgDir is not valid Go, and
// returns the number of files checked.
func assertPackageParses(t *testing.T, pkgDir string) int {
	t.Helper()
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("reading %s: %v", pkgDir, err)
	}
	fset := token.NewFileSet()
	n := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(pkgDir, e.Name())
		if _, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution); err != nil {
			t.Errorf("%s is not valid Go: %v", e.Name(), err)
		}
		n++
	}
	if n == 0 {
		t.Errorf("no Go files generated in %s", pkgDir)
	}
	return n
}

// readGenerated returns the contents of one generated file, failing if absent.
func readGenerated(t *testing.T, pkgDir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(pkgDir, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(data)
}

// TestFoundationGeneratesValidGo generates Foundation from committed metadata
// and checks every emitted file parses as Go.
func TestFoundationGeneratesValidGo(t *testing.T) {
	fwDir := requireFrameworkMeta(t, "Foundation")
	pkgDir := filepath.Join(generateFrameworks(t, fwDir), "foundation")

	n := assertPackageParses(t, pkgDir)
	t.Logf("foundation: %d generated files parsed", n)

	// The runtime file is what makes a purego package work: it dlopens the
	// framework and registers its symbols. There must be no cgo anywhere.
	runtimeSrc := readGenerated(t, pkgDir, "foundation_runtime.go")
	if !strings.Contains(runtimeSrc, "SymbolAvailable") {
		t.Errorf("runtime file should expose the SymbolAvailable probe; got:\n%s", runtimeSrc)
	}
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("reading %s: %v", pkgDir, err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".m") || strings.HasSuffix(e.Name(), ".h") {
			t.Errorf("purego package must contain no ObjC bridge sources; found %s", e.Name())
		}
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if strings.Contains(readGenerated(t, pkgDir, e.Name()), "import \"C\"") {
			t.Errorf("%s must not use cgo", e.Name())
		}
	}
}

// TestGenericClassGeneration verifies NSArray keeps its ObjC generic as a Go
// type parameter, constrained by the runtime's object interface.
func TestGenericClassGeneration(t *testing.T) {
	fwDir := requireFrameworkMeta(t, "Foundation")
	pkgDir := filepath.Join(generateFrameworks(t, fwDir), "foundation")

	content := readGenerated(t, pkgDir, "NSArray.go")
	if !strings.Contains(content, "type NSArray[") {
		t.Errorf("NSArray.go: expected a generic type declaration; got:\n%s", content)
	}
	if !strings.Contains(content, "purego.AnyObject]") {
		t.Errorf("NSArray.go: expected the purego.AnyObject constraint; got:\n%s", content)
	}
}

// TestEnumGeneration verifies named enums come through as Go constants with a
// named underlying type.
func TestEnumGeneration(t *testing.T) {
	fwDir := requireFrameworkMeta(t, "Foundation")
	pkgDir := filepath.Join(generateFrameworks(t, fwDir), "foundation")

	content := readGenerated(t, pkgDir, "foundation_enums.go")
	for _, want := range []string{"type NSComparisonResult ", "NSOrderedAscending"} {
		if !strings.Contains(content, want) {
			t.Errorf("enums file missing %q", want)
		}
	}
}

// TestCrossFrameworkTypeResolution generates AppKit together with Foundation
// and verifies AppKit resolves Foundation types by importing that package
// rather than degrading them to unsafe.Pointer.
func TestCrossFrameworkTypeResolution(t *testing.T) {
	appKitDir := requireFrameworkMeta(t, "AppKit")
	foundationDir := requireFrameworkMeta(t, "Foundation")
	outDir := generateFrameworks(t, appKitDir, foundationDir)

	pkgDir := filepath.Join(outDir, "appkit")
	assertPackageParses(t, pkgDir)

	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("reading %s: %v", pkgDir, err)
	}
	found := false
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if strings.Contains(readGenerated(t, pkgDir, e.Name()), "raw/frameworks/foundation") {
			found = true
			break
		}
	}
	if !found {
		t.Error("appkit: expected at least one file importing the foundation package")
	}
}
