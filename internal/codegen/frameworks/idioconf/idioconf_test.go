package idioconf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
)

func writeSidecar(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	metaPath := filepath.Join(dir, "Foundation-arm64-26.5.gometa.json")
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return metaPath
}

func TestLoadAdjacentMissingFileIsNoOp(t *testing.T) {
	metaPath := filepath.Join(t.TempDir(), "Foundation-arm64-26.5.gometa.json")
	file, found, err := LoadAdjacent(metaPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found || file != nil {
		t.Fatalf("expected not found, got found=%v file=%v", found, file)
	}
}

func TestLoadAdjacentValidFile(t *testing.T) {
	metaPath := writeSidecar(t, `{
		"rename_methods": [
			{"class": "NSString", "selector": "stringWithContentsOfFile:encoding:error:", "class_method": true, "go_name": "NewStringFromFile"}
		],
		"rename_functions": [
			{"c_name": "hv_vm_create", "go_name": "CreateVM"}
		],
		"delegates": {"include": ["VZVirtualMachineDelegate"], "exclude": ["NSCopying"]},
		"error_typedefs": [
			{"typedef": "hv_return_t", "success_value": 0, "domain": "HypervisorReturnDomain", "sentinel_enum": "hv_error"}
		]
	}`)

	file, found, err := LoadAdjacent(metaPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected found")
	}

	if goName, ok := file.MethodGoName("NSString", "stringWithContentsOfFile:encoding:error:", true); !ok || goName != "NewStringFromFile" {
		t.Errorf("MethodGoName class-method match = %q, %v; want NewStringFromFile, true", goName, ok)
	}
	if _, ok := file.MethodGoName("NSString", "stringWithContentsOfFile:encoding:error:", false); ok {
		t.Error("MethodGoName matched an instance method despite class_method=true")
	}
	if _, ok := file.MethodGoName("NSString", "otherSelector", true); ok {
		t.Error("MethodGoName matched a different selector")
	}
	if goName, ok := file.FunctionGoName("hv_vm_create"); !ok || goName != "CreateVM" {
		t.Errorf("FunctionGoName = %q, %v; want CreateVM, true", goName, ok)
	}
	if !file.IsDelegateIncluded("VZVirtualMachineDelegate") {
		t.Error("IsDelegateIncluded = false; want true")
	}
	if !file.IsDelegateExcluded("NSCopying") {
		t.Error("IsDelegateExcluded = false; want true")
	}
	typedef, ok := file.ErrorTypedefFor("hv_return_t")
	if !ok || typedef.Domain != "HypervisorReturnDomain" || typedef.SentinelEnum != "hv_error" {
		t.Errorf("ErrorTypedefFor = %+v, %v", typedef, ok)
	}
}

func TestMethodRenameWithoutKindMatchesEither(t *testing.T) {
	metaPath := writeSidecar(t, `{
		"rename_methods": [{"class": "NSString", "selector": "length", "go_name": "Length"}]
	}`)
	file, _, err := LoadAdjacent(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := file.MethodGoName("NSString", "length", false); !ok {
		t.Error("instance-method lookup failed without class_method")
	}
	if _, ok := file.MethodGoName("NSString", "length", true); !ok {
		t.Error("class-method lookup failed without class_method")
	}
}

func TestLoadAdjacentRejectsMalformedEntries(t *testing.T) {
	cases := map[string]string{
		"missing go_name":     `{"rename_methods": [{"class": "NSString", "selector": "length"}]}`,
		"unexported go_name":  `{"rename_methods": [{"class": "NSString", "selector": "length", "go_name": "length"}]}`,
		"missing c_name":      `{"rename_functions": [{"go_name": "CreateVM"}]}`,
		"missing domain":      `{"error_typedefs": [{"typedef": "hv_return_t"}]}`,
		"unknown field":       `{"renamed_methods": []}`,
		"invalid JSON":        `{`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			metaPath := writeSidecar(t, content)
			if _, _, err := LoadAdjacent(metaPath); err == nil {
				t.Errorf("expected error for %s", name)
			}
		})
	}
}

func TestValidateWarnsOnStaleEntries(t *testing.T) {
	framework := &meta.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]meta.Class{
			"NSString": {Methods: []meta.Method{{Selector: "length"}}},
		},
		Functions: []meta.Function{{Name: "NSLog"}},
		Protocols: map[string]meta.Protocol{"NSCacheDelegate": {}},
		Typedefs:  map[string]string{"NSTimeInterval": "double"},
		Enums:     map[string]meta.Enum{"NSComparisonResult": {}},
	}

	file := &File{
		RenameMethods: []MethodRename{
			{Class: "NSString", Selector: "length", GoName: "Length"},       // valid
			{Class: "NSString", Selector: "gone:", GoName: "Gone"},          // stale selector
			{Class: "NSGone", Selector: "length", GoName: "Length"},         // stale class
		},
		RenameFunctions: []FunctionRename{
			{CName: "NSLog", GoName: "Log"},      // valid
			{CName: "NSGoneFunc", GoName: "Gone"}, // stale
		},
		Delegates: DelegateConfig{
			Include: []string{"NSCacheDelegate", "NSGoneDelegate"},
			Exclude: []string{"NSAlsoGone"},
		},
		ErrorTypedefs: []ErrorTypedef{
			{Typedef: "NSTimeInterval", Domain: "D"},                                // valid
			{Typedef: "gone_t", Domain: "D"},                                        // stale typedef
			{Typedef: "NSTimeInterval", Domain: "D", SentinelEnum: "missing_enum"},  // stale enum
		},
	}

	warnings := Validate(file, framework)
	wantFragments := []string{
		`no method gone: on NSString`,
		`no class "NSGone"`,
		`no function "NSGoneFunc"`,
		`no protocol "NSGoneDelegate"`,
		`no protocol "NSAlsoGone"`,
		`no typedef "gone_t"`,
		`no sentinel enum "missing_enum"`,
	}
	if len(warnings) != len(wantFragments) {
		t.Fatalf("got %d warnings, want %d:\n%s", len(warnings), len(wantFragments), strings.Join(warnings, "\n"))
	}
	for _, fragment := range wantFragments {
		found := false
		for _, warning := range warnings {
			if strings.Contains(warning, fragment) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no warning containing %q in:\n%s", fragment, strings.Join(warnings, "\n"))
		}
	}
}

func TestValidateNilFile(t *testing.T) {
	if warnings := Validate(nil, &meta.FrameworkMeta{}); warnings != nil {
		t.Errorf("Validate(nil) = %v; want nil", warnings)
	}
}

func TestNilReceiverHelpers(t *testing.T) {
	var file *File
	if _, ok := file.MethodGoName("A", "b", false); ok {
		t.Error("nil MethodGoName should miss")
	}
	if _, ok := file.FunctionGoName("f"); ok {
		t.Error("nil FunctionGoName should miss")
	}
	if file.IsDelegateIncluded("P") || file.IsDelegateExcluded("P") {
		t.Error("nil delegate lookups should be false")
	}
	if _, ok := file.ErrorTypedefFor("t"); ok {
		t.Error("nil ErrorTypedefFor should miss")
	}
}
