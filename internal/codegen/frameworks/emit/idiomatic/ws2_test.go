package idiomatic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// vzFile reads a generated Virtualization wrapper file from the committed
// idiomatic tree, skipping the test when it is absent (so the check never blocks
// a checkout that has not been regenerated).
func vzFile(t *testing.T, name string) string {
	t.Helper()
	// from internal/codegen/frameworks/emit/idiomatic up to the repo root.
	path := filepath.Join("..", "..", "..", "..", "..",
		"opinionated", "idiomatic", "framework", "virtualization", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("generated file %s not present (%v); run `generate idiomatic`", name, err)
	}
	return string(data)
}

// TestWS2EmbeddingAndSealing locks the WS2 contract on the VZBootLoader family:
// the abstract base embeds objref.Handle and defines the sealing marker; each
// concrete subclass embeds the base (not objref.Handle) and asserts it satisfies
// the sealed provider; the provider interface requires the unexported marker.
func TestWS2EmbeddingAndSealing(t *testing.T) {
	base := vzFile(t, "VZBootLoader_generated.go")
	if !strings.Contains(base, "type BootLoader struct {\n\tobjref.Handle\n}") {
		t.Error("BootLoader (root base) must embed objref.Handle")
	}
	// WS3: an abstract base emits no constructor (you build a concrete subclass).
	if strings.Contains(base, "func NewBootLoader") {
		t.Error("BootLoader is an abstract base; it must not emit a New constructor (WS3)")
	}
	if !strings.Contains(base, "func (x *BootLoader) isBootLoader() {}") {
		t.Error("BootLoader must define the sealing marker isBootLoader")
	}
	if !strings.Contains(base, "var _ BootLoaderProvider = (*BootLoader)(nil)") {
		t.Error("BootLoader must assert it satisfies BootLoaderProvider")
	}

	for _, sub := range []string{"VZMacOSBootLoader_generated.go", "VZLinuxBootLoader_generated.go", "VZEFIBootLoader_generated.go"} {
		src := vzFile(t, sub)
		name := strings.TrimSuffix(strings.TrimPrefix(sub, "VZ"), "_generated.go")
		if !strings.Contains(src, "type "+name+" struct {\n\tBootLoader\n}") {
			t.Errorf("%s must embed BootLoader (not objref.Handle)", name)
		}
		if strings.Contains(src, "func (x *"+name+") isBootLoader() {}") {
			t.Errorf("%s must not redefine the marker; it is promoted from BootLoader", name)
		}
		if !strings.Contains(src, "var _ BootLoaderProvider = (*"+name+")(nil)") {
			t.Errorf("%s must assert it satisfies BootLoaderProvider", name)
		}
	}

	providers := vzFile(t, "virtualization_providers_generated.go")
	if !strings.Contains(providers, "isBootLoader()") {
		t.Error("BootLoaderProvider must require the unexported isBootLoader marker (the seal)")
	}
}

// TestWS4Docs locks the WS4 documentation: the package doc.go carries a grouped
// type index with godoc links, and a class doc is name-first (E2) with a Usage
// paragraph that cross-links the hierarchy.
func TestWS4Docs(t *testing.T) {
	doc := vzFile(t, "doc.go")
	if !strings.Contains(doc, "# Types") {
		t.Error("doc.go must carry a # Types index section")
	}
	if !strings.Contains(doc, "BootLoader: [EFIBootLoader], [LinuxBootLoader], [MacOSBootLoader]") {
		t.Error("doc.go type index must group the BootLoader base to its concrete subtypes as godoc links")
	}

	base := vzFile(t, "VZBootLoader_generated.go")
	if !strings.Contains(base, "// BootLoader is an idiomatic wrapper") {
		t.Error("class doc must start with the type name (E2)")
	}
	if !strings.Contains(base, "abstract base") || !strings.Contains(base, "[MacOSBootLoader]") {
		t.Error("abstract base doc must state it is abstract and link its concrete subtypes")
	}

	sub := vzFile(t, "VZMacOSBootLoader_generated.go")
	if !strings.Contains(sub, "It embeds [BootLoader]") {
		t.Error("subclass doc must link the base it embeds")
	}
}
