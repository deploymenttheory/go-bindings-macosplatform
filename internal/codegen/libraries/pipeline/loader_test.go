package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// writeMeta writes a FrameworkMeta to a temp subdirectory and returns the
// directory path that was written to.  The file is named per macosplatformmetadata.Write
// convention: <Framework>-<Arch>-<SDKVersion>.gometa.json.
func writeMeta(t *testing.T, dir string, framework *macosplatformmetadata.FrameworkMeta) string {
	t.Helper()
	if err := macosplatformmetadata.Write(framework, dir); err != nil {
		t.Fatalf("macosplatformmetadata.Write: %v", err)
	}
	return dir
}

// minimalFM returns a FrameworkMeta with sane defaults so macosplatformmetadata.Write succeeds.
func minimalFM(framework string) *macosplatformmetadata.FrameworkMeta {
	return &macosplatformmetadata.FrameworkMeta{
		Framework:  framework,
		SDKVersion: "15.0",
		Arch:       "arm64",
		Classes:    map[string]macosplatformmetadata.Class{},
		Protocols:  map[string]macosplatformmetadata.Protocol{},
		Enums:      map[string]macosplatformmetadata.Enum{},
		Structs:    map[string]macosplatformmetadata.Struct{},
		Functions:  []macosplatformmetadata.Function{},
		Externs:    []macosplatformmetadata.Extern{},
		BlockTypes: map[string]macosplatformmetadata.BlockType{},
		Typedefs:   map[string]string{},
	}
}

// ── TestLoadAllSingleFile ─────────────────────────────────────────────────────
func TestLoadAllSingleFile(t *testing.T) {
	dir := t.TempDir()
	fwDir := filepath.Join(dir, "Foundation")
	writeMeta(t, fwDir, minimalFM("Foundation"))

	reg, err := LoadAll([]string{dir}, "github.com/example/fw")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(reg.Frameworks) != 1 {
		t.Fatalf("expected 1 framework, got %d", len(reg.Frameworks))
	}
	if reg.Frameworks[0].Framework != "Foundation" {
		t.Errorf("expected Foundation, got %s", reg.Frameworks[0].Framework)
	}
}

// ── TestLoadAllMultipleFiles ──────────────────────────────────────────────────
func TestLoadAllMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	writeMeta(t, filepath.Join(dir, "Foundation"), minimalFM("Foundation"))
	writeMeta(t, filepath.Join(dir, "UIKit"), minimalFM("UIKit"))

	reg, err := LoadAll([]string{dir}, "github.com/example/fw")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(reg.Frameworks) != 2 {
		t.Fatalf("expected 2 frameworks, got %d", len(reg.Frameworks))
	}
	names := map[string]bool{}
	for _, framework := range reg.Frameworks {
		names[framework.Framework] = true
	}
	for _, want := range []string{"Foundation", "UIKit"} {
		if !names[want] {
			t.Errorf("expected framework %q in registry", want)
		}
	}
}

// ── TestLoadAllNoFiles ────────────────────────────────────────────────────────
func TestLoadAllNoFiles(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadAll([]string{dir}, "github.com/example/fw")
	if err == nil {
		t.Error("expected error for directory with no .gometa.json files, got nil")
	}
}

// ── TestLoadAllInvalidJSON ────────────────────────────────────────────────────
func TestLoadAllInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "broken.gometa.json")
	if err := os.WriteFile(bad, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadAll([]string{bad}, "github.com/example/fw")
	if err == nil {
		t.Error("expected error for malformed JSON file, got nil")
	}
}

// ── TestLoadAllKnownClasses ───────────────────────────────────────────────────
func TestLoadAllKnownClasses(t *testing.T) {
	dir := t.TempDir()
	framework := minimalFM("Foundation")
	framework.Classes["NSString"] = macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{{Selector: "length", Return: macosplatformmetadata.ReturnType{ObjCType: "NSUInteger"}}},
	}
	writeMeta(t, filepath.Join(dir, "Foundation"), framework)

	reg, err := LoadAll([]string{dir}, "github.com/example/fw")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if !reg.ClassNameIndex["NSString"] {
		t.Error("expected NSString in ClassNameIndex")
	}
}

