package overrides

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

func sampleFramework() *macosplatformmetadata.FrameworkMeta {
	return &macosplatformmetadata.FrameworkMeta{
		Framework:  "Foo",
		SDKVersion: "26.5",
		Arch:       "arm64",
		Classes: map[string]macosplatformmetadata.Class{
			"FooThing": {
				Methods: []macosplatformmetadata.Method{
					{Selector: "doIt:", Params: []macosplatformmetadata.Param{{Name: "value", ObjCType: "NSInteger"}},
						Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
					{Selector: "broken", Return: macosplatformmetadata.ReturnType{ObjCType: "FooRef"}},
					{Selector: "make", IsClassMethod: true, Return: macosplatformmetadata.ReturnType{ObjCType: "instancetype"}},
				},
			},
			"FooGone": {},
		},
		Enums: map[string]macosplatformmetadata.Enum{
			"FooOptions": {GoType: "uint64", Members: []macosplatformmetadata.EnumMember{{Name: "FooA", Value: "1"}}},
		},
		Functions: []macosplatformmetadata.Function{
			{Name: "FooCreate", Params: []macosplatformmetadata.Param{{Name: "flags", ObjCType: "int"}},
				Return: macosplatformmetadata.ReturnType{ObjCType: "void *"}},
			{Name: "FooDoomed", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}},
		},
	}
}

func TestExcludes(t *testing.T) {
	framework := sampleFramework()
	falseVal := false
	_ = falseVal
	warnings := Apply(&File{
		ExcludeClasses:   []string{"FooGone"},
		ExcludeMethods:   []MethodRef{{Class: "FooThing", Selector: "broken"}},
		ExcludeFunctions: []string{"FooDoomed"},
	}, framework)

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if _, ok := framework.Classes["FooGone"]; ok {
		t.Error("FooGone should be excluded")
	}
	for _, method := range framework.Classes["FooThing"].Methods {
		if method.Selector == "broken" {
			t.Error("broken method should be excluded")
		}
	}
	for _, function := range framework.Functions {
		if function.Name == "FooDoomed" {
			t.Error("FooDoomed should be excluded")
		}
	}
}

func TestExcludeMethodRespectsClassMethodFlag(t *testing.T) {
	framework := sampleFramework()
	// Selector "make" exists only as a class method; excluding the instance
	// variant must not match and must warn.
	warnings := Apply(&File{
		ExcludeMethods: []MethodRef{{Class: "FooThing", Selector: "make", IsClassMethod: false}},
	}, framework)
	if len(warnings) != 1 || !strings.Contains(warnings[0], "stale") {
		t.Errorf("expected one stale warning, got %v", warnings)
	}
	if len(framework.Classes["FooThing"].Methods) != 3 {
		t.Error("no method should have been removed")
	}
}

func TestRemapTypes(t *testing.T) {
	framework := sampleFramework()
	warnings := Apply(&File{
		RemapTypes: []TypeRemap{
			{Class: "FooThing", Selector: "doIt:", Param: "value", ObjCType: "FooOptions"},
			{Class: "FooThing", Selector: "broken", Param: "return", ObjCType: "void *"},
			{Function: "FooCreate", Param: "flags", ObjCType: "FooOptions"},
		},
	}, framework)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	methods := framework.Classes["FooThing"].Methods
	if methods[0].Params[0].ObjCType != "FooOptions" {
		t.Errorf("param remap not applied: %+v", methods[0].Params[0])
	}
	if methods[1].Return.ObjCType != "void *" {
		t.Errorf("return remap not applied: %+v", methods[1].Return)
	}
	if framework.Functions[0].Params[0].ObjCType != "FooOptions" {
		t.Errorf("function param remap not applied: %+v", framework.Functions[0].Params[0])
	}
}

func TestForceBitmaskAndAvailability(t *testing.T) {
	framework := sampleFramework()
	unavailable := true
	warnings := Apply(&File{
		ForceBitmaskEnums: []string{"FooOptions"},
		AvailabilityFixes: []AvailabilityFix{
			{Class: "FooThing", MacOSIntroduced: "11.0", MacOSDeprecated: "15.0"},
			{Enum: "FooOptions", IsUnavailable: &unavailable},
			{Function: "FooCreate", MacOSIntroduced: "12.0"},
		},
	}, framework)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if !framework.Enums["FooOptions"].IsBitmask {
		t.Error("FooOptions should be forced to bitmask")
	}
	if got := framework.Classes["FooThing"].Availability; got.MacOSIntroduced != "11.0" || got.MacOSDeprecated != "15.0" {
		t.Errorf("class availability fix not applied: %+v", got)
	}
	if !framework.Enums["FooOptions"].Availability.IsUnavailable {
		t.Error("enum unavailable fix not applied")
	}
	if framework.Functions[0].Availability.MacOSIntroduced != "12.0" {
		t.Error("function availability fix not applied")
	}
}

func TestLinkLibOverride(t *testing.T) {
	framework := sampleFramework()
	Apply(&File{LinkLib: "foo"}, framework)
	if framework.LinkLib != "foo" {
		t.Errorf("LinkLib: got %q, want foo", framework.LinkLib)
	}
}

func TestStaleEntriesWarn(t *testing.T) {
	framework := sampleFramework()
	warnings := Apply(&File{
		ExcludeClasses:    []string{"NoSuchClass"},
		ExcludeFunctions:  []string{"NoSuchFunc"},
		RemapTypes:        []TypeRemap{{Class: "FooThing", Selector: "doIt:", Param: "nope", ObjCType: "int"}},
		ForceBitmaskEnums: []string{"NoSuchEnum"},
		AvailabilityFixes: []AvailabilityFix{{Class: "NoSuchClass", MacOSIntroduced: "11.0"}},
	}, framework)
	if len(warnings) != 5 {
		t.Errorf("expected 5 stale warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestApplyAdjacent(t *testing.T) {
	dir := t.TempDir()
	metaPath := filepath.Join(dir, "Foo-arm64-26.5.gometa.json")
	overridesPath := filepath.Join(dir, FileName)
	content := `{"comment": "test", "exclude_classes": ["FooGone"]}`
	if err := os.WriteFile(overridesPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	framework := sampleFramework()
	warnings, err := ApplyAdjacent(metaPath, framework)
	if err != nil {
		t.Fatalf("ApplyAdjacent: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if _, ok := framework.Classes["FooGone"]; ok {
		t.Error("FooGone should be excluded via adjacent file")
	}
}

func TestApplyAdjacentMissingFileIsNoop(t *testing.T) {
	framework := sampleFramework()
	warnings, err := ApplyAdjacent(filepath.Join(t.TempDir(), "Foo-arm64-26.5.gometa.json"), framework)
	if err != nil || warnings != nil {
		t.Errorf("missing overrides file must be a no-op, got warnings=%v err=%v", warnings, err)
	}
	if len(framework.Classes) != 2 {
		t.Error("framework must be unmodified")
	}
}

func TestApplyAdjacentMalformedFileErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("{nope"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := ApplyAdjacent(filepath.Join(dir, "Foo-arm64-26.5.gometa.json"), sampleFramework())
	if err == nil {
		t.Fatal("malformed overrides file must error")
	}
}
