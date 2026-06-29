package rawlib

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// newEnumFM is a helper that returns a minimal FrameworkMeta with the given
// enums map. Framework is set to "TestFW" for all enum tests.
func newEnumFM(enums map[string]macosplatformmetadata.Enum) *macosplatformmetadata.FrameworkMeta {
	return &macosplatformmetadata.FrameworkMeta{
		Framework: "TestFW",
		Enums:     enums,
	}
}

// TestEnumsEmpty verifies that an empty enum map produces no output.
func TestEnumsEmpty(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

// TestEnumsNamedEnum verifies a basic named enum emits a type declaration and
// a const block containing its members.
func TestEnumsNamedEnum(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"NSComparisonResult": {
			GoType: "int64",
			Members: []macosplatformmetadata.EnumMember{
				{Name: "NSOrderedAscending", Value: "-1"},
				{Name: "NSOrderedSame", Value: "0"},
				{Name: "NSOrderedDescending", Value: "1"},
			},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "type NSComparisonResult int64") {
		t.Errorf("missing type declaration; got:\n%s", out)
	}
	if !strings.Contains(out, "const (") {
		t.Errorf("missing const block; got:\n%s", out)
	}
	if !strings.Contains(out, "NSOrderedAscending NSComparisonResult = -1") {
		t.Errorf("missing first member; got:\n%s", out)
	}
	if !strings.Contains(out, "NSOrderedDescending NSComparisonResult = 1") {
		t.Errorf("missing last member; got:\n%s", out)
	}
}

// TestEnumsAnonEnum verifies that an anonymous enum (IsAnon=true) emits only a
// const block with no type declaration.
func TestEnumsAnonEnum(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"__anon_constants": {
			GoType: "uint64",
			IsAnon: true,
			Members: []macosplatformmetadata.EnumMember{
				{Name: "NSNotFound", Value: "9223372036854775807"},
			},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "type ") {
		t.Errorf("anon enum must not emit a type declaration; got:\n%s", out)
	}
	if !strings.Contains(out, "const (") {
		t.Errorf("missing const block; got:\n%s", out)
	}
	if !strings.Contains(out, "NSNotFound = 9223372036854775807") {
		t.Errorf("missing member; got:\n%s", out)
	}
}

// TestEnumsAnonEnumEmpty verifies that an anonymous enum with no members
// produces no output at all.
func TestEnumsAnonEnumEmpty(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"__anon_empty": {
			GoType:  "uint64",
			IsAnon:  true,
			Members: nil,
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty anon enum, got %q", buf.String())
	}
}

// TestEnumsAvailabilityComment verifies that MacOSIntroduced produces an
// "Introduced: macOS X.Y" comment before the type declaration.
func TestEnumsAvailabilityComment(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"NSActivityOptions": {
			GoType: "uint64",
			Availability: macosplatformmetadata.Availability{
				MacOSIntroduced: "10.9",
			},
			Members: []macosplatformmetadata.EnumMember{
				{Name: "NSActivityBackground", Value: "0"},
			},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "// Introduced: macOS 10.9") {
		t.Errorf("missing availability comment; got:\n%s", out)
	}
}

// TestEnumsDeprecatedComment verifies that MacOSDeprecated produces the
// appropriate deprecated comment.
func TestEnumsDeprecatedComment(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"OldOptions": {
			GoType: "uint64",
			Availability: macosplatformmetadata.Availability{
				MacOSDeprecated: "12.0",
			},
			Members: []macosplatformmetadata.EnumMember{
				{Name: "OldOptionOne", Value: "1"},
			},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Deprecated: Deprecated in macOS 12.0") {
		t.Errorf("missing deprecated comment; got:\n%s", out)
	}
}

// TestEnumsDeprecatedWithReplacement verifies that a deprecated enum that
// also has a ReplacedBy value includes "use X instead" in the comment.
func TestEnumsDeprecatedWithReplacement(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"OldEnum": {
			GoType: "int64",
			Availability: macosplatformmetadata.Availability{
				MacOSDeprecated: "13.0",
				ReplacedBy:      "NewEnum",
			},
			Members: []macosplatformmetadata.EnumMember{
				{Name: "OldEnumValue", Value: "0"},
			},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Use NewEnum instead") {
		t.Errorf("missing replacement hint; got:\n%s", out)
	}
}

// TestEnumsMemberAvailability verifies that availability on an individual enum
// member produces an inline comment above that member.
func TestEnumsMemberAvailability(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"NSProcessInfoThermalState": {
			GoType: "int64",
			Members: []macosplatformmetadata.EnumMember{
				{
					Name:  "NSProcessInfoThermalStateNominal",
					Value: "0",
					Availability: macosplatformmetadata.Availability{
						MacOSIntroduced: "10.10.3",
					},
				},
			},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "// Introduced: macOS 10.10.3") {
		t.Errorf("missing member availability comment; got:\n%s", out)
	}
}

// TestEnumsSortOrder verifies that multiple enums appear in alphabetical order
// by their ObjC name.
func TestEnumsSortOrder(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"ZebraEnum": {
			GoType:  "int64",
			Members: []macosplatformmetadata.EnumMember{{Name: "ZebraVal", Value: "0"}},
		},
		"AlphaEnum": {
			GoType:  "int64",
			Members: []macosplatformmetadata.EnumMember{{Name: "AlphaVal", Value: "1"}},
		},
		"MidEnum": {
			GoType:  "int64",
			Members: []macosplatformmetadata.EnumMember{{Name: "MidVal", Value: "2"}},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	alphaIdx := strings.Index(out, "AlphaEnum")
	midIdx := strings.Index(out, "MidEnum")
	zebraIdx := strings.Index(out, "ZebraEnum")
	if alphaIdx == -1 || midIdx == -1 || zebraIdx == -1 {
		t.Fatalf("one or more enum names missing from output:\n%s", out)
	}
	if !(alphaIdx < midIdx && midIdx < zebraIdx) {
		t.Errorf("enums not in alphabetical order: alpha=%d mid=%d zebra=%d\n%s",
			alphaIdx, midIdx, zebraIdx, out)
	}
}

// TestEnumsIntValue verifies that an integer-typed member value is emitted
// verbatim.
func TestEnumsIntValue(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"NSQualityOfService": {
			GoType: "int64",
			Members: []macosplatformmetadata.EnumMember{
				{Name: "NSQualityOfServiceUserInteractive", Value: "33"},
			},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "= 33") {
		t.Errorf("expected integer value 33 in output; got:\n%s", out)
	}
}

// TestEnumsHexValue verifies that a hexadecimal member value is preserved as
// written.
func TestEnumsHexValue(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"NSDataWritingOptions": {
			GoType: "uint64",
			Members: []macosplatformmetadata.EnumMember{
				{Name: "NSDataWritingAtomic", Value: "0x1"},
			},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "= 0x1") {
		t.Errorf("expected hex value 0x1 in output; got:\n%s", out)
	}
}

// TestEnumsGoTypeName verifies that a name beginning with a lowercase letter
// is capitalised in the emitted type declaration (GoTypeName behaviour).
func TestEnumsGoTypeName(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		// Lowercase-leading ObjC name — uncommon but possible.
		"myPrivateEnum": {
			GoType: "int64",
			Members: []macosplatformmetadata.EnumMember{
				{Name: "myPrivateValue", Value: "0"},
			},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	// GoTypeName capitalises the first letter.
	if !strings.Contains(out, "type MyPrivateEnum int64") {
		t.Errorf("expected capitalised type name MyPrivateEnum; got:\n%s", out)
	}
	if !strings.Contains(out, "MyPrivateValue MyPrivateEnum = 0") {
		t.Errorf("expected capitalised member MyPrivateValue; got:\n%s", out)
	}
}

// TestEnumsMemberDeprecatedWithReplacement verifies that a deprecated member
// with a replacement emits the correct inline comment.
func TestEnumsMemberDeprecatedWithReplacement(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"NSStringEncoding": {
			GoType: "uint64",
			Members: []macosplatformmetadata.EnumMember{
				{
					Name:  "NSASCIIStringEncoding",
					Value: "1",
					Availability: macosplatformmetadata.Availability{
						MacOSDeprecated: "14.0",
						ReplacedBy:      "NSUTF8StringEncoding",
					},
				},
			},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Deprecated: Deprecated in macOS 14.0") {
		t.Errorf("missing deprecated comment on member; got:\n%s", out)
	}
	if !strings.Contains(out, "Use NSUTF8StringEncoding instead") {
		t.Errorf("missing replacement hint on member; got:\n%s", out)
	}
}

// TestEnumsMultipleMembers verifies that all members of a named enum are
// emitted inside a single const block.
func TestEnumsMultipleMembers(t *testing.T) {
	members := []macosplatformmetadata.EnumMember{
		{Name: "NSWindowCollectionBehaviorDefault", Value: "0"},
		{Name: "NSWindowCollectionBehaviorCanJoinAllSpaces", Value: "1"},
		{Name: "NSWindowCollectionBehaviorMoveToActiveSpace", Value: "2"},
	}
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"NSWindowCollectionBehavior": {GoType: "uint64", Members: members},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	// All three members should appear.
	for _, m := range members {
		if !strings.Contains(out, m.Name) {
			t.Errorf("member %q missing from output:\n%s", m.Name, out)
		}
	}
	// Exactly one const block.
	if count := strings.Count(out, "const ("); count != 1 {
		t.Errorf("expected 1 const block, got %d:\n%s", count, out)
	}
}

// TestEnumsNamedAndAnonMixed verifies that a named enum and an anonymous enum
// are both emitted, with the named one producing a type declaration and the
// anonymous one omitting it.
func TestEnumsNamedAndAnonMixed(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"NSBinarySearchingOptions": {
			GoType:  "uint64",
			Members: []macosplatformmetadata.EnumMember{{Name: "NSBinarySearchingFirstEqual", Value: "256"}},
		},
		"__anon_global": {
			GoType:  "uint64",
			IsAnon:  true,
			Members: []macosplatformmetadata.EnumMember{{Name: "NSIntegerMax", Value: "9223372036854775807"}},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "type NSBinarySearchingOptions uint64") {
		t.Errorf("missing named type declaration; got:\n%s", out)
	}
	if !strings.Contains(out, "NSIntegerMax = 9223372036854775807") {
		t.Errorf("missing anon member; got:\n%s", out)
	}
}

// TestEnumsNoAvailabilityNoComment verifies that when no availability is set
// no comment lines appear before the type declaration.
func TestEnumsNoAvailabilityNoComment(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"NSLineBreakMode": {
			GoType:  "int64",
			Members: []macosplatformmetadata.EnumMember{{Name: "NSLineBreakByWordWrapping", Value: "0"}},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "// Introduced") || strings.Contains(out, "// Deprecated") {
		t.Errorf("unexpected comment when no availability set; got:\n%s", out)
	}
}

// TestEnumsClosingBrace verifies the const block is closed with a ')' and the
// output ends with a trailing newline.
func TestEnumsClosingBrace(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"NSEdgeInsets": {
			GoType:  "float64",
			Members: []macosplatformmetadata.EnumMember{{Name: "NSEdgeInsetsZero", Value: "0"}},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, ")\n") {
		t.Errorf("const block not properly closed; got:\n%s", out)
	}
}

// TestEnumsNilMapProducesNoOutput verifies that a nil Enums map is handled
// gracefully and produces no output.
func TestEnumsNilMapProducesNoOutput(t *testing.T) {
	var buf bytes.Buffer
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "TestFW", Enums: nil}
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for nil map, got %q", buf.String())
	}
}

// ── EnumsNeedImports ─────────────────────────────────────────────────────────

func TestEnumsNeedImportsNonBitmask(t *testing.T) {
	// Non-bitmask named enums set needsFmt=true (used for fmt.Sprintf in serialize).
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"Simple": {GoType: "int64", Members: []macosplatformmetadata.EnumMember{{Name: "A", Value: "1"}}},
	})
	needsFmt, needsStrings := EnumsNeedImports(framework)
	if !needsFmt {
		t.Error("non-bitmask enum should set needsFmt=true")
	}
	if needsStrings {
		t.Error("non-bitmask enum should not need strings import")
	}
}