// ── TestLoadAllGenericClasses ─────────────────────────────────────────────────
func TestLoadAllGenericClasses(t *testing.T) {
	dir := t.TempDir()
	framework := minimalFM("Foundation")
	framework.Classes["NSArray"] = macosplatformmetadata.Class{
		GenericParams: []string{"ObjectType"},
		Methods:       []macosplatformmetadata.Method{{Selector: "count", Return: macosplatformmetadata.ReturnType{ObjCType: "NSUInteger"}}},
	}
	writeMeta(t, filepath.Join(dir, "Foundation"), framework)

	reg, err := LoadAll([]string{dir}, "github.com/example/fw")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if !reg.GenericClasses["NSArray"] {
		t.Error("expected NSArray in GenericClasses (has GenericParams)")
	}
}

// ── TestLoadAllFrameworkOwnerFewestMethods ────────────────────────────────────
// The canonical owner of a class is the framework with the fewest
// methods+properties (most minimal = most canonical definition).
func TestLoadAllFrameworkOwnerFewestMethods(t *testing.T) {
	dir := t.TempDir()

	// Foundation declares NSString with 1 method (canonical).
	fmFoundation := minimalFM("Foundation")
	fmFoundation.Classes["NSString"] = macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{{Selector: "length", Return: macosplatformmetadata.ReturnType{ObjCType: "NSUInteger"}}},
	}
	writeMeta(t, filepath.Join(dir, "Foundation"), fmFoundation)

	// UIKit re-declares NSString with 3 methods (superset from header inclusion).
	fmUIKit := minimalFM("UIKit")
	fmUIKit.Classes["NSString"] = macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{
			{Selector: "length", Return: macosplatformmetadata.ReturnType{ObjCType: "NSUInteger"}},
			{Selector: "description", Return: macosplatformmetadata.ReturnType{ObjCType: "NSString *"}},
			{Selector: "hash", Return: macosplatformmetadata.ReturnType{ObjCType: "NSUInteger"}},
		},
	}
	writeMeta(t, filepath.Join(dir, "UIKit"), fmUIKit)

	reg, err := LoadAll([]string{dir}, "github.com/example/fw")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if got := reg.OwnerIndex["NSString"]; got != "Foundation" {
		t.Errorf("expected OwnerIndex[NSString]=Foundation, got %q", got)
	}
}

// ── TestLoadAllGhostsFallBack ─────────────────────────────────────────────────
// When all entries for a class are ghosts (0 methods+properties), the
// alphabetically first framework wins.
func TestLoadAllGhostsFallBack(t *testing.T) {
	dir := t.TempDir()

	// Both frameworks declare NSGhost with no methods (ghosts).
	fmA := minimalFM("Alpha")
	fmA.Classes["NSGhost"] = macosplatformmetadata.Class{} // score 0
	writeMeta(t, filepath.Join(dir, "Alpha"), fmA)

	fmB := minimalFM("Beta")
	fmB.Classes["NSGhost"] = macosplatformmetadata.Class{} // score 0
	writeMeta(t, filepath.Join(dir, "Beta"), fmB)

	reg, err := LoadAll([]string{dir}, "github.com/example/fw")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	// Alphabetically first framework: "Alpha" < "Beta".
	if got := reg.OwnerIndex["NSGhost"]; got != "Alpha" {
		t.Errorf("expected OwnerIndex[NSGhost]=Alpha (all ghosts, alpha first), got %q", got)
	}
}

