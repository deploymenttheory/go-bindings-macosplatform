package pipeline

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// firstFileIn returns the path of the first file found in dir.
func firstFileIn(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir %s: %v", dir, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			return filepath.Join(dir, e.Name())
		}
	}
	t.Fatalf("no file found in %s", dir)
	return ""
}

// --------------------------------------------------------------------------
// LoadAll — resolvePaths error propagation (loader.go:45)
// --------------------------------------------------------------------------

func TestLoadAllResolvePathsError(t *testing.T) {
	_, err := LoadAll([]string{"/nonexistent/pipeline/test/path"}, "github.com/example/fw")
	if err == nil {
		t.Error("LoadAll should return error for nonexistent path")
	}
}

// --------------------------------------------------------------------------
// LoadAll — ParentFramework already set skip (loader.go:77) AND
//           deduplication top-level replaces sub-framework (loader.go:101)
// --------------------------------------------------------------------------

// TestLoadAllParentFWSkipAndDedup exercises both the "ParentFramework already
// set → continue" early-return at loader.go:77-78 and the deduplication branch
// at loader.go:101-103 that replaces a sub-framework entry with a top-level one.
func TestLoadAllParentFWSkipAndDedup(t *testing.T) {
	dir := t.TempDir()

	// Sub-framework with ParentFramework already set — line 77-78 fires for this.
	subFM := minimalFM("CoreGraphics")
	subFM.ParentFramework = "Cocoa"
	subDir := filepath.Join(dir, "sub")
	writeMeta(t, subDir, subFM)
	subFile := firstFileIn(t, subDir)

	// Top-level with no parent — when encountered after subFM, line 101-103 fires.
	topFM := minimalFM("CoreGraphics")
	topDir := filepath.Join(dir, "top")
	writeMeta(t, topDir, topFM)
	topFile := firstFileIn(t, topDir)

	// Pass sub before top so that byName["CoreGraphics"] is first set to the
	// sub-framework entry, then replaced by the top-level entry (curIsTop &&
	// !prevIsTop).
	reg, err := LoadAll([]string{subFile, topFile}, "github.com/example/fw")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	// The registry should contain exactly one CoreGraphics framework, and it
	// must be the top-level one (ParentFramework == "").
	var found *meta.FrameworkMeta
	for _, framework := range reg.Frameworks {
		if framework.Framework == "CoreGraphics" {
			found = framework
		}
	}
	if found == nil {
		t.Fatal("CoreGraphics not found in registry")
	}
	if found.ParentFramework != "" {
		t.Errorf("expected top-level CoreGraphics (ParentFramework empty), got %q", found.ParentFramework)
	}
}

// --------------------------------------------------------------------------
// LoadAll — winner selection: second candidate beats first (loader.go:170)
// --------------------------------------------------------------------------

// TestLoadAllWinnerSelectionLowerScore verifies that when multiple frameworks
// define the same class, the one with fewer methods (lower score) wins even
// if it appears later in the iteration order.
func TestLoadAllWinnerSelectionLowerScore(t *testing.T) {
	dir := t.TempDir()

	// Zebra has 3 methods — high score (less canonical).
	zebraFM := minimalFM("Zebra")
	zebraFM.Classes["NSWidget"] = meta.Class{
		Methods: []meta.Method{
			{Selector: "a", Return: meta.ReturnType{ObjCType: "void"}},
			{Selector: "b", Return: meta.ReturnType{ObjCType: "void"}},
			{Selector: "c", Return: meta.ReturnType{ObjCType: "void"}},
		},
	}
	zebraDir := filepath.Join(dir, "Zebra")
	writeMeta(t, zebraDir, zebraFM)
	zebraFile := firstFileIn(t, zebraDir)

	// Alpha has 1 method — low score (more canonical). Passed second so that
	// Zebra is winner=candidates[0] and Alpha's lower score triggers the update
	// at loader.go:170-172.
	alphaFM := minimalFM("Alpha")
	alphaFM.Classes["NSWidget"] = meta.Class{
		Methods: []meta.Method{
			{Selector: "draw", Return: meta.ReturnType{ObjCType: "void"}},
		},
	}
	alphaDir := filepath.Join(dir, "Alpha")
	writeMeta(t, alphaDir, alphaFM)
	alphaFile := firstFileIn(t, alphaDir)

	// Zebra first → winner starts as {Zebra, 3}; Alpha second → Alpha wins (1 < 3).
	reg, err := LoadAll([]string{zebraFile, alphaFile}, "github.com/example/fw")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if got := reg.OwnerIndex["NSWidget"]; got != "Alpha" {
		t.Errorf("expected Alpha to win (fewer methods), got %q", got)
	}
}