func TestEnumsNeedImportsBitmask(t *testing.T) {
	// Bitmask enums use strings.Join/Split, so they set needsStrings=true.
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"Options": {
			GoType:    "uint64",
			IsBitmask: true,
			Members:   []macosplatformmetadata.EnumMember{{Name: "OptionA", Value: "1"}, {Name: "OptionB", Value: "2"}},
		},
	})
	_, needsStrings := EnumsNeedImports(framework)
	if !needsStrings {
		t.Error("bitmask enum should need strings import")
	}
}

func TestEnumsNeedImportsEmpty(t *testing.T) {
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{})
	needsFmt, needsStrings := EnumsNeedImports(framework)
	if needsFmt || needsStrings {
		t.Error("empty enum map should need no imports")
	}
}

// ── writeEnumStringBitmask / writeEnumParseBitmask ───────────────────────────

func TestEnumsBitmaskStringAndParse(t *testing.T) {
	var buf bytes.Buffer
	framework := newEnumFM(map[string]macosplatformmetadata.Enum{
		"NSEventMask": {
			GoType:    "uint64",
			IsBitmask: true,
			Members: []macosplatformmetadata.EnumMember{
				{Name: "NSEventMaskLeftMouseDown", Value: "1"},
				{Name: "NSEventMaskRightMouseDown", Value: "2"},
			},
		},
	})
	if err := EmitEnums(&buf, framework); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "NSEventMask") {
		t.Errorf("expected NSEventMask type; got:\n%s", out)
	}
	if !strings.Contains(out, "String()") {
		t.Errorf("expected String() method for bitmask enum; got:\n%s", out)
	}
	if !strings.Contains(out, "ParseNSEventMask") {
		t.Errorf("expected ParseNSEventMask function; got:\n%s", out)
	}
	if !strings.Contains(out, "NSEventMaskLeftMouseDown") {
		t.Errorf("expected member name in output; got:\n%s", out)
	}
}

