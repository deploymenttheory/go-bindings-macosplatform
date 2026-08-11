//go:build darwin

// Package acceptance_test validates the full code-generation pipeline against
// real framework metadata checked into the repository. Tests load pre-scanned
// .gometa.json files from the metadata directory (or the path given by
// GO_BINDINGS_METADATA_DIR), run pipeline.Generate, and verify that the output
// is syntactically correct Go code.
package acceptance_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/pipeline"
)

const (
	// defaultMetaDir is the metadata directory relative to the repo root.
	// Tests are run from the acceptance/ package directory, so we step up one
	// level to reach the project root.
	defaultMetaDir   = "../metadata"
	defaultModPrefix = "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/frameworks"

	// Generated filename constants. Keeping them here means a rename only
	// requires a single edit rather than a grep-and-replace across test bodies.
	fileDoc               = "doc.go"
	fileCgo               = "cgo.go"
	fileNSObject          = "NSObject.go"
	fileAppkitEnums       = "appkit_enums.go"
	fileForeignExtensions = "applescriptobjc_foreign_extensions.go"
)

// metaDir returns the metadata directory to use for acceptance tests.
// The environment variable GO_BINDINGS_METADATA_DIR overrides the default.
func metaDir() string {
	if d := os.Getenv("GO_BINDINGS_METADATA_DIR"); d != "" {
		return d
	}
	return defaultMetaDir
}

// requireRepoRootMetaDir returns the metadata directory for a test that has
// already chdir'd to the repo root (GO_BINDINGS_METADATA_DIR still wins), and
// skips the test if it does not exist on disk.
func requireRepoRootMetaDir(t *testing.T) string {
	t.Helper()
	dir := "metadata"
	if d := os.Getenv("GO_BINDINGS_METADATA_DIR"); d != "" {
		dir = d
	}
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("metadata directory %q not found — set GO_BINDINGS_METADATA_DIR: %v", dir, err)
	}
	return dir
}

// requireMetaDir returns the metadata directory and skips the test if it does
// not exist on disk.
func requireMetaDir(t *testing.T) string {
	t.Helper()
	dir := metaDir()
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("metadata directory %q not found — set GO_BINDINGS_METADATA_DIR or run from the repo root: %v", dir, err)
	}
	return dir
}

// requireFrameworkMeta skips the test if the specified framework subdirectory
// is absent inside dir. The committed layout nests frameworks and C libraries
// one level down (metadata/frameworks/<name>/, metadata/libraries/<name>/);
// the flat <dir>/<name>/ form is still accepted for ad-hoc metadata dirs.
func requireFrameworkMeta(t *testing.T, dir, framework string) string {
	t.Helper()
	name := strings.ToLower(framework)
	candidates := []string{
		filepath.Join(dir, "frameworks", name),
		filepath.Join(dir, "libraries", name),
		filepath.Join(dir, name),
	}
	for _, fwDir := range candidates {
		if _, err := os.Stat(fwDir); err == nil {
			return fwDir
		}
	}
	t.Skipf("metadata for framework %q not found in %s (tried %v)", framework, dir, candidates)
	return ""
}

// loadRegistry is a helper that calls pipeline.LoadAll on a list of metadata
// paths (directories or files). It fails the test on any error.
func loadRegistry(t *testing.T, paths []string) *pipeline.Registry {
	t.Helper()
	reg, err := pipeline.LoadAll(paths, defaultModPrefix)
	if err != nil {
		t.Fatalf("pipeline.LoadAll(%v): %v", paths, err)
	}
	return reg
}

// generateTo runs pipeline.GenerateBindings into outDir using reg and fails the test on
// error.
func generateTo(t *testing.T, reg *pipeline.Registry, outDir string) {
	t.Helper()
	if err := pipeline.GenerateBindings(pipeline.BindingsConfig{
		Registry:         reg,
		FrameworksOutDir: outDir,
		LibrariesOutDir:  t.TempDir(),
	}); err != nil {
		t.Fatalf("pipeline.GenerateBindings: %v", err)
	}
}

// assertFileContains fails the test if no file under root (recursively) whose
// name matches filename contains the expected substring.
func assertFileContains(t *testing.T, root, filename, want string) {
	t.Helper()
	found := false
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if filepath.Base(path) != filename {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), want) {
			found = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	if !found {
		t.Errorf("expected %q to contain %q, but no matching file found under %s", filename, want, root)
	}
}

// assertAnyFileContains fails the test if no .go file under root (recursively)
// contains the expected substring.
func assertAnyFileContains(t *testing.T, root, want string) {
	t.Helper()
	found := false
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), want) {
			found = true
		}
		return nil
	})
	if !found {
		t.Errorf("no generated .go file under %s contains %q", root, want)
	}
}

