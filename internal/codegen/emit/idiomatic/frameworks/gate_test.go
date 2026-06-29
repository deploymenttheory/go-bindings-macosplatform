//go:build darwin

package idiomatic

import (
	"os"
	"path/filepath"
	"testing"
)

// TestVerifyHermetic checks that the hermetic gate flags a generated file that
// imports the raw bindings and passes a clean package.
func TestVerifyHermetic(t *testing.T) {
	dir := t.TempDir()

	clean := `//go:build darwin
package foundation
import "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/obj"
func use() obj.Object { return nil }
`
	if err := os.WriteFile(filepath.Join(dir, "clean_generated.go"), []byte(clean), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyHermetic(dir); err != nil {
		t.Fatalf("clean package should pass the hermetic gate: %v", err)
	}

	raw := `//go:build darwin
package foundation
import raw "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
func use() *raw.NSString { return nil }
`
	if err := os.WriteFile(filepath.Join(dir, "leaky_generated.go"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyHermetic(dir); err == nil {
		t.Fatal("a package importing the raw bindings must fail the hermetic gate")
	}
}