// ── TestLoadAllTopLevelBeatsSubFramework ──────────────────────────────────────
// When the same framework appears as both a top-level entry (ParentFramework=="")
// and a sub-framework (ParentFramework!=""), the top-level version wins.
// We write both files to separate directories and pass both directories to LoadAll
// so resolvePaths discovers them at the same scan depth.
func TestLoadAllTopLevelBeatsSubFramework(t *testing.T) {
	// Two separate scan roots: topDir for top-level fw, subDir for sub-framework.
	topDir := t.TempDir()
	subDir := t.TempDir()

	// Top-level: topDir/CoreGraphics/CoreGraphics-arm64-15.0.gometa.json
	// ParentFramework is "" (not set).
	topFM := minimalFM("CoreGraphics")
	topFM.Classes["CGPath"] = macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{{Selector: "isRect", Return: macosplatformmetadata.ReturnType{ObjCType: "BOOL"}}},
	}
	writeMeta(t, filepath.Join(topDir, "CoreGraphics"), topFM)

	// Sub-framework: subDir/AppServices/CoreGraphics/CoreGraphics-arm64-15.0.gometa.json
	// The loader infers ParentFramework="AppServices" from grandparent detection.
	// We also write AppServices at subDir level so the grandparent dir is recognised.
	parentFM := minimalFM("AppServices")
	writeMeta(t, filepath.Join(subDir, "AppServices"), parentFM)

	subFM := minimalFM("CoreGraphics")
	subFM.Classes["CGPath"] = macosplatformmetadata.Class{} // ghost
	writeMeta(t, filepath.Join(subDir, "AppServices", "CoreGraphics"), subFM)

	// Pass both scan roots: resolvePaths will find one-level-deep .gometa.json files
	// under each, which gives us both the top-level and sub-framework CoreGraphics.
	reg, err := LoadAll([]string{topDir, filepath.Join(subDir, "AppServices")}, "github.com/example/fw")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	// The top-level CoreGraphics should win: its entry has ParentFramework=="".
	var cg *macosplatformmetadata.FrameworkMeta
	for _, framework := range reg.Frameworks {
		if framework.Framework == "CoreGraphics" {
			cg = framework
			break
		}
	}
	if cg == nil {
		t.Fatal("CoreGraphics not found in registry")
	}
	if cg.ParentFramework != "" {
		t.Errorf("expected top-level CoreGraphics (ParentFramework empty), got ParentFramework=%q", cg.ParentFramework)
	}
}

// ── TestLoadAllModulePrefix ───────────────────────────────────────────────────
func TestLoadAllModulePrefix(t *testing.T) {
	dir := t.TempDir()
	writeMeta(t, filepath.Join(dir, "Foundation"), minimalFM("Foundation"))

	const prefix = "github.com/myorg/bindings/frameworks"
	reg, err := LoadAll([]string{dir}, prefix)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if reg.ModulePrefix != prefix {
		t.Errorf("expected ModulePrefix=%q, got %q", prefix, reg.ModulePrefix)
	}
}