// TestFoundationFrameworkGeneratesAndBuilds loads Foundation metadata, runs the
// generator, and verifies the output contains expected files and key type
// declarations.
func TestFoundationFrameworkGeneratesAndBuilds(t *testing.T) {
	base := requireMetaDir(t)
	fwDir := requireFrameworkMeta(t, base, "Foundation")

	reg := loadRegistry(t, []string{fwDir})
	outDir := t.TempDir()
	generateTo(t, reg, outDir)

	foundationOut := filepath.Join(outDir, "foundation")
	if _, err := os.Stat(foundationOut); err != nil {
		t.Fatalf("expected foundation output directory at %s: %v", foundationOut, err)
	}

	// doc.go must exist with the package declaration.
	docPath := filepath.Join(foundationOut, fileDoc)
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("reading doc.go: %v", err)
	}
	if !strings.Contains(string(data), "package foundation") {
		t.Errorf("doc.go missing package declaration; got:\n%s", string(data))
	}

	// NSObject.go must declare the NSObject struct.
	assertFileContains(t, foundationOut, fileNSObject, "type NSObject struct")

	// cgo.go must reference -framework Foundation.
	assertFileContains(t, foundationOut, fileCgo, "-framework Foundation")
}

// TestAppKitFrameworkGeneratesAndBuilds loads AppKit metadata (and Foundation
// so cross-framework types resolve) and verifies the generated output.
func TestAppKitFrameworkGeneratesAndBuilds(t *testing.T) {
	base := requireMetaDir(t)
	foundationDir := requireFrameworkMeta(t, base, "Foundation")
	appkitDir := requireFrameworkMeta(t, base, "AppKit")

	reg := loadRegistry(t, []string{foundationDir, appkitDir})
	outDir := t.TempDir()
	generateTo(t, reg, outDir)

	appkitOut := filepath.Join(outDir, "appkit")
	if _, err := os.Stat(appkitOut); err != nil {
		t.Fatalf("expected appkit output directory at %s: %v", appkitOut, err)
	}

	// cgo.go must link AppKit.
	assertFileContains(t, appkitOut, fileCgo, "-framework AppKit")

	// NSWindow.go must declare the NSWindow struct.
	assertFileContains(t, appkitOut, "NSWindow.go", "type NSWindow struct")

	// AppKit must have an enums file with a type declaration.
	enumPath := filepath.Join(appkitOut, fileAppkitEnums)
	if _, err := os.Stat(enumPath); err != nil {
		t.Fatalf("expected appkit_enums.go at %s: %v", enumPath, err)
	}
}

// TestCrossFrameworkTypeResolution loads Foundation and AppKit together and
// verifies that AppKit-generated files correctly embed Foundation types —
// specifically that classes with NSObject as their direct ObjC superclass embed
// foundation.NSObject rather than a local type.
func TestCrossFrameworkTypeResolution(t *testing.T) {
	base := requireMetaDir(t)
	foundationDir := requireFrameworkMeta(t, base, "Foundation")
	appkitDir := requireFrameworkMeta(t, base, "AppKit")

	reg := loadRegistry(t, []string{foundationDir, appkitDir})
	outDir := t.TempDir()
	generateTo(t, reg, outDir)

	appkitOut := filepath.Join(outDir, "appkit")

	// NSToolbar's ObjC superclass is NSObject (owned by Foundation). The
	// generated NSToolbar.go must embed foundation.NSObject, not a local NSObject.
	toolbarPath := filepath.Join(appkitOut, "NSToolbar.go")
	data, err := os.ReadFile(toolbarPath)
	if err != nil {
		t.Fatalf("reading NSToolbar.go: %v", err)
	}
	if !strings.Contains(string(data), "foundation.NSObject") {
		t.Errorf("NSToolbar.go: expected 'foundation.NSObject' embedding for cross-framework superclass; got:\n%s", string(data))
	}

	// The file must import the foundation package.
	if !strings.Contains(string(data), `"github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/frameworks/foundation"`) {
		t.Errorf("NSToolbar.go: expected foundation import; got:\n%s", string(data))
	}
}

