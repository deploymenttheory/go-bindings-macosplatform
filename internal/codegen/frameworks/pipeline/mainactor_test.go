package pipeline

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestMainActorPropagation is an integration check that the committed mainactor
// sidecars load and propagate @MainActor isolation down the class hierarchy:
// an explicit root (NSView), a same-framework subclass (NSButton), a
// cross-framework subclass (MKMapView), and a queue-based class that must stay
// off the main thread (VZVirtualMachine).
func TestMainActorPropagation(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	metaDir := filepath.Join(repo, "metadata")

	reg, err := LoadAll([]string{
		filepath.Join(metaDir, "frameworks"),
		filepath.Join(metaDir, "libraries"),
	}, "x/frameworks", "x/libraries")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	lookup := func(name string) (bool, bool) {
		for _, fw := range reg.Frameworks {
			if cls, ok := fw.Classes[name]; ok {
				return cls.IsMainThreadRequired, true
			}
		}
		return false, false
	}

	want := map[string]bool{
		"NSView":           true,  // explicit root
		"NSButton":         true,  // subclass via NSControl→NSView
		"MKMapView":        true,  // cross-framework subclass of NSView
		"VZVirtualMachine": false, // queue-based, not main-thread
	}
	for name, exp := range want {
		got, found := lookup(name)
		if !found {
			t.Logf("class %s not found (skipping)", name)
			continue
		}
		if got != exp {
			t.Errorf("%s: IsMainThreadRequired=%v want %v", name, got, exp)
		} else {
			t.Logf("%s: IsMainThreadRequired=%v (ok)", name, got)
		}
	}
}
