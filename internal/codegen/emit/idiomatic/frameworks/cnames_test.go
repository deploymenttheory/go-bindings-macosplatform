package idiofw

import (
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
)

func TestBuildCEnumLocalNames(t *testing.T) {
	framework := &meta.FrameworkMeta{
		Framework: "Hypervisor",
		Enums: map[string]meta.Enum{
			"hv_exit_reason_t":         {},
			"hv_gic_distributor_reg_t": {},
			"hv_ipa_granule_t":         {},
			"hv_error":                 {}, // stripping hv_ would leave "Error" — keeps the prefix
			"_anon_HV_FOO":             {IsAnon: true},
			"VZSomeObjCEnum":           {}, // no underscore: not in the C table
		},
	}
	names := buildCEnumLocalNames(framework, nil)

	want := map[string]string{
		"Hv_exit_reason_t":         "ExitReason",
		"hv_exit_reason_t":         "ExitReason",
		"Hv_gic_distributor_reg_t": "GICDistributorReg",
		"Hv_ipa_granule_t":         "IPAGranule",
		"Hv_error":                 "HvError",
	}
	for in, w := range want {
		if got := names[in]; got != w {
			t.Errorf("localName[%q] = %q; want %q", in, got, w)
		}
	}
	if _, present := names["VZSomeObjCEnum"]; present {
		t.Error("ObjC-style enum names must not enter the C rename table")
	}
	if _, present := names["_anon_HV_FOO"]; present {
		t.Error("anonymous enums must not enter the C rename table")
	}
}

func TestCEnumMemberNames(t *testing.T) {
	members := []meta.EnumMember{
		{Name: "HV_EXIT_REASON_CANCELED", Value: "0"},
		{Name: "HV_EXIT_REASON_EXCEPTION", Value: "1"},
		{Name: "HV_EXIT_REASON_VTIMER_ACTIVATED", Value: "2"},
	}
	got := cEnumMemberNames("ExitReason", members)
	want := map[string]string{
		"HV_EXIT_REASON_CANCELED":         "ExitReasonCanceled",
		"HV_EXIT_REASON_EXCEPTION":        "ExitReasonException",
		"HV_EXIT_REASON_VTIMER_ACTIVATED": "ExitReasonVtimerActivated",
	}
	for in, w := range want {
		if got[in] != w {
			t.Errorf("member %q = %q; want %q", in, got[in], w)
		}
	}

	// A lone member keeps its last segment as the suffix.
	solo := cEnumMemberNames("AllocateFlags", []meta.EnumMember{{Name: "HV_ALLOCATE_DEFAULT", Value: "0"}})
	if solo["HV_ALLOCATE_DEFAULT"] != "AllocateFlagsDefault" {
		t.Errorf("solo member = %q; want AllocateFlagsDefault", solo["HV_ALLOCATE_DEFAULT"])
	}

	// CamelCase (ObjC-style) members opt the whole enum out.
	if cEnumMemberNames("State", []meta.EnumMember{{Name: "VZVirtualMachineStateStopped"}}) != nil {
		t.Error("CamelCase members must return nil (keep existing naming)")
	}
}

func TestStructFieldGoName(t *testing.T) {
	cases := map[string]string{
		"virtual_address":  "VirtualAddress",
		"physical_address": "PhysicalAddress",
		"reason":           "Reason",
		"ipa":              "IPA",
	}
	for in, want := range cases {
		if got := structFieldGoName(in); got != want {
			t.Errorf("structFieldGoName(%q) = %q; want %q", in, got, want)
		}
	}
}