// TestGenericClassGeneration loads Foundation and verifies that the generated
// NSArray.go uses Go generics syntax (a type parameter constrained to
// runtime.Object).
func TestGenericClassGeneration(t *testing.T) {
	base := requireMetaDir(t)
	fwDir := requireFrameworkMeta(t, base, "Foundation")

	reg := loadRegistry(t, []string{fwDir})
	outDir := t.TempDir()
	generateTo(t, reg, outDir)

	arrayPath := filepath.Join(outDir, "foundation", "NSArray.go")
	data, err := os.ReadFile(arrayPath)
	if err != nil {
		t.Fatalf("reading NSArray.go: %v", err)
	}

	// The struct must carry a type parameter. The constraint moved from the old
	// internal runtime package (runtime.Object) to the public bindings/runtime/cgo
	// package (cgo.Object) when the runtime was relocated.
	content := string(data)
	hasGeneric := strings.Contains(content, "[T cgo.Object]") ||
		strings.Contains(content, "[T runtime.Object]")
	if !hasGeneric {
		t.Errorf("NSArray.go: expected generic type parameter (e.g. '[T cgo.Object]'); got:\n%s", content)
	}

	// The struct declaration must reference the type parameter.
	if !strings.Contains(content, "type NSArray") {
		t.Errorf("NSArray.go: missing type NSArray declaration:\n%s", content)
	}
}

// TestEnumGeneration loads AppKit (which has hundreds of enums) and verifies
// that the generated enum file contains proper Go type declarations.
func TestEnumGeneration(t *testing.T) {
	base := requireMetaDir(t)
	appkitDir := requireFrameworkMeta(t, base, "AppKit")

	// Load Foundation too so type resolution works correctly.
	foundationDir := requireFrameworkMeta(t, base, "Foundation")
	reg := loadRegistry(t, []string{foundationDir, appkitDir})
	outDir := t.TempDir()
	generateTo(t, reg, outDir)

	enumPath := filepath.Join(outDir, "appkit", fileAppkitEnums)
	data, err := os.ReadFile(enumPath)
	if err != nil {
		t.Fatalf("reading appkit_enums.go: %v", err)
	}
	content := string(data)

	// The file must have at least one type declaration and one const block.
	if !strings.Contains(content, "type ") {
		t.Errorf("appkit_enums.go: no type declarations found:\n%s", content[:min(len(content), 500)])
	}
	if !strings.Contains(content, "const (") {
		t.Errorf("appkit_enums.go: no const blocks found:\n%s", content[:min(len(content), 500)])
	}

	// A well-known AppKit enum type must be present.
	if !strings.Contains(content, "NSWindowStyleMask") && !strings.Contains(content, "NSApplicationActivationPolicy") {
		t.Errorf("appkit_enums.go: expected at least one well-known AppKit enum type; got partial content:\n%s",
			content[:min(len(content), 500)])
	}
}

// TestForeignExtensionsGeneration loads AppleScriptObjC (which adds methods to
// NSBundle from Foundation) and verifies the _foreign_extensions.go file is
// generated (or is correctly absent when all receivers are blocked by import
// cycle prevention).
func TestForeignExtensionsGeneration(t *testing.T) {
	base := requireMetaDir(t)
	aoDir := requireFrameworkMeta(t, base, "applescriptobjc")
	foundationDir := requireFrameworkMeta(t, base, "Foundation")

	reg := loadRegistry(t, []string{foundationDir, aoDir})
	outDir := t.TempDir()
	generateTo(t, reg, outDir)

	aoOut := filepath.Join(outDir, "applescriptobjc")
	if _, err := os.Stat(aoOut); err != nil {
		t.Fatalf("expected applescriptobjc output directory at %s: %v", aoOut, err)
	}

	fePath := filepath.Join(aoOut, fileForeignExtensions)
	if _, err := os.Stat(fePath); err == nil {
		// File exists — verify it has meaningful content.
		data, err := os.ReadFile(fePath)
		if err != nil {
			t.Fatalf("reading foreign_extensions file: %v", err)
		}
		if !strings.Contains(string(data), "package applescriptobjc") {
			t.Errorf("foreign_extensions.go missing package declaration; content:\n%s", string(data))
		}
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat foreign_extensions.go: %v", err)
	}
	// If the file does not exist, all receivers were blocked by cycle prevention,
	// which is a legitimate outcome — the test passes either way.
}

