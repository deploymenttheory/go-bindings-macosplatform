package validate

import (
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// frameworkWith returns a minimal FrameworkMeta with the given classes.
func frameworkWith(name string, classes map[string]macosplatformmetadata.Class) *macosplatformmetadata.FrameworkMeta {
	return &macosplatformmetadata.FrameworkMeta{
		Framework:  name,
		SDKVersion: "26.5",
		Arch:       "arm64",
		Classes:    classes,
	}
}

// findingsContaining returns the findings whose message contains substr.
func findingsContaining(findings []Finding, substr string) []Finding {
	var out []Finding
	for _, f := range findings {
		if strings.Contains(f.Message, substr) {
			out = append(out, f)
		}
	}
	return out
}

func TestDanglingSuperclassReported(t *testing.T) {
	frameworks := []*macosplatformmetadata.FrameworkMeta{
		frameworkWith("Foo", map[string]macosplatformmetadata.Class{
			"FooThing": {Super: "MissingBase", Methods: []macosplatformmetadata.Method{{Selector: "x"}}},
		}),
	}
	findings := Frameworks(frameworks)
	got := findingsContaining(findings, "unknown superclass MissingBase")
	if len(got) != 1 {
		t.Fatalf("expected 1 dangling-superclass finding, got %d: %v", len(got), findings)
	}
	if got[0].Severity != SeverityWarning {
		t.Errorf("severity: got %s, want %s", got[0].Severity, SeverityWarning)
	}
}

func TestCrossFrameworkSuperclassIsNotDangling(t *testing.T) {
	frameworks := []*macosplatformmetadata.FrameworkMeta{
		frameworkWith("Foundation", map[string]macosplatformmetadata.Class{"NSObject": {}}),
		frameworkWith("AppKit", map[string]macosplatformmetadata.Class{"NSView": {Super: "NSObject"}}),
	}
	findings := Frameworks(frameworks)
	if got := findingsContaining(findings, "unknown superclass"); len(got) != 0 {
		t.Errorf("expected no dangling-superclass findings, got %v", got)
	}
}

func TestSuperchainCycleIsError(t *testing.T) {
	frameworks := []*macosplatformmetadata.FrameworkMeta{
		frameworkWith("Foo", map[string]macosplatformmetadata.Class{
			"A": {Super: "B"},
			"B": {Super: "A"},
		}),
	}
	findings := Frameworks(frameworks)
	got := findingsContaining(findings, "superclass cycle")
	if len(got) == 0 {
		t.Fatalf("expected superclass-cycle finding, got %v", findings)
	}
	if got[0].Severity != SeverityError {
		t.Errorf("severity: got %s, want %s", got[0].Severity, SeverityError)
	}
	if !HasErrors(findings) {
		t.Error("HasErrors should be true when a cycle is present")
	}
}

func TestOwnershipTieReportedAcrossFrameworks(t *testing.T) {
	class := macosplatformmetadata.Class{Methods: []macosplatformmetadata.Method{{Selector: "x"}}}
	frameworks := []*macosplatformmetadata.FrameworkMeta{
		frameworkWith("Alpha", map[string]macosplatformmetadata.Class{"Shared": class}),
		frameworkWith("Beta", map[string]macosplatformmetadata.Class{"Shared": class}),
	}
	findings := Frameworks(frameworks)
	got := findingsContaining(findings, "ownership tie")
	if len(got) != 1 {
		t.Fatalf("expected 1 ownership-tie finding, got %d: %v", len(got), findings)
	}
}

func TestOwnershipTieIgnoredForSameFrameworkDuplicates(t *testing.T) {
	// A framework shipped both top-level and nested in an umbrella loads twice
	// under the same name — not an ownership ambiguity.
	class := macosplatformmetadata.Class{Methods: []macosplatformmetadata.Method{{Selector: "x"}}}
	frameworks := []*macosplatformmetadata.FrameworkMeta{
		frameworkWith("PDFKit", map[string]macosplatformmetadata.Class{"PDFDocument": class}),
		frameworkWith("PDFKit", map[string]macosplatformmetadata.Class{"PDFDocument": class}),
	}
	findings := Frameworks(frameworks)
	if got := findingsContaining(findings, "ownership tie"); len(got) != 0 {
		t.Errorf("expected no ownership-tie findings for same-name duplicates, got %v", got)
	}
}

func TestEnumBaseTypeConflictReported(t *testing.T) {
	members := []macosplatformmetadata.EnumMember{{Name: "A", Value: "0"}}
	frameworks := []*macosplatformmetadata.FrameworkMeta{
		func() *macosplatformmetadata.FrameworkMeta {
			f := frameworkWith("Alpha", nil)
			f.Enums = map[string]macosplatformmetadata.Enum{"SharedEnum": {GoType: "int64", Members: members}}
			return f
		}(),
		func() *macosplatformmetadata.FrameworkMeta {
			f := frameworkWith("Beta", nil)
			f.Enums = map[string]macosplatformmetadata.Enum{"SharedEnum": {GoType: "uint64", Members: members}}
			return f
		}(),
	}
	findings := Frameworks(frameworks)
	if got := findingsContaining(findings, "conflicting base types"); len(got) != 1 {
		t.Errorf("expected 1 enum-conflict finding, got %v", findings)
	}
}

func TestEmptyFrameworkReported(t *testing.T) {
	empty := frameworkWith("Hollow", nil)
	swiftOnly := frameworkWith("SwiftThing", nil)
	swiftOnly.IsSwiftOnly = true
	umbrella := frameworkWith("Carbon", nil)
	umbrella.UmbrellaFor = []string{"HIToolbox"}

	findings := Frameworks([]*macosplatformmetadata.FrameworkMeta{empty, swiftOnly, umbrella})
	got := findingsContaining(findings, "no declarations extracted")
	if len(got) != 1 || got[0].Framework != "Hollow" {
		t.Errorf("expected exactly one empty-framework finding for Hollow, got %v", got)
	}
}

func TestAvailabilityAnomalies(t *testing.T) {
	f := frameworkWith("Foo", map[string]macosplatformmetadata.Class{
		"BadVersion":  {Availability: macosplatformmetadata.Availability{MacOSIntroduced: "minVers"}},
		"Inverted":    {Availability: macosplatformmetadata.Availability{MacOSIntroduced: "12.0", MacOSDeprecated: "10.0"}},
		"FutureDepr":  {Availability: macosplatformmetadata.Availability{MacOSIntroduced: "12.0", MacOSDeprecated: "API_TO_BE_DEPRECATED"}},
		"SentinelNum": {Availability: macosplatformmetadata.Availability{MacOSIntroduced: "12.0", MacOSDeprecated: "100000"}},
		"NormalLife":  {Availability: macosplatformmetadata.Availability{MacOSIntroduced: "10.10", MacOSDeprecated: "13.0"}},
	})
	findings := Frameworks([]*macosplatformmetadata.FrameworkMeta{f})

	if got := findingsContaining(findings, `malformed introduced version "minVers"`); len(got) != 1 {
		t.Errorf("expected malformed-version finding for minVers, got %v", findings)
	}
	if got := findingsContaining(findings, "deprecated (10.0) before introduced (12.0)"); len(got) != 1 {
		t.Errorf("expected inverted-range finding, got %v", findings)
	}
	if got := findingsContaining(findings, "API_TO_BE_DEPRECATED"); len(got) != 0 {
		t.Errorf("sentinel API_TO_BE_DEPRECATED must not be reported, got %v", got)
	}
	if got := findingsContaining(findings, "(100000)"); len(got) != 0 {
		t.Errorf("sentinel 100000 must not be reported, got %v", got)
	}
}

func TestSDKConsistency(t *testing.T) {
	a := frameworkWith("Alpha", map[string]macosplatformmetadata.Class{"A": {}})
	b := frameworkWith("Beta", map[string]macosplatformmetadata.Class{"B": {}})
	stale := frameworkWith("Stale", map[string]macosplatformmetadata.Class{"S": {}})
	stale.SDKVersion = "15.0"

	findings := Frameworks([]*macosplatformmetadata.FrameworkMeta{a, b, stale})
	got := findingsContaining(findings, "scanned against SDK 15.0")
	if len(got) != 1 || got[0].Framework != "Stale" {
		t.Errorf("expected SDK-consistency finding for Stale, got %v", findings)
	}
}

func TestParseVersion(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{"10.12", true},
		{"10.12.4", true},
		{"26", true},
		{"", false},
		{"minVers", false},
		{"15_4", false},
		{"10.12.4.1", false},
	}
	for _, c := range cases {
		if _, ok := parseVersion(c.in); ok != c.ok {
			t.Errorf("parseVersion(%q): ok=%v, want %v", c.in, ok, c.ok)
		}
	}
	v1, _ := parseVersion("10.9")
	v2, _ := parseVersion("10.12")
	if v1 >= v2 {
		t.Errorf("10.9 should compare below 10.12 (%d vs %d)", v1, v2)
	}
}
