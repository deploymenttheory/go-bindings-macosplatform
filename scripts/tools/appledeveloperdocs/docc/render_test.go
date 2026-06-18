package docc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadFixture reads a checked-in DocC render-JSON sample.
func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return data
}

func TestParseObjCAbstract(t *testing.T) {
	node, err := ParseObjC(loadFixture(t, "vzvirtualmachine.json"))
	if err != nil {
		t.Fatalf("ParseObjC: %v", err)
	}

	if node.Metadata.Title != "VZVirtualMachine" {
		t.Errorf("title = %q, want VZVirtualMachine", node.Metadata.Title)
	}
	want := "An object that manages the overall state and configuration of your VM."
	if got := node.Abstract(); got != want {
		t.Errorf("abstract = %q, want %q", got, want)
	}
}

// TestParseObjCProjectsSelectors is the core invariant: applying the occ
// variantOverrides patch rewrites Swift-named references into ObjC selectors.
func TestParseObjCProjectsSelectors(t *testing.T) {
	node, err := ParseObjC(loadFixture(t, "vzvirtualmachine.json"))
	if err != nil {
		t.Fatalf("ParseObjC: %v", err)
	}

	// Collect the ObjC titles of references that are direct children of
	// VZVirtualMachine and look like methods.
	methodTitles := map[string]Reference{}
	for _, ref := range node.References {
		if !strings.Contains(ref.Identifier, "/Virtualization/VZVirtualMachine/") {
			continue
		}
		if ref.HasMethodFragment() {
			methodTitles[ref.Title] = ref
		}
	}

	for _, want := range []string{
		"startWithCompletionHandler:",
		"startWithOptions:completionHandler:",
		"stopWithCompletionHandler:",
		"pauseWithCompletionHandler:",
	} {
		ref, ok := methodTitles[want]
		if !ok {
			t.Errorf("missing ObjC selector %q after projection", want)
			continue
		}
		if ref.IsClassMethod() {
			t.Errorf("%q reported as class method, want instance", want)
		}
		if ref.AbstractText(node.References) == "" {
			t.Errorf("%q has empty abstract", want)
		}
	}
}

func TestFlattenInlineResolvesReferences(t *testing.T) {
	refs := map[string]Reference{
		"doc://x": {Title: "NSString"},
	}
	content := []Inline{
		{Type: "text", Text: "Returns a "},
		{Type: "reference", Identifier: "doc://x"},
		{Type: "codeVoice", Code: " value"},
		{Type: "emphasis", InlineContent: []Inline{{Type: "text", Text: " now"}}},
	}
	if got, want := flattenInline(content, refs), "Returns a NSString value now"; got != want {
		t.Errorf("flattenInline = %q, want %q", got, want)
	}
}

func TestParseObjCNoOverrides(t *testing.T) {
	// A document without variantOverrides should parse unchanged.
	data := []byte(`{"metadata":{"title":"X"},"abstract":[{"type":"text","text":"hi"}]}`)
	node, err := ParseObjC(data)
	if err != nil {
		t.Fatalf("ParseObjC: %v", err)
	}
	if node.Abstract() != "hi" {
		t.Errorf("abstract = %q, want hi", node.Abstract())
	}
}