// ── TestLoadAllParentFrameworkInferred ────────────────────────────────────────
// When a file sits at {parent}/{child}/file.gometa.json, the loader should
// infer ParentFramework from the grandparent directory.
// We pass the parent directory directly to LoadAll so both the parent framework
// file and the sub-framework file are discovered in a single-level glob.
func TestLoadAllParentFrameworkInferred(t *testing.T) {
	root := t.TempDir()
	carbonDir := filepath.Join(root, "Carbon")

	// Write the Carbon framework at root/Carbon/.
	carbonFM := minimalFM("Carbon")
	writeMeta(t, carbonDir, carbonFM)

	// Write HIToolbox one level deeper: root/Carbon/HIToolbox/.
	// When LoadAll is given carbonDir, it globs carbonDir/*/*.gometa.json and
	// finds HIToolbox. The grandparent of root/Carbon/HIToolbox is root/Carbon,
	// which appears in fwDirPaths as "Carbon" → ParentFramework inferred.
	hiDir := filepath.Join(carbonDir, "HIToolbox")
	hiMeta := minimalFM("HIToolbox")
	hiMeta.Classes["HIView"] = macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{{Selector: "getID", Return: macosplatformmetadata.ReturnType{ObjCType: "uint32"}}},
	}
	writeMeta(t, hiDir, hiMeta)

	// Pass carbonDir: flat glob finds Carbon-arm64-15.0.gometa.json,
	// sub glob finds HIToolbox/HIToolbox-arm64-15.0.gometa.json.
	reg, err := LoadAll([]string{carbonDir}, "github.com/example/fw")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	var hi *macosplatformmetadata.FrameworkMeta
	for _, framework := range reg.Frameworks {
		if framework.Framework == "HIToolbox" {
			hi = framework
			break
		}
	}
	if hi == nil {
		t.Fatal("HIToolbox not found in registry")
	}
	if hi.ParentFramework != "Carbon" {
		t.Errorf("expected HIToolbox.ParentFramework=Carbon, got %q", hi.ParentFramework)
	}
}

// ── TestLoadAllAllClassesPopulated ────────────────────────────────────────────
// ClassIndex must contain every canonical class definition.
func TestLoadAllAllClassesPopulated(t *testing.T) {
	dir := t.TempDir()
	framework := minimalFM("Foundation")
	framework.Classes["NSObject"] = macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{{Selector: "init", Return: macosplatformmetadata.ReturnType{IsInstancetype: true}}},
	}
	writeMeta(t, filepath.Join(dir, "Foundation"), framework)

	reg, err := LoadAll([]string{dir}, "github.com/example/fw")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if _, ok := reg.ClassIndex["NSObject"]; !ok {
		t.Error("expected NSObject in ClassIndex")
	}
}

// ── TestLoadAllDirectFilePath ─────────────────────────────────────────────────
// LoadAll must accept a direct path to a .gometa.json file (not a directory).
func TestLoadAllDirectFilePath(t *testing.T) {
	dir := t.TempDir()
	fwDir := filepath.Join(dir, "Foundation")
	writeMeta(t, fwDir, minimalFM("Foundation"))

	// Find the written file.
	entries, err := os.ReadDir(fwDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no files written by macosplatformmetadata.Write")
	}
	filePath := filepath.Join(fwDir, entries[0].Name())

	reg, err := LoadAll([]string{filePath}, "github.com/example/fw")
	if err != nil {
		t.Fatalf("LoadAll with direct file: %v", err)
	}
	if len(reg.Frameworks) != 1 {
		t.Fatalf("expected 1 framework from direct file path, got %d", len(reg.Frameworks))
	}
}

// ── TestLoadAllNonGhostBeatsGhost ─────────────────────────────────────────────
// A framework with non-zero methods should win over a ghost entry even if
// the ghost framework sorts earlier alphabetically.
func TestLoadAllNonGhostBeatsGhost(t *testing.T) {
	dir := t.TempDir()

	// "Alpha" has a ghost entry for NSWidget (score 0).
	fmA := minimalFM("Alpha")
	fmA.Classes["NSWidget"] = macosplatformmetadata.Class{} // ghost
	writeMeta(t, filepath.Join(dir, "Alpha"), fmA)

	// "Beta" has a real entry for NSWidget (score 1).
	fmB := minimalFM("Beta")
	fmB.Classes["NSWidget"] = macosplatformmetadata.Class{
		Methods: []macosplatformmetadata.Method{{Selector: "draw", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}}},
	}
	writeMeta(t, filepath.Join(dir, "Beta"), fmB)

	reg, err := LoadAll([]string{dir}, "github.com/example/fw")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if got := reg.OwnerIndex["NSWidget"]; got != "Beta" {
		t.Errorf("expected OwnerIndex[NSWidget]=Beta (non-ghost wins), got %q", got)
	}
}
