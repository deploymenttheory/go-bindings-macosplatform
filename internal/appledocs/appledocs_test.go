package appledocs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

func TestMethodKey(t *testing.T) {
	if got := MethodKey("start:", false); got != "-start:" {
		t.Errorf("instance MethodKey = %q, want -start:", got)
	}
	if got := MethodKey("alloc", true); got != "+alloc" {
		t.Errorf("class MethodKey = %q, want +alloc", got)
	}
}

func sampleDocs() *Docs {
	return &Docs{
		SchemaVersion: SchemaVersion,
		Framework:     "Virtualization",
		Classes: map[string]SymbolDoc{
			"VZVirtualMachine": {Doc: "Apple class doc."},
		},
		Methods: map[string]map[string]SymbolDoc{
			"VZVirtualMachine": {
				"-startWithCompletionHandler:": {Doc: "Apple start doc."},
				"+isSupported":                 {Doc: "Apple class-method doc."},
			},
		},
		Properties: map[string]map[string]SymbolDoc{
			"VZVirtualMachine": {"state": {Doc: "Apple state doc."}},
		},
		Enums: map[string]EnumDoc{
			"VZErrorCode": {
				Doc:     "Apple enum doc.",
				Members: map[string]SymbolDoc{"VZErrorInternal": {Doc: "Apple member doc."}},
			},
		},
		Functions: map[string]SymbolDoc{"VZIsSupported": {Doc: "Apple func doc."}},
	}
}

func sampleFramework() *macosplatformmetadata.FrameworkMeta {
	return &macosplatformmetadata.FrameworkMeta{
		Framework: "Virtualization",
		Classes: map[string]macosplatformmetadata.Class{
			"VZVirtualMachine": {
				Doc: "header class doc",
				Methods: []macosplatformmetadata.Method{
					{Selector: "startWithCompletionHandler:"},
					{Selector: "isSupported", IsClassMethod: true},
					{Selector: "pauseWithCompletionHandler:", Doc: "header pause doc"},
				},
				Properties: []macosplatformmetadata.Property{
					{Name: "state"},
					{Name: "canStart", Doc: "header canStart doc"},
				},
			},
		},
		Enums: map[string]macosplatformmetadata.Enum{
			"VZErrorCode": {
				Members: []macosplatformmetadata.EnumMember{
					{Name: "VZErrorInternal"},
					{Name: "VZErrorOther", Doc: "header other doc"},
				},
			},
		},
		Functions: []macosplatformmetadata.Function{
			{Name: "VZIsSupported"},
			{Name: "VZUntouched", Doc: "header untouched doc"},
		},
	}
}

func TestApplyApplePreferredHeaderFallback(t *testing.T) {
	fw := sampleFramework()
	Apply(sampleDocs(), fw)

	cls := fw.Classes["VZVirtualMachine"]
	if cls.Doc != "Apple class doc." {
		t.Errorf("class doc = %q, want Apple class doc.", cls.Doc)
	}
	if got := cls.Methods[0].Doc; got != "Apple start doc." {
		t.Errorf("instance method doc = %q, want Apple start doc.", got)
	}
	if got := cls.Methods[1].Doc; got != "Apple class-method doc." {
		t.Errorf("class method doc = %q, want Apple class-method doc.", got)
	}
	// No Apple doc for pause → header comment preserved (fallback).
	if got := cls.Methods[2].Doc; got != "header pause doc" {
		t.Errorf("pause doc = %q, want header pause doc (fallback)", got)
	}
	if got := cls.Properties[0].Doc; got != "Apple state doc." {
		t.Errorf("property doc = %q, want Apple state doc.", got)
	}
	if got := cls.Properties[1].Doc; got != "header canStart doc" {
		t.Errorf("canStart doc = %q, want header fallback", got)
	}

	enum := fw.Enums["VZErrorCode"]
	if enum.Doc != "Apple enum doc." {
		t.Errorf("enum doc = %q, want Apple enum doc.", enum.Doc)
	}
	if got := enum.Members[0].Doc; got != "Apple member doc." {
		t.Errorf("enum member doc = %q, want Apple member doc.", got)
	}
	if got := enum.Members[1].Doc; got != "header other doc" {
		t.Errorf("enum member fallback = %q, want header other doc", got)
	}

	if got := fw.Functions[0].Doc; got != "Apple func doc." {
		t.Errorf("function doc = %q, want Apple func doc.", got)
	}
	if got := fw.Functions[1].Doc; got != "header untouched doc" {
		t.Errorf("function fallback = %q, want header untouched doc", got)
	}
}

func TestLoadAdjacentMissing(t *testing.T) {
	dir := t.TempDir()
	metaPath := filepath.Join(dir, "X-arm64-26.5.gometa.json")
	_, found, err := LoadAdjacent(metaPath)
	if err != nil {
		t.Fatalf("LoadAdjacent: %v", err)
	}
	if found {
		t.Error("found = true, want false for missing sidecar")
	}
}

func TestLoadAdjacentRoundTrip(t *testing.T) {
	dir := t.TempDir()
	metaPath := filepath.Join(dir, "X-arm64-26.5.gometa.json")
	data, _ := json.Marshal(sampleDocs())
	if err := os.WriteFile(filepath.Join(dir, FileName), data, 0o644); err != nil {
		t.Fatal(err)
	}
	docs, found, err := LoadAdjacent(metaPath)
	if err != nil || !found {
		t.Fatalf("LoadAdjacent found=%v err=%v", found, err)
	}
	if docs.Classes["VZVirtualMachine"].Doc != "Apple class doc." {
		t.Errorf("round-trip class doc = %q", docs.Classes["VZVirtualMachine"].Doc)
	}
}
