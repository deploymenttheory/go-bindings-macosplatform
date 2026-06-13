//go:build darwin

package pipeline

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/typemap"
)

// TestVZTypeResolution verifies that the foundational ObjC classes that are
// declared in Foundation's headers but whose runtime symbols are exported via
// re-exported libraries (CoreFoundation toll-free bridging, libobjc) are
// recognised as Foundation primary classes after the scanner filter fix.
// It then exercises the type mapper end-to-end against a Virtualization-style
// context and asserts that NSArray<VZ…> resolves to a typed Foundation import
// rather than degrading to unsafe.Pointer.
func TestVZTypeResolution(t *testing.T) {
	matches, _ := filepath.Glob("../../../metadata/*/*.gometa.json")
	sub, _ := filepath.Glob("../../../metadata/*/*/*.gometa.json")
	matches = append(matches, sub...)

	reg, err := LoadAll(matches, "github.com/deploymenttheory/go-bindings-macosplatform/frameworks")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Foundational re-exported classes must be owned by Foundation.
	// Until Foundation has been re-scanned with the union-friendly TBD filter,
	// these may still be missing from the committed metadata; skip in that case
	// rather than fail. After a re-scan this becomes a hard assertion.
	foundationals := []string{"NSArray", "NSDictionary", "NSObject", "NSData", "NSDate", "NSNumber", "NSSet", "NSValue"}
	var missing []string
	for _, name := range foundationals {
		if reg.OwnerIndex[name] != "Foundation" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Skipf("foundational classes not yet owned by Foundation (re-scan needed): %v", missing)
	}

	if !reg.GenericClasses["NSArray"] {
		t.Errorf("NSArray is not flagged as a generic class")
	}
	if got := reg.GenericParamIndex["NSArray"]; len(got) != 1 || got[0] != "ObjectType" {
		t.Errorf("NSArray GenericParamIndex = %v, want [ObjectType]", got)
	}
	if got := reg.GenericParamIndex["NSDictionary"]; len(got) != 2 {
		t.Errorf("NSDictionary GenericParamIndex = %v, want 2 params", got)
	}

	m := typemap.New()
	m.GenericClasses = reg.GenericClasses
	m.GenericParamIndex = reg.GenericParamIndex
	m.OwnerIndex = reg.OwnerIndex
	m.EnumIndex = reg.EnumIndex
	m.EnumGoTypeIndex = reg.EnumGoTypeIndex
	m.TypedefIndex = reg.TypedefIndex
	m.ModulePrefix = reg.ModulePrefix
	m.BlockedImports = resolveBlockedImports(reg)

	ctx := m.BaseContext("Virtualization", reg.ClassNameIndex)
	imports := make(typemap.ImportSet)

	// NSArray<VZConsoleDevice *> * must resolve to a typed Foundation reference,
	// not unsafe.Pointer. The exact form depends on whether NSArray uses Go
	// generics — accept either "*foundation.NSArray" or a typed generic.
	got := m.GoType("NSArray<VZConsoleDevice *> *", ctx, imports)
	if strings.Contains(got, "unsafe.Pointer") {
		t.Errorf("NSArray<VZConsoleDevice *> * resolved to %q, want a typed Foundation import", got)
	}
	if !strings.Contains(got, "foundation.NSArray") && !strings.Contains(got, "*NSArray") {
		t.Errorf("NSArray<VZConsoleDevice *> * resolved to %q, want a Foundation NSArray reference", got)
	}

	if got := m.GoType("NSURL * _Nonnull", ctx, imports); strings.Contains(got, "unsafe.Pointer") {
		t.Errorf("NSURL * resolved to %q, want a typed Foundation reference", got)
	}
}