// --------------------------------------------------------------------------
// sortFrameworksByDependency — cycle fallback (generator.go:125)
// --------------------------------------------------------------------------

// TestTopoSortCycleFallback creates a mutual superclass dependency so both
// frameworks have non-zero in-degree and are never added by Kahn's algorithm.
// The fallback loop at generator.go:124-127 must append them.
func TestTopoSortCycleFallback(t *testing.T) {
	fmA := &meta.FrameworkMeta{
		Framework: "FrameworkA",
		Classes:   map[string]meta.Class{"ClassA": {Super: "ClassB"}},
	}
	fmB := &meta.FrameworkMeta{
		Framework: "FrameworkB",
		Classes:   map[string]meta.Class{"ClassB": {Super: "ClassA"}},
	}
	reg := &Registry{
		Frameworks:     []*meta.FrameworkMeta{fmA, fmB},
		ClassNameIndex:   map[string]bool{"ClassA": true, "ClassB": true},
		GenericClasses: map[string]bool{},
		OwnerIndex: map[string]string{"ClassA": "FrameworkA", "ClassB": "FrameworkB"},
		ClassIndex:     map[string]meta.Class{},
		ModulePrefix:   "",
	}

	sorted := sortFrameworksByDependency(reg)
	if len(sorted) != 2 {
		t.Fatalf("expected 2 frameworks from topoSort with cycle, got %d", len(sorted))
	}
	names := map[string]bool{}
	for _, framework := range sorted {
		names[framework.Framework] = true
	}
	if !names["FrameworkA"] || !names["FrameworkB"] {
		t.Errorf("both frameworks should be present via fallback; got %v", names)
	}
}

// --------------------------------------------------------------------------
// resolveBlockedImports — class method args (generator.go:395) and
//                          protocol method args (generator.go:403)
// --------------------------------------------------------------------------

// TestComputeBlockedImportsMethodArgs verifies that addWeights is called for
// each method argument in both classes (line 395-397) and protocols (403-405).
func TestComputeBlockedImportsMethodArgs(t *testing.T) {
	fmAlpha := &meta.FrameworkMeta{
		Framework: "Alpha",
		Classes: map[string]meta.Class{
			"AlphaObj": {Methods: []meta.Method{{Selector: "run", Return: meta.ReturnType{}}}},
		},
		Protocols: map[string]meta.Protocol{},
	}
	fmBeta := &meta.FrameworkMeta{
		Framework: "Beta",
		// Class method with AlphaObj * arg → covers class arg addWeights (line 395-397).
		Classes: map[string]meta.Class{
			"BetaObj": {Methods: []meta.Method{{
				Selector: "doWith",
				Params:     []meta.Param{{Name: "obj", ObjCType: "AlphaObj *"}},
				Return:   meta.ReturnType{},
			}}},
		},
		// Protocol method with AlphaObj * arg → covers protocol arg addWeights (line 403-405).
		Protocols: map[string]meta.Protocol{
			"BetaProto": {Methods: []meta.Method{{
				Selector: "handle",
				Params:     []meta.Param{{Name: "obj", ObjCType: "AlphaObj *"}},
				Return:   meta.ReturnType{},
			}}},
		},
	}
	reg := &Registry{
		Frameworks:     []*meta.FrameworkMeta{fmAlpha, fmBeta},
		OwnerIndex: map[string]string{"AlphaObj": "Alpha"},
		ClassNameIndex:   map[string]bool{"AlphaObj": true},
		GenericClasses: map[string]bool{},
		ClassIndex:     map[string]meta.Class{},
		ModulePrefix:   "",
	}
	blocked := resolveBlockedImports(reg)
	// No cycle (Alpha←Beta only), so nothing should be blocked.
	if len(blocked) != 0 {
		t.Errorf("expected no blocked imports, got %v", blocked)
	}
}

// --------------------------------------------------------------------------
// resolveBlockedImports — multi-edge removal in cycle break (generator.go:471)
// --------------------------------------------------------------------------