// TestSubFrameworkLinkFlag loads HIToolbox (a sub-framework of Carbon) and
// verifies that cgo.go emits '-framework Carbon' (the parent) rather than
// '-framework HIToolbox'.
func TestSubFrameworkLinkFlag(t *testing.T) {
	base := requireMetaDir(t)
	carbonDir := filepath.Join(base, "carbon")
	hiDir := filepath.Join(carbonDir, "hitoolbox")
	if _, err := os.Stat(hiDir); err != nil {
		t.Skipf("HIToolbox sub-framework metadata not found at %s: %v", hiDir, err)
	}

	reg := loadRegistry(t, []string{carbonDir})
	outDir := t.TempDir()
	generateTo(t, reg, outDir)

	cgoPath := filepath.Join(outDir, "hitoolbox", fileCgo)
	data, err := os.ReadFile(cgoPath)
	if err != nil {
		t.Fatalf("reading hitoolbox/cgo.go: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "-framework Carbon") {
		t.Errorf("hitoolbox/cgo.go: expected '-framework Carbon' link flag for sub-framework; got:\n%s", content)
	}
	if strings.Contains(content, "-framework HIToolbox") {
		t.Errorf("hitoolbox/cgo.go: must NOT contain '-framework HIToolbox' (use parent '-framework Carbon'); got:\n%s", content)
	}
}

// TestAllFrameworksGenerateWithoutError loads every metadata file in the
// metadata directory and generates all framework packages. This is a smoke test
// that catches generator panics, crashes, or structural errors across the full
// corpus. The CGo bridge emitters read repo-relative shim headers
// (metadata/shims/…), so corpus generation must run from the repo root.
func TestAllFrameworksGenerateWithoutError(t *testing.T) {
	t.Chdir("..")
	base := requireRepoRootMetaDir(t)
	outDir := t.TempDir()

	reg, err := pipeline.LoadAll([]string{base}, defaultModPrefix)
	if err != nil {
		t.Fatalf("LoadAll(%s): %v", base, err)
	}
	if len(reg.Frameworks) == 0 {
		t.Fatal("LoadAll returned zero frameworks")
	}
	t.Logf("loaded %d frameworks", len(reg.Frameworks))

	if err := pipeline.GenerateBindings(pipeline.BindingsConfig{
		Registry:         reg,
		FrameworksOutDir: outDir,
		LibrariesOutDir:  t.TempDir(),
	}); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Verify at least one package directory was written.
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", outDir, err)
	}
	if len(entries) == 0 {
		t.Error("Generate produced no output directories")
	}
}

// TestAllFrameworksBuild generates all frameworks and runs 'go build ./...' on
// the output to confirm the generated code compiles. This test is intentionally
// slow and is skipped with -short.
func TestAllFrameworksBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow build test in -short mode")
	}

	// Check that the go tool is available.
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go binary not found in PATH: %v", err)
	}

	// Corpus generation reads repo-relative shim headers; run from the repo root.
	t.Chdir("..")
	base := requireRepoRootMetaDir(t)

	// We need a compilable module: write generated packages under a subdirectory
	// of the real project repo so go.mod is inherited automatically. The test
	// already chdir'd to the repo root above.
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("resolving repo root: %v", err)
	}

	// The generator writes packages to OutDir/<fwName>/, so OutDir must be set
	// to the "frameworks/" subdirectory of the module root so the generated
	// import paths (github.com/.../frameworks/<fwName>) resolve correctly.
	moduleRoot := t.TempDir()
	outDir := filepath.Join(moduleRoot, "bindings", "frameworks")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("creating output dir: %v", err)
	}

	reg, err := pipeline.LoadAll([]string{base}, defaultModPrefix)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	t.Logf("loaded %d frameworks; generating...", len(reg.Frameworks))

	if err := pipeline.GenerateBindings(pipeline.BindingsConfig{
		Registry:         reg,
		FrameworksOutDir: outDir,
		LibrariesOutDir:  t.TempDir(),
	}); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Copy go.mod and go.sum into the module root so go build can resolve
	// the module path.
	goModSrc := filepath.Join(repoRoot, "go.mod")
	goSumSrc := filepath.Join(repoRoot, "go.sum")
	for _, pair := range [][2]string{
		{goModSrc, filepath.Join(moduleRoot, "go.mod")},
		{goSumSrc, filepath.Join(moduleRoot, "go.sum")},
	} {
		data, readErr := os.ReadFile(pair[0])
		if readErr != nil {
			t.Skipf("reading %s: %v — cannot set up build environment", pair[0], readErr)
		}
		if writeErr := os.WriteFile(pair[1], data, 0o644); writeErr != nil {
			t.Fatalf("writing %s: %v", pair[1], writeErr)
		}
	}

	// Copy the runtime package into the module root so all imports resolve
	// within a single self-contained module (no replace directive needed).
	runtimeSrc := os.DirFS(filepath.Join(repoRoot, "runtime"))
	if err := os.CopyFS(filepath.Join(moduleRoot, "runtime"), runtimeSrc); err != nil {
		t.Fatalf("copying runtime: %v", err)
	}

	cmd := exec.Command(goBin, "build", "./...")
	cmd.Dir = moduleRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1", "GOOS=darwin")
	out, buildErr := cmd.CombinedOutput()
	if buildErr != nil {
		t.Fatalf("go build ./... failed:\n%s\nerror: %v", string(out), buildErr)
	}
	if len(out) > 0 {
		t.Logf("go build output:\n%s", string(out))
	}
}
