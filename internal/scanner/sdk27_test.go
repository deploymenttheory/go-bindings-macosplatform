//go:build darwin

package scanner

import (
	"path/filepath"
	"testing"
)

func TestSDK27AttributedDeclarationTypes(t *testing.T) {
	sdkPath := t.TempDir()
	header := filepath.Join(FrameworkHeaderDir(sdkPath, "Fixture"), "Fixture.h")
	root := &ASTNode{Kind: "TranslationUnitDecl", Inner: []ASTNode{
		{Kind: "VarDecl", Name: "Notification", StorageClass: "extern", Loc: &Location{FilePath: header},
			Type: &ASTType{QualType: "API_UNAVAILABLE API_AVAILABLE const NSNotificationName"}},
		{Kind: "FunctionDecl", Name: "MakeClock", Loc: &Location{FilePath: header},
			Type: &ASTType{QualType: "CM_RETURNS_RETAINED CMClockRef (void)"}},
		{Kind: "TypedefDecl", Name: "CreateTask", Loc: &Location{FilePath: header},
			Type: &ASTType{QualType: "API_OBSOLETED void (^)(void)"}},
	}}
	framework := Extract(root, sdkPath, "Fixture", "27.0", "arm64", nil)
	if len(framework.Externs) != 1 || framework.Externs[0].ObjCType != "const NSNotificationName" {
		t.Fatalf("extern types retain availability macros: %+v", framework.Externs)
	}
	if len(framework.Functions) != 1 || framework.Functions[0].Return.ObjCType != "CMClockRef" {
		t.Fatalf("function return retains ownership macro: %+v", framework.Functions)
	}
	if got := framework.Typedefs["CreateTask"]; got != "void (^)(void)" {
		t.Fatalf("typedef = %q, want undecorated block type", got)
	}
}

func TestSDK27ObsoletedAvailability(t *testing.T) {
	for _, test := range []struct {
		annotation  string
		removed     bool
		replacement string
	}{
		{`API_OBSOLETED("No longer supported", macos(11.0, 27.0, 27.0)) API_UNAVAILABLE(ios, tvos)`, true, ""},
		{`API_OBSOLETED_WITH_REPLACEMENT("PGCreateDeviceWithDescriptor", macos(11.0, 15.2, 27.0))`, true, "PGCreateDeviceWithDescriptor"},
		{`API_OBSOLETED("Removed on iOS", ios(11.0, 27.0, 27.0))`, false, ""},
	} {
		t.Run(test.annotation, func(t *testing.T) {
			availability := parseAvailAnnotation(test.annotation)
			if availability.IsUnavailable != test.removed || availability.ReplacedBy != test.replacement {
				t.Fatalf("availability = %+v", availability)
			}
			if test.removed && (availability.MacOSIntroduced != "11.0" || availability.MacOSObsoleted != "27.0") {
				t.Fatalf("lost obsoletion version window: %+v", availability)
			}
		})
	}
}