// TestComputeBlockedImportsCycleMultiEdge creates a cycle (Alpha↔Beta) where
// Alpha also has an edge to Gamma. When the Alpha→Beta edge is removed as the
// minimum-weight edge, the inner loop at generator.go:470-474 must process
// both Beta (skip) and Gamma (keep), covering line 471-473.
func TestComputeBlockedImportsCycleMultiEdge(t *testing.T) {
	// Alpha → Beta (weight 1, return type), Alpha → Gamma (weight 1, return type).
	// Beta → Alpha (weight 2, two return types). Cycle: Alpha↔Beta.
	// weight[Alpha][Beta]=1 < weight[Beta][Alpha]=2 → minSrc=Alpha, minDst=Beta.
	// adj[Alpha]=[Beta, Gamma]; inner loop appends Gamma → covers line 471-473.
	fmAlpha := &meta.FrameworkMeta{
		Framework: "Alpha",
		Classes: map[string]meta.Class{
			"AlphaObj": {Methods: []meta.Method{{
				Selector: "get",
				Return:   meta.ReturnType{ObjCType: "BetaObj *"}, // weight[Alpha][Beta]++
			}}},
			"AlphaObj2": {Methods: []meta.Method{{
				Selector: "create",
				Return:   meta.ReturnType{ObjCType: "GammaObj *"}, // weight[Alpha][Gamma]++
			}}},
		},
		Protocols: map[string]meta.Protocol{},
	}
	fmBeta := &meta.FrameworkMeta{
		Framework: "Beta",
		Classes: map[string]meta.Class{
			"BetaObj": {Methods: []meta.Method{
				{Selector: "first", Return: meta.ReturnType{ObjCType: "AlphaObj *"}},  // weight[Beta][Alpha]++
				{Selector: "second", Return: meta.ReturnType{ObjCType: "AlphaObj2 *"}}, // weight[Beta][Alpha]++
			}},
		},
		Protocols: map[string]meta.Protocol{},
	}
	fmGamma := &meta.FrameworkMeta{
		Framework: "Gamma",
		Classes:   map[string]meta.Class{"GammaObj": {}},
		Protocols: map[string]meta.Protocol{},
	}
	reg := &Registry{
		Frameworks: []*meta.FrameworkMeta{fmAlpha, fmBeta, fmGamma},
		OwnerIndex: map[string]string{
			"AlphaObj":  "Alpha",
			"AlphaObj2": "Alpha",
			"BetaObj":   "Beta",
			"GammaObj":  "Gamma",
		},
		ClassNameIndex:   map[string]bool{},
		GenericClasses: map[string]bool{},
		ClassIndex:     map[string]meta.Class{},
		ModulePrefix:   "",
	}
	blocked := resolveBlockedImports(reg)
	// The Alpha→Beta edge should be blocked (lower weight = fewer references).
	if !blocked["Alpha"]["Beta"] {
		t.Errorf("expected Alpha→Beta to be blocked; got %v", blocked)
	}
}

// --------------------------------------------------------------------------
// writeFile — os.WriteFile error path (generator.go:324)
// --------------------------------------------------------------------------

// TestWriteFileOsWriteFileError verifies that writeFile returns an error when
// the target path cannot be written (e.g. because it is a directory).
func TestWriteFileOsWriteFileError(t *testing.T) {
	dir := t.TempDir()
	// Place a directory at the target file path so os.WriteFile fails.
	targetPath := filepath.Join(dir, "test.go")
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		t.Fatalf("setup: MkdirAll %s: %v", targetPath, err)
	}

	err := writeFile(targetPath, func(buf *bytes.Buffer) error {
		buf.WriteString("package test\n")
		return nil
	})
	if err == nil {
		t.Error("expected error when target path is a directory, got nil")
	}
}

// --------------------------------------------------------------------------
// emitFramework — MkdirAll error (generator.go:156) propagated by
//                     Generate (generator.go:42)
// --------------------------------------------------------------------------

