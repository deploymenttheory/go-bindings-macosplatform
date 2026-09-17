//go:build darwin

package scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// SDK 27 ARKit forwards to private headers unless USE_ARKIT_PUBLIC_HEADERS is
// defined. Reproduce that header-selection contract without depending on ARKit
// or a particular installed SDK, and exercise both actual Clang passes.
func TestConfiguredDefinesReachASTAndLayouts(t *testing.T) {
	if _, err := exec.LookPath("xcrun"); err != nil {
		t.Skip("requires xcrun clang")
	}
	sdkPath := t.TempDir()
	header := FrameworkHeader(sdkPath, "Fixture")
	if err := os.MkdirAll(filepath.Dir(header), 0o755); err != nil {
		t.Fatal(err)
	}
	source := `
#if USE_PUBLIC_HEADERS == 1
struct PublicRecord { int count; char flag; unsigned short tag; } publicRecord;
#else
struct PrivateRecord { long value; } privateRecord;
#endif
void Redirect(void) __asm__("_Redirect_v2");
`
	if err := os.WriteFile(header, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	saved := scanConfigs
	t.Cleanup(func() { scanConfigs = saved })
	scanConfigs = map[string]ScanConfig{
		"Fixture": {Defines: []string{"USE_PUBLIC_HEADERS=1"}},
	}
	root, err := DumpAST(sdkPath, "Fixture", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, node := range root.Inner {
		if node.Kind == "RecordDecl" && node.Name == "PublicRecord" {
			found = true
		}
		if node.Name == "PrivateRecord" {
			t.Fatal("AST selected the private header branch")
		}
	}
	if !found {
		t.Fatal("AST omitted the public record")
	}
	metadata := Extract(root, sdkPath, "Fixture", "27.0", "arm64", nil)
	if len(metadata.Functions) != 1 || metadata.Functions[0].Name != "Redirect" || metadata.Functions[0].LinkSymbol() != "Redirect_v2" {
		t.Fatalf("assembly alias was not preserved: %+v", metadata.Functions)
	}
	layouts, err := DumpRecordLayouts(sdkPath, "Fixture", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	layout, ok := layouts["PublicRecord"]
	if !ok || layout.Size != 8 || len(layout.FieldOffsets) != 3 || layout.FieldOffsets[2] != 6 {
		t.Fatalf("public record layout = %+v, want size 8 and last field at offset 6", layout)
	}
	if _, ok := layouts["PrivateRecord"]; ok {
		t.Fatal("layout pass selected the private header branch")
	}
}
