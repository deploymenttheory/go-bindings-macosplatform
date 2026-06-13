package overrides

import (
	"testing"

	rootoverrides "github.com/deploymenttheory/go-bindings-macosplatform/internal/overrides"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/purecg/meta"
)

func sampleFramework() *meta.FrameworkMeta {
	return &meta.FrameworkMeta{
		Framework: "Foo",
		Classes: map[string]meta.Class{
			"FooThing": {
				Methods: []meta.Method{
					{Selector: "doIt:", Params: []meta.Param{{Name: "value", ObjCType: "NSInteger"}},
						Return: meta.ReturnType{ObjCType: "void"}},
					{Selector: "broken", Return: meta.ReturnType{ObjCType: "FooRef"}},
				},
			},
			"FooGone": {},
		},
		Enums: map[string]meta.Enum{
			"FooOptions": {GoType: "uint64", Members: []meta.EnumMember{{Name: "FooA", Value: "1"}}},
		},
		Functions: []meta.Function{
			{Name: "FooCreate", Params: []meta.Param{{Name: "flags", ObjCType: "int"}},
				Return: meta.ReturnType{ObjCType: "void *"}},
		},
	}
}

// TestApplyMirrorsRootSemantics exercises every override operation against the
// purecg meta types — the same semantics as internal/overrides, on the
// mirrored model.
func TestApplyMirrorsRootSemantics(t *testing.T) {
	framework := sampleFramework()
	warnings := Apply(&rootoverrides.File{
		ExcludeClasses: []string{"FooGone"},
		ExcludeMethods: []rootoverrides.MethodRef{{Class: "FooThing", Selector: "broken"}},
		RemapTypes: []rootoverrides.TypeRemap{
			{Class: "FooThing", Selector: "doIt:", Param: "value", ObjCType: "FooOptions"},
			{Function: "FooCreate", Param: "return", ObjCType: "FooRef"},
		},
		ForceBitmaskEnums: []string{"FooOptions"},
		AvailabilityFixes: []rootoverrides.AvailabilityFix{
			{Class: "FooThing", MacOSIntroduced: "11.0"},
		},
		LinkLib: "foo",
	}, framework)

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if _, ok := framework.Classes["FooGone"]; ok {
		t.Error("FooGone should be excluded")
	}
	methods := framework.Classes["FooThing"].Methods
	if len(methods) != 1 || methods[0].Selector != "doIt:" {
		t.Errorf("broken method should be excluded, got %+v", methods)
	}
	if methods[0].Params[0].ObjCType != "FooOptions" {
		t.Errorf("param remap not applied: %+v", methods[0].Params[0])
	}
	if framework.Functions[0].Return.ObjCType != "FooRef" {
		t.Errorf("function return remap not applied: %+v", framework.Functions[0].Return)
	}
	if !framework.Enums["FooOptions"].IsBitmask {
		t.Error("FooOptions should be forced to bitmask")
	}
	if framework.Classes["FooThing"].Availability.MacOSIntroduced != "11.0" {
		t.Error("availability fix not applied")
	}
	if framework.LinkLib != "foo" {
		t.Errorf("LinkLib: got %q, want foo", framework.LinkLib)
	}
}

func TestStaleEntriesWarn(t *testing.T) {
	framework := sampleFramework()
	warnings := Apply(&rootoverrides.File{
		ExcludeClasses: []string{"NoSuchClass"},
	}, framework)
	if len(warnings) != 1 {
		t.Errorf("expected 1 stale warning, got %v", warnings)
	}
}