// TestGenerateFrameworkMkdirAllError verifies that emitFramework returns
// an error when the framework output directory cannot be created, and that
// Generate propagates that error to the caller.
func TestGenerateFrameworkMkdirAllError(t *testing.T) {
	dir := t.TempDir()

	// Place a regular file at the path that would be the framework output dir
	// so os.MkdirAll fails (cannot create a directory over a file).
	fwDirPath := filepath.Join(dir, "foundation")
	if err := os.WriteFile(fwDirPath, []byte("placeholder"), 0o644); err != nil {
		t.Fatalf("setup WriteFile: %v", err)
	}

	reg := buildMinimalRegistry("Foundation", map[string]meta.Class{})
	cfg := BindingsConfig{Registry: reg, FrameworksOutDir: dir}
	if err := GenerateBindings(cfg); err == nil {
		t.Error("expected error from Generate when framework outDir is a file, got nil")
	}
}

// --------------------------------------------------------------------------
// resolveBlockedImports — multi-hop uppercase typedef chain (generator.go)
// --------------------------------------------------------------------------

// TestComputeBlockedImportsUppercaseTypedefChain verifies that addMethodWeight
// expands uppercase typedef names (not just lowercase ones) and follows
// multi-hop chains when detecting cross-framework import edges.
//
// Chain: FrameworkC uses AliasForBType (typedef in FrameworkA) → BType *
// (owned by FrameworkB). Prior to the fix, only lowercase typedefs were
// expanded, so the FrameworkC→FrameworkB edge was silently missed, producing
// incorrect cycle analysis when FrameworkB→FrameworkC was also present.
func TestComputeBlockedImportsUppercaseTypedefChain(t *testing.T) {
	// FrameworkA defines: typedef BType * AliasForBType;
	// FrameworkB owns BType.
	// FrameworkC has a method that takes AliasForBType.
	// FrameworkB→FrameworkC: BType has a method returning CType *.
	// Without uppercase typedef expansion: only FrameworkC→FrameworkA edge
	// is visible; the FrameworkC→FrameworkB transitive edge is missed, so the
	// cycle FrameworkC↔FrameworkB is not detected.
	fmA := &meta.FrameworkMeta{
		Framework: "FrameworkA",
		Classes:   map[string]meta.Class{},
		Protocols: map[string]meta.Protocol{},
	}
	fmB := &meta.FrameworkMeta{
		Framework: "FrameworkB",
		Classes: map[string]meta.Class{
			"BType": {Methods: []meta.Method{{
				Selector: "toC",
				Return:   meta.ReturnType{ObjCType: "CType *"}, // FrameworkB → FrameworkC
			}}},
		},
		Protocols: map[string]meta.Protocol{},
	}
	fmC := &meta.FrameworkMeta{
		Framework: "FrameworkC",
		Classes: map[string]meta.Class{
			"CType": {Methods: []meta.Method{{
				Selector: "doThing",
				// AliasForBType is an uppercase typedef for "BType *" owned by FrameworkA.
				// With multi-depth expansion, this creates FrameworkC→FrameworkB too.
				Params: []meta.Param{{Name: "x", ObjCType: "AliasForBType"}},
			}}},
		},
		Protocols: map[string]meta.Protocol{},
	}

	reg := &Registry{
		Frameworks: []*meta.FrameworkMeta{fmA, fmB, fmC},
		OwnerIndex: map[string]string{
			"BType": "FrameworkB",
			"CType": "FrameworkC",
		},
		ClassNameIndex:   map[string]bool{"BType": true, "CType": true},
		GenericClasses: map[string]bool{},
		ClassIndex:     map[string]meta.Class{},
		EnumIndex:     map[string]string{},
		EnumGoTypeIndex: map[string]string{},
		TypedefIndex: map[string]string{
			"AliasForBType": "BType *", // uppercase typedef → pointer to foreign class
		},
		TypedefOwnerIndex: map[string]string{
			"AliasForBType": "FrameworkA",
		},
		ProtocolIndex: map[string]string{},
		StructIndex:   map[string]string{},
		ModulePrefix:   "",
	}

	blocked := resolveBlockedImports(reg)

	// The graph has a cycle: FrameworkB→FrameworkC (return type CType) and
	// FrameworkC→FrameworkB (via AliasForBType → BType *). One of these edges
	// must be blocked.
	bToC := blocked["FrameworkB"]["FrameworkC"]
	cToB := blocked["FrameworkC"]["FrameworkB"]
	if !bToC && !cToB {
		t.Errorf("expected cycle between FrameworkB and FrameworkC to be broken; blocked=%v", blocked)
	}
}