// ── anonEnumsCoveredByNamed ───────────────────────────────────────────────────

func TestAnonEnumsCoveredByNamedSubset(t *testing.T) {
	enums := map[string]macosplatformmetadata.Enum{
		"NSEventType": {
			Members: []macosplatformmetadata.EnumMember{
				{Name: "NSEventTypeLeftMouseDown"},
				{Name: "NSEventTypeRightMouseDown"},
			},
		},
		"_anon_events_": {
			Members: []macosplatformmetadata.EnumMember{
				{Name: "NSEventTypeLeftMouseDown"},  // also in NSEventType
				{Name: "NSEventTypeRightMouseDown"}, // also in NSEventType
			},
		},
	}
	skip := anonEnumsCoveredByNamed(enums)
	if !skip["_anon_events_"] {
		t.Error("anonymous enum whose members all appear in a named enum should be skipped")
	}
}

func TestAnonEnumsCoveredByNamedNotSubset(t *testing.T) {
	enums := map[string]macosplatformmetadata.Enum{
		"NSEventType": {
			Members: []macosplatformmetadata.EnumMember{
				{Name: "NSEventTypeLeftMouseDown"},
			},
		},
		"_anon_extra_": {
			Members: []macosplatformmetadata.EnumMember{
				{Name: "NSEventTypeLeftMouseDown"},
				{Name: "NSEventTypeScrollWheel"}, // NOT in NSEventType
			},
		},
	}
	skip := anonEnumsCoveredByNamed(enums)
	if skip["_anon_extra_"] {
		t.Error("anonymous enum with uncovered members should NOT be skipped")
	}
}

func TestAnonEnumsCoveredByNamedEmpty(t *testing.T) {
	enums := map[string]macosplatformmetadata.Enum{
		"_anon_empty_": {Members: []macosplatformmetadata.EnumMember{}},
	}
	skip := anonEnumsCoveredByNamed(enums)
	// Empty anon enum with no members → not covered
	if skip["_anon_empty_"] {
		t.Error("empty anonymous enum should not be marked as covered")
	}
}
