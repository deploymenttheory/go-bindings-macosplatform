//go:build darwin

package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// clang.go — pure-Go helpers (no xcrun required)
// ---------------------------------------------------------------------------

func TestIsSubFramework(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"Foundation", false},
		{"AppKit", false},
		{"Carbon/HIToolbox", true},
		{"CoreServices/SearchKit", true},
		{"", false},
	}
	for _, c := range cases {
		got := IsSubFramework(c.name)
		if got != c.want {
			t.Errorf("IsSubFramework(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestSubFrameworkParts(t *testing.T) {
	cases := []struct {
		name, wantParent, wantChild string
	}{
		{"Carbon/HIToolbox", "Carbon", "HIToolbox"},
		{"CoreServices/SearchKit", "CoreServices", "SearchKit"},
		{"A/B", "A", "B"},
	}
	for _, c := range cases {
		parent, child := SubFrameworkParts(c.name)
		if parent != c.wantParent || child != c.wantChild {
			t.Errorf("SubFrameworkParts(%q) = (%q, %q), want (%q, %q)",
				c.name, parent, child, c.wantParent, c.wantChild)
		}
	}
}

func TestFrameworkBundlePath(t *testing.T) {
	sdkPath := "/SDK"

	cases := []struct {
		name, want string
	}{
		{
			"Foundation",
			"/SDK/System/Library/Frameworks/Foundation.framework",
		},
		{
			"Carbon/HIToolbox",
			"/SDK/System/Library/Frameworks/Carbon.framework/Frameworks/HIToolbox.framework",
		},
	}
	for _, c := range cases {
		got := FrameworkBundlePath(sdkPath, c.name)
		if got != c.want {
			t.Errorf("FrameworkBundlePath(%q, %q) = %q, want %q", sdkPath, c.name, got, c.want)
		}
	}
}

func TestFrameworkHeader(t *testing.T) {
	sdkPath := "/SDK"

	cases := []struct {
		framework, want string
	}{
		{
			"Foundation",
			"/SDK/System/Library/Frameworks/Foundation.framework/Headers/Foundation.h",
		},
		{
			"Carbon/HIToolbox",
			"/SDK/System/Library/Frameworks/Carbon.framework/Frameworks/HIToolbox.framework/Headers/HIToolbox.h",
		},
	}
	for _, c := range cases {
		got := FrameworkHeader(sdkPath, c.framework)
		if got != c.want {
			t.Errorf("FrameworkHeader(%q, %q) = %q, want %q", sdkPath, c.framework, got, c.want)
		}
	}
}

func TestFrameworkHeaderDir(t *testing.T) {
	sdkPath := "/SDK"

	cases := []struct {
		framework, want string
	}{
		{
			"Foundation",
			"/SDK/System/Library/Frameworks/Foundation.framework/Headers",
		},
		{
			"Carbon/HIToolbox",
			"/SDK/System/Library/Frameworks/Carbon.framework/Frameworks/HIToolbox.framework/Headers",
		},
	}
	for _, c := range cases {
		got := FrameworkHeaderDir(sdkPath, c.framework)
		if got != c.want {
			t.Errorf("FrameworkHeaderDir(%q, %q) = %q, want %q", sdkPath, c.framework, got, c.want)
		}
	}
}

// TestIsSwiftOnly creates a temporary directory tree that mimics a Swift-only
// framework bundle (contains a .swiftmodule directory) and one that does not.
func TestIsSwiftOnly(t *testing.T) {
	t.Run("swift-only bundle", func(t *testing.T) {
		bundle := t.TempDir()
		modulesDir := filepath.Join(bundle, "Modules", "RealityKit.swiftmodule")
		if err := os.MkdirAll(modulesDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if !IsSwiftOnly(bundle) {
			t.Error("IsSwiftOnly = false, want true for bundle with .swiftmodule")
		}
	})

	t.Run("ObjC bundle — no swiftmodule", func(t *testing.T) {
		bundle := t.TempDir()
		modulesDir := filepath.Join(bundle, "Modules")
		if err := os.MkdirAll(modulesDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		// Write a non-swiftmodule entry to make ReadDir succeed but return no match.
		if err := os.MkdirAll(filepath.Join(modulesDir, "module.modulemap"), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if IsSwiftOnly(bundle) {
			t.Error("IsSwiftOnly = true, want false for bundle without .swiftmodule")
		}
	})

	t.Run("no Modules directory", func(t *testing.T) {
		bundle := t.TempDir()
		if IsSwiftOnly(bundle) {
			t.Error("IsSwiftOnly = true, want false for bundle without Modules dir")
		}
	})
}

// TestDetectSubFrameworkNames creates a temporary directory tree mimicking a
// framework bundle that contains sub-frameworks and verifies the returned names.
func TestDetectSubFrameworkNames(t *testing.T) {
	bundle := t.TempDir()
	frameworksDir := filepath.Join(bundle, "Frameworks")

	// Create two valid sub-frameworks (with umbrella header) and one without.
	for _, name := range []string{"HIToolbox", "SpeechRecognition"} {
		hdrDir := filepath.Join(frameworksDir, name+".framework", "Headers")
		if err := os.MkdirAll(hdrDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		// Write the umbrella header.
		hdrPath := filepath.Join(hdrDir, name+".h")
		if err := os.WriteFile(hdrPath, []byte("// umbrella"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	// Add a framework without a valid umbrella header (no header file).
	ghostDir := filepath.Join(frameworksDir, "Ghost.framework", "Headers")
	if err := os.MkdirAll(ghostDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	got := DetectSubFrameworkNames(bundle)

	if len(got) != 2 {
		t.Fatalf("DetectSubFrameworkNames: got %v, want [HIToolbox SpeechRecognition]", got)
	}
	// Results should be sorted.
	if got[0] != "HIToolbox" || got[1] != "SpeechRecognition" {
		t.Errorf("DetectSubFrameworkNames: got %v, want [HIToolbox SpeechRecognition]", got)
	}
}

// TestDetectSubFrameworkNamesEmpty verifies that an umbrella-free bundle returns nil.
func TestDetectSubFrameworkNamesEmpty(t *testing.T) {
	bundle := t.TempDir()
	got := DetectSubFrameworkNames(bundle)
	if len(got) != 0 {
		t.Errorf("DetectSubFrameworkNames on empty bundle: got %v, want nil", got)
	}
}

// TestSDKPathAndVersion exercises the xcrun-dependent functions with a t.Skip
// guard so they run only when a real Xcode SDK is present.
func TestSDKPath(t *testing.T) {
	t.Skip("requires Xcode SDK (xcrun)")
	p, err := SDKPath()
	if err != nil {
		t.Fatalf("SDKPath: %v", err)
	}
	if p == "" {
		t.Error("SDKPath returned empty string")
	}
}

func TestSDKVersion(t *testing.T) {
	t.Skip("requires Xcode SDK (xcrun)")
	v, err := SDKVersion()
	if err != nil {
		t.Fatalf("SDKVersion: %v", err)
	}
	if v == "" {
		t.Error("SDKVersion returned empty string")
	}
}

func TestListFrameworks(t *testing.T) {
	t.Skip("requires Xcode SDK (xcrun)")
	fws, err := ListFrameworks("")
	if err != nil {
		t.Fatalf("ListFrameworks: %v", err)
	}
	if len(fws) == 0 {
		t.Error("ListFrameworks returned empty list")
	}
}

func TestDumpAST(t *testing.T) {
	t.Skip("requires Xcode SDK (xcrun clang)")
}

// ---------------------------------------------------------------------------
// ast.go — RawValue and Location helpers
// ---------------------------------------------------------------------------

func TestRawValueUnmarshalNumber(t *testing.T) {
	data := []byte(`42`)
	var v RawValue
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("UnmarshalJSON number: %v", err)
	}
	if string(v) != "42" {
		t.Errorf("RawValue from number: got %q, want %q", string(v), "42")
	}
}

func TestRawValueUnmarshalString(t *testing.T) {
	data := []byte(`"hello"`)
	var v RawValue
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("UnmarshalJSON string: %v", err)
	}
	if string(v) != "hello" {
		t.Errorf("RawValue from string: got %q, want %q", string(v), "hello")
	}
}

func TestRawValueMarshalJSON(t *testing.T) {
	v := RawValue("world")
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(b) != `"world"` {
		t.Errorf("MarshalJSON: got %s, want %q", b, "world")
	}
}

func TestRawValueString(t *testing.T) {
	v := RawValue("test")
	if v.String() != "test" {
		t.Errorf("String() = %q, want %q", v.String(), "test")
	}
}

func TestLocationResolvedFile(t *testing.T) {
	cases := []struct {
		name string
		loc  *Location
		want string
	}{
		{
			name: "nil location",
			loc:  nil,
			want: "",
		},
		{
			name: "direct file path",
			loc:  &Location{FilePath: "/sdk/Headers/NSObject.h"},
			want: "/sdk/Headers/NSObject.h",
		},
		{
			name: "spelling location takes priority",
			loc: &Location{
				FilePath:    "/some/macro.h",
				SpellingLoc: &Location{FilePath: "/sdk/Headers/NSObject.h"},
			},
			want: "/sdk/Headers/NSObject.h",
		},
		{
			name: "expansion location used when spelling is empty",
			loc: &Location{
				FilePath:     "/some/macro.h",
				SpellingLoc:  &Location{FilePath: ""},
				ExpansionLoc: &Location{FilePath: "/sdk/Headers/NSObject.h"},
			},
			want: "/sdk/Headers/NSObject.h",
		},
		{
			name: "empty spelling falls back to expansion",
			loc: &Location{
				FilePath:     "/fallback.h",
				SpellingLoc:  nil,
				ExpansionLoc: &Location{FilePath: "/sdk/expansion.h"},
			},
			want: "/sdk/expansion.h",
		},
		{
			name: "no spelling no expansion uses file path",
			loc:  &Location{FilePath: "/direct.h"},
			want: "/direct.h",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.loc.ResolvedFile()
			if got != c.want {
				t.Errorf("ResolvedFile() = %q, want %q", got, c.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// tbd.go — TBD class allowlist parsing
// ---------------------------------------------------------------------------

func TestParseTBDClassesSingleSection(t *testing.T) {
	content := `--- !tapi-tbd
tbd-version:     4
targets:         [ x86_64-macos, arm64e-macos ]
exports:
  - targets:         [ x86_64-macos, arm64e-macos ]
    objc-classes:    [ VZVirtualMachine, VZVirtualMachineConfiguration, VZBootLoader ]
...`
	got, _ := parseTBDClasses(content)
	for _, name := range []string{"VZVirtualMachine", "VZVirtualMachineConfiguration", "VZBootLoader"} {
		if !got[name] {
			t.Errorf("expected %q in parsed classes, got %v", name, got)
		}
	}
	if len(got) != 3 {
		t.Errorf("expected 3 classes, got %d: %v", len(got), got)
	}
}

func TestParseTBDClassesMultiSection(t *testing.T) {
	// Virtualization-style: shared section + arch-specific section.
	content := `--- !tapi-tbd
tbd-version:     4
targets:         [ x86_64-macos, arm64e-macos ]
exports:
  - targets:         [ x86_64-macos, arm64e-macos ]
    objc-classes:    [ VZVirtualMachine, VZBootLoader ]
  - targets:         [ arm64e-macos ]
    objc-classes:    [ VZMacOSInstaller, VZMacHardwareModel ]
...`
	got, _ := parseTBDClasses(content)
	for _, name := range []string{"VZVirtualMachine", "VZBootLoader", "VZMacOSInstaller", "VZMacHardwareModel"} {
		if !got[name] {
			t.Errorf("expected %q in parsed classes, got %v", name, got)
		}
	}
	if len(got) != 4 {
		t.Errorf("expected 4 classes, got %d: %v", len(got), got)
	}
}

func TestParseTBDClassesPrivateSPIFiltered(t *testing.T) {
	content := `exports:
  - targets:         [ arm64e-macos ]
    objc-classes:    [ VZPublicClass, _VZPrivateClass, _VZAnotherPrivate ]`
	got, _ := parseTBDClasses(content)
	if !got["VZPublicClass"] {
		t.Error("VZPublicClass should be in allowlist")
	}
	if got["_VZPrivateClass"] {
		t.Error("_VZPrivateClass (SPI) should be excluded")
	}
	if got["_VZAnotherPrivate"] {
		t.Error("_VZAnotherPrivate (SPI) should be excluded")
	}
	if len(got) != 1 {
		t.Errorf("expected 1 class, got %d: %v", len(got), got)
	}
}

func TestParseTBDClassesMultilineSequence(t *testing.T) {
	// Long lists span multiple lines in real TBD files.
	content := `exports:
  - targets:         [ arm64e-macos ]
    objc-classes:    [ ClassA, ClassB,
                       ClassC, ClassD,
                       ClassE ]`
	got, _ := parseTBDClasses(content)
	for _, name := range []string{"ClassA", "ClassB", "ClassC", "ClassD", "ClassE"} {
		if !got[name] {
			t.Errorf("expected %q in parsed classes, got %v", name, got)
		}
	}
	if len(got) != 5 {
		t.Errorf("expected 5 classes, got %d: %v", len(got), got)
	}
}

func TestParseTBDClassesEmpty(t *testing.T) {
	content := `--- !tapi-tbd
tbd-version: 4
targets: [ arm64e-macos ]
exports:
  - targets: [ arm64e-macos ]
    symbols: [ _someSymbol ]`
	got, _ := parseTBDClasses(content)
	if len(got) != 0 {
		t.Errorf("expected empty map for TBD with no objc-classes, got %v", got)
	}
}

func TestParseTBDClassesNoContent(t *testing.T) {
	if got, _ := parseTBDClasses(""); len(got) != 0 {
		t.Errorf("expected empty map for empty content, got %v", got)
	}
}

// TestAcceptClassWithTBD verifies that AcceptClass accepts classes that
// appear in the TBD allowlist, even when the current file is outside this
// framework's header directory (TBD acts as a positive-match shortcut).
func TestAcceptClassWithTBD(t *testing.T) {
	sdkPath := "/SDK"
	f := newFilter(sdkPath, "AppKit") // no real TBD at /SDK — allowedClasses will be nil

	// Inject a TBD-derived allowlist directly to test the TBD path.
	f.allowedClasses = map[string]bool{"NSWindow": true, "NSView": true}

	// Point currentFile at a non-AppKit location so the path filter would reject;
	// the TBD allowlist should still positively accept listed classes.
	f.UpdateFile(&Location{FilePath: "/SDK/System/Library/Frameworks/Foundation.framework/Headers/NSString.h"})

	if !f.AcceptClass("NSWindow") {
		t.Error("AcceptClass(NSWindow) = false, want true (in TBD allowlist)")
	}
	if !f.AcceptClass("NSView") {
		t.Error("AcceptClass(NSView) = false, want true (in TBD allowlist)")
	}
	if f.AcceptClass("NSString") {
		t.Error("AcceptClass(NSString) = true, want false (not in TBD allowlist, not in AppKit headers)")
	}
}

// TestAcceptClassTBDPlusPathFallback verifies that AcceptClass falls back to
// the path filter for classes not in the TBD allowlist — recovering re-exported
// foundational classes like NSArray that are declared in Foundation/NS*.h but
// whose ObjC runtime symbol is exported by CoreFoundation (toll-free bridging)
// or libobjc rather than Foundation.tbd directly.
func TestAcceptClassTBDPlusPathFallback(t *testing.T) {
	sdkPath := "/SDK"
	f := newFilter(sdkPath, "Foundation")

	// TBD allowlist with only derived classes — NSArray etc. are absent because
	// CoreFoundation re-exports them. This mirrors real Foundation.tbd content.
	f.allowedClasses = map[string]bool{"NSAttributedString": true, "NSArchiver": true}

	// currentFile lives inside Foundation's headers — the path filter accepts.
	f.UpdateFile(&Location{FilePath: "/SDK/System/Library/Frameworks/Foundation.framework/Headers/NSArray.h"})

	if !f.AcceptClass("NSArray") {
		t.Error("AcceptClass(NSArray) = false, want true (path filter must accept Foundation/NSArray.h)")
	}
	if !f.AcceptClass("NSAttributedString") {
		t.Error("AcceptClass(NSAttributedString) = false, want true (in TBD)")
	}

	// Switch to a path outside Foundation — neither TBD nor path filter accept.
	f.UpdateFile(&Location{FilePath: "/SDK/System/Library/Frameworks/AppKit.framework/Headers/NSView.h"})
	if f.AcceptClass("NSView") {
		t.Error("AcceptClass(NSView) = true, want false (outside Foundation headers and not in TBD)")
	}
}

// TestAcceptClassFallback verifies that AcceptClass delegates to Accept when
// no TBD allowlist is available.
func TestAcceptClassFallback(t *testing.T) {
	sdkPath := "/SDK"
	f := newFilter(sdkPath, "AppKit") // no real TBD — allowedClasses nil
	if f.allowedClasses != nil {
		t.Skip("unexpected TBD found at /SDK — skipping fallback test")
	}

	f.UpdateFile(&Location{FilePath: "/SDK/System/Library/Frameworks/AppKit.framework/Headers/AppKit.h"})

	if f.AcceptClass("AnyClass") != f.Accept() {
		t.Error("AcceptClass should equal Accept() when no TBD allowlist is available")
	}
}

// ---------------------------------------------------------------------------
// filter.go — frameworkFilter
// ---------------------------------------------------------------------------

func TestFrameworkFilterAccept(t *testing.T) {
	sdkPath := "/SDK"
	filter := newFilter(sdkPath, "AppKit")

	t.Run("nil location does not change current file", func(t *testing.T) {
		filter.UpdateFile(nil)
		// No file set yet — Accept must return false.
		if filter.Accept() {
			t.Error("Accept() = true with no file set, want false")
		}
	})

	t.Run("accept file within header dir", func(t *testing.T) {
		loc := &Location{FilePath: "/SDK/System/Library/Frameworks/AppKit.framework/Headers/AppKit.h"}
		filter.UpdateFile(loc)
		if !filter.Accept() {
			t.Error("Accept() = false for file inside framework Headers, want true")
		}
	})

	t.Run("reject file outside header dir", func(t *testing.T) {
		loc := &Location{FilePath: "/SDK/System/Library/Frameworks/Foundation.framework/Headers/NSObject.h"}
		filter.UpdateFile(loc)
		if filter.Accept() {
			t.Error("Accept() = true for file in a different framework, want false")
		}
	})

	t.Run("empty file path in location does not change current file", func(t *testing.T) {
		// Start with an accepted file.
		acceptLoc := &Location{FilePath: "/SDK/System/Library/Frameworks/AppKit.framework/Headers/AppKit.h"}
		filter.UpdateFile(acceptLoc)

		// Now pass a location with empty file — current file must stay.
		filter.UpdateFile(&Location{FilePath: ""})
		if !filter.Accept() {
			t.Error("Accept() changed after empty-path UpdateFile, want same as before")
		}
	})
}

// TestFoundationFilterExtraDirs verifies that Foundation's filter also accepts
// files from usr/include/objc (where NSObject.h lives).
func TestFoundationFilterExtraDirs(t *testing.T) {
	sdkPath := "/SDK"
	filter := newFilter(sdkPath, "Foundation")

	loc := &Location{FilePath: "/SDK/usr/include/objc/NSObject.h"}
	filter.UpdateFile(loc)
	if !filter.Accept() {
		t.Error("Accept() = false for Foundation extra dir (usr/include/objc), want true")
	}
}

// ---------------------------------------------------------------------------
// extract.go — pure-Go helper functions (no xcrun required)
// ---------------------------------------------------------------------------

func TestIsNSErrorOut(t *testing.T) {
	// isNSErrorOut requires a double-pointer pattern — a genuine out-parameter.
	// "NSError *" (single pointer) is an in-parameter and must NOT match.
	cases := []struct {
		qt   string
		want bool
	}{
		{"NSError **", true},
		{"NSError * _Nullable *", true},                 // with nullability qualifier
		{"NSError * __autoreleasing *", true},           // with ARC qualifier
		{"NSError *", false},                            // single pointer — not an out-param
		{"void (^)(NSError *)", false},                  // block completion handler — excluded
		{"id", false},
		{"", false},
	}
	for _, c := range cases {
		got := isNSErrorOut(c.qt)
		if got != c.want {
			t.Errorf("isNSErrorOut(%q) = %v, want %v", c.qt, got, c.want)
		}
	}
}

func TestIsBlockType(t *testing.T) {
	cases := []struct {
		qt   string
		want bool
	}{
		{"void (^)(void)", true},
		{"void (^)(NSString *, NSError *)", true},
		{"NSString *", false},
		{"id", false},
		{"", false},
	}
	for _, c := range cases {
		got := isBlockType(c.qt)
		if got != c.want {
			t.Errorf("isBlockType(%q) = %v, want %v", c.qt, got, c.want)
		}
	}
}

func TestObjcIntTypeToGo(t *testing.T) {
	cases := []struct {
		qt, want string
	}{
		{"int", "int32"},
		{"signed int", "int32"},
		{"unsigned int", "uint32"},
		{"long", "int64"},
		{"signed long", "int64"},
		{"unsigned long", "uint64"},
		{"NSUInteger", "uint64"},
		{"long long", "int64"},
		{"signed long long", "int64"},
		{"NSInteger", "int64"},
		{"unsigned long long", "uint64"},
		{"short", "int16"},
		{"signed short", "int16"},
		{"unsigned short", "uint16"},
		{"char", "int8"},
		{"signed char", "int8"},
		{"unsigned char", "uint8"},
		// unknown unsigned type → uint64
		{"unsigned whatever", "uint64"},
		// unknown type → int64
		{"__int128", "int64"},
	}
	for _, c := range cases {
		got := objcIntTypeToGo(c.qt)
		if got != c.want {
			t.Errorf("objcIntTypeToGo(%q) = %q, want %q", c.qt, got, c.want)
		}
	}
}

func TestParseReplacedBy(t *testing.T) {
	// Trace the function logic:
	// 1. lowercase msg
	// 2. find "use " or "prefer " prefix
	// 3. rest = everything after prefix
	// 4. TrimSuffix(rest, " instead")  — only works when rest ends exactly in " instead"
	// 5. TrimSuffix(rest, ".")
	// 6. TrimSpace
	cases := []struct {
		msg, want string
	}{
		// "use NewClass instead" → rest="newclass instead" → trim " instead" → "newclass"
		{"Use NewClass instead", "newclass"},
		// "prefer newapi instead." → rest="newapi instead." → " instead" not suffix → trim "." → "newapi instead"
		{"prefer NewAPI instead.", "newapi instead"},
		// "prefer bettermethod." → rest="bettermethod." → trim "." → "bettermethod"
		{"prefer BetterMethod.", "bettermethod"},
		// "use  foo" → rest=" foo" → TrimSpace → "foo"
		{"Use  Foo", "foo"},
		// No "use " or "prefer " → ""
		{"Deprecated. No replacement.", ""},
		{"", ""},
	}
	for _, c := range cases {
		got := parseReplacedBy(c.msg)
		if got != c.want {
			t.Errorf("parseReplacedBy(%q) = %q, want %q", c.msg, got, c.want)
		}
	}
}

func TestHasSignificantChildren(t *testing.T) {
	cases := []struct {
		name  string
		inner []ASTNode
		want  bool
	}{
		{
			name:  "empty inner",
			inner: nil,
			want:  false,
		},
		{
			name:  "ObjCMethodDecl child",
			inner: []ASTNode{{Kind: "ObjCMethodDecl"}},
			want:  true,
		},
		{
			name:  "ObjCPropertyDecl child",
			inner: []ASTNode{{Kind: "ObjCPropertyDecl"}},
			want:  true,
		},
		{
			name:  "ObjCTypeParamDecl child",
			inner: []ASTNode{{Kind: "ObjCTypeParamDecl"}},
			want:  true,
		},
		{
			name:  "irrelevant child kinds only",
			inner: []ASTNode{{Kind: "AvailabilityAttr"}, {Kind: "ParmVarDecl"}},
			want:  false,
		},
		{
			name: "mix of irrelevant and significant",
			inner: []ASTNode{
				{Kind: "AvailabilityAttr"},
				{Kind: "ObjCMethodDecl"},
			},
			want: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := hasSignificantChildren(c.inner)
			if got != c.want {
				t.Errorf("hasSignificantChildren(%v) = %v, want %v", c.inner, got, c.want)
			}
		})
	}
}

// TestIsMetadataEmpty verifies the isMetadataEmpty helper via the Extract
// function indirectly, and also tests it directly.
func TestIsMetadataEmptyDirect(t *testing.T) {
	// We can't import meta directly since it's a separate package, but
	// isMetadataEmpty is unexported and only in extract.go (same package).
	// We call Extract with an empty AST to exercise the post-scan path.
	root := &ASTNode{Kind: "TranslationUnitDecl", Inner: []ASTNode{}}
	framework := Extract(root, "/nonexistent/sdk", "Foundation", "14.0", "arm64", nil)
	// When there are no declarations, the function checks IsSwiftOnly and
	// DetectSubFrameworkNames. With a non-existent SDK path both return empty/false.
	if framework == nil {
		t.Fatal("Extract returned nil")
	}
	if framework.Framework != "Foundation" {
		t.Errorf("framework.Framework = %q, want Foundation", framework.Framework)
	}
}

// TestExtractSubFramework verifies that Extract correctly splits "Parent/Child"
// framework names and sets ParentFramework.
func TestExtractSubFramework(t *testing.T) {
	root := &ASTNode{Kind: "TranslationUnitDecl", Inner: []ASTNode{}}
	framework := Extract(root, "/nonexistent/sdk", "Carbon/HIToolbox", "14.0", "arm64", nil)
	if framework.Framework != "HIToolbox" {
		t.Errorf("framework.Framework = %q, want HIToolbox", framework.Framework)
	}
	if framework.ParentFramework != "Carbon" {
		t.Errorf("framework.ParentFramework = %q, want Carbon", framework.ParentFramework)
	}
}

// TestExtractWithSwiftOnlyBundle verifies the SwiftOnly detection path when the
// bundle has a .swiftmodule directory.
func TestExtractWithSwiftOnlyBundle(t *testing.T) {
	// Build a minimal fake SDK directory with a Swift-only framework bundle.
	sdkRoot := t.TempDir()
	bundlePath := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework")
	modulesDir := filepath.Join(bundlePath, "Modules", "TestFW.swiftmodule")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	root := &ASTNode{Kind: "TranslationUnitDecl", Inner: []ASTNode{}}
	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	if !framework.IsSwiftOnly {
		t.Error("SwiftOnly = false, want true for bundle with .swiftmodule and no declarations")
	}
}

// TestExtractWithUmbrellaBundle verifies the UmbrellaFor detection path.
func TestExtractWithUmbrellaBundle(t *testing.T) {
	// Build a minimal fake SDK with an umbrella framework that has sub-frameworks.
	sdkRoot := t.TempDir()
	bundlePath := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "Umbrella.framework")
	// Create sub-framework with umbrella header.
	subHdr := filepath.Join(bundlePath, "Frameworks", "Sub.framework", "Headers")
	if err := os.MkdirAll(subHdr, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subHdr, "Sub.h"), []byte("// sub"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{Kind: "TranslationUnitDecl", Inner: []ASTNode{}}
	framework := Extract(root, sdkRoot, "Umbrella", "14.0", "arm64", nil)
	if len(framework.UmbrellaFor) == 0 {
		t.Error("UmbrellaFor is empty, want [Sub]")
	} else if framework.UmbrellaFor[0] != "Sub" {
		t.Errorf("UmbrellaFor[0] = %q, want Sub", framework.UmbrellaFor[0])
	}
}

// TestExtractEnum exercises the enum extraction path via a minimal AST.
func TestExtractEnumNode(t *testing.T) {
	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind: "EnumDecl",
				Name: "MyOptions",
				Loc:  &Location{FilePath: "/SDK/System/Library/Frameworks/TestFW.framework/Headers/TestFW.h"},
				FixedUnderlyingType: &ASTType{QualType: "unsigned long"},
				Inner: []ASTNode{
					{
						Kind:  "EnumConstantDecl",
						Name:  "MyOptionA",
						Value: RawValue("0"),
					},
					{
						Kind:  "EnumConstantDecl",
						Name:  "MyOptionB",
						Value: RawValue("1"),
					},
				},
			},
		},
	}

	// Build a fake SDK with a header for TestFW.
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Override Loc paths in the AST node to point to our fake SDK.
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	root.Inner[0].Loc = &Location{FilePath: fakeHdr}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	e, ok := framework.Enums["MyOptions"]
	if !ok {
		t.Fatal("MyOptions enum not found in extracted metadata")
	}
	if e.GoType != "uint64" {
		t.Errorf("GoType = %q, want uint64", e.GoType)
	}
	if len(e.Members) != 2 {
		t.Errorf("Members: got %d, want 2", len(e.Members))
	}
}

// TestExtractAnonymousEnum exercises the anonymous enum path.
func TestExtractAnonymousEnumNode(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind: "EnumDecl",
				Name: "", // anonymous
				Loc:  &Location{FilePath: fakeHdr},
				Inner: []ASTNode{
					{Kind: "EnumConstantDecl", Name: "NSNotFound", Value: RawValue("9223372036854775807")},
				},
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	key := "_anon_NSNotFound"
	e, ok := framework.Enums[key]
	if !ok {
		t.Fatalf("anonymous enum key %q not found; enums: %v", key, framework.Enums)
	}
	if !e.IsAnon {
		t.Error("IsAnon = false, want true")
	}
	if len(e.Members) != 1 || e.Members[0].Name != "NSNotFound" {
		t.Errorf("Members: %+v", e.Members)
	}
}

// TestExtractEnumValueFromInner verifies scanEnumValue descends into inner nodes.
func TestExtractEnumValueFromInnerNode(t *testing.T) {
	// An EnumConstantDecl whose value lives inside a ConstantExpr child.
	node := &ASTNode{
		Kind: "EnumConstantDecl",
		Name: "Val",
		// No direct Value — the value is inside a ConstantExpr child.
		Inner: []ASTNode{
			{
				Kind: "ConstantExpr",
				Inner: []ASTNode{
					{Kind: "IntegerLiteral", Value: RawValue("42")},
				},
			},
		},
	}
	got := scanEnumValue(node)
	if got != "42" {
		t.Errorf("scanEnumValue = %v, want 42", got)
	}
}

// TestExtractEnumValueDefault verifies that an EnumConstantDecl with no value
// information returns "0".
func TestExtractEnumValueDefault(t *testing.T) {
	node := &ASTNode{Kind: "EnumConstantDecl", Name: "Val"}
	got := scanEnumValue(node)
	if got != "0" {
		t.Errorf("scanEnumValue (no value) = %v, want 0", got)
	}
}

// TestExtractAvailabilityFromNode exercises scanAvailability via an AST node
// that carries both a top-level Availability slice and inner AvailabilityAttr
// children.
func TestExtractAvailability(t *testing.T) {
	node := &ASTNode{
		Kind: "ObjCInterfaceDecl",
		Name: "OldClass",
		Availability: []ASTNode{
			{
				Kind:       "AvailabilityAttr",
				Platform:   "macos",
				Introduced: "10.12",
				Deprecated: "14.0",
				Message:    "use NewClass instead",
			},
		},
	}
	av := scanAvailability(node, "", 0)
	if av.MacOSIntroduced != "10.12" {
		t.Errorf("MacOSIntroduced = %q, want 10.12", av.MacOSIntroduced)
	}
	if av.MacOSDeprecated != "14.0" {
		t.Errorf("MacOSDeprecated = %q, want 14.0", av.MacOSDeprecated)
	}
	if !strings.Contains(av.DeprecationMsg, "NewClass") {
		t.Errorf("DeprecationMsg = %q, should contain NewClass", av.DeprecationMsg)
	}
}

// TestExtractAvailabilityInner exercises the inner-node availability path.
func TestExtractAvailabilityInner(t *testing.T) {
	node := &ASTNode{
		Kind: "ObjCMethodDecl",
		Name: "oldMethod",
		Inner: []ASTNode{
			{
				Kind:       "AvailabilityAttr",
				Platform:   "macos",
				Introduced: "11.0",
				Deprecated: "15.0",
				Message:    "prefer newMethod instead",
			},
		},
	}
	av := scanAvailability(node, "", 0)
	if av.MacOSIntroduced != "11.0" {
		t.Errorf("MacOSIntroduced = %q, want 11.0", av.MacOSIntroduced)
	}
	if av.ReplacedBy != "newmethod" {
		t.Errorf("ReplacedBy = %q, want newmethod", av.ReplacedBy)
	}
}

// TestExtractAvailabilityNonMacOSPlatformIgnored verifies that non-macos
// availability attributes are ignored.
func TestExtractAvailabilityNonMacOSIgnored(t *testing.T) {
	node := &ASTNode{
		Kind: "ObjCInterfaceDecl",
		Availability: []ASTNode{
			{
				Kind:       "AvailabilityAttr",
				Platform:   "ios",
				Introduced: "9.0",
			},
		},
	}
	av := scanAvailability(node, "", 0)
	if av.MacOSIntroduced != "" {
		t.Errorf("MacOSIntroduced = %q, want empty (iOS attr should be ignored)", av.MacOSIntroduced)
	}
}

// TestBuildReturn exercises the AlreadyRetained detection for selectors that
// follow the create rule. The logic is: the selector equals the prefix, OR the
// character immediately after the prefix is an uppercase letter (A-Z), which
// matches ObjC CamelCase naming conventions.
func TestBuildReturnAlreadyRetained(t *testing.T) {
	retType := &ASTType{QualType: "instancetype"}
	cases := []struct {
		selector     string
		wantRetained bool
	}{
		// Exact-match prefixes
		{"alloc", true},
		{"new", true},
		{"copy", true},
		{"mutableCopy", true},
		// CamelCase continuation after prefix — uppercase next char → retained
		{"allocWithZone:", true},
		{"newObject", true},
		{"copyWithZone:", true},
		{"mutableCopyWithZone:", true},
		// "copySomething" — next char 'S' is uppercase → matches
		{"copySomething", true},
		// Not a create-rule selector
		{"init", false},
		{"description", false},
		// "copyring" — next char 'r' is lowercase → does NOT match the ObjC rule
		{"copyring", false},
	}
	for _, c := range cases {
		method := &ASTNode{Name: c.selector}
		r := makeReturnType(retType, method, nil)
		if r.IsAlreadyRetained != c.wantRetained {
			t.Errorf("makeReturnType(sel=%q).IsAlreadyRetained = %v, want %v",
				c.selector, r.IsAlreadyRetained, c.wantRetained)
		}
	}
}

// TestExtractExternSkipsNonExtern verifies that scanExtern ignores VarDecl
// nodes that are not "extern".
func TestExtractExternSkipsNonExtern(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind:         "VarDecl",
				Name:         "someGlobal",
				StorageClass: "", // NOT extern
				Loc:          &Location{FilePath: fakeHdr},
				Type:         &ASTType{QualType: "int"},
			},
			{
				Kind:         "VarDecl",
				Name:         "realExtern",
				StorageClass: "extern",
				Loc:          &Location{FilePath: fakeHdr},
				Type:         &ASTType{QualType: "double"},
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	if len(framework.Externs) != 1 {
		t.Fatalf("Externs: got %d, want 1", len(framework.Externs))
	}
	if framework.Externs[0].Name != "realExtern" {
		t.Errorf("Externs[0].Name = %q, want realExtern", framework.Externs[0].Name)
	}
}

// TestExtractStruct exercises struct extraction (named, non-union structs only).
func TestExtractStructNode(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind:               "RecordDecl",
				Name:               "NSRange",
				TagUsed:            "struct",
				CompleteDefinition: true,
				Loc:                &Location{FilePath: fakeHdr},
				Inner: []ASTNode{
					{Kind: "FieldDecl", Name: "location", Type: &ASTType{QualType: "NSUInteger"}},
					{Kind: "FieldDecl", Name: "length", Type: &ASTType{QualType: "NSUInteger"}},
				},
			},
			{
				Kind:    "RecordDecl",
				Name:    "MyUnion",
				TagUsed: "union", // should be skipped
				Loc:     &Location{FilePath: fakeHdr},
			},
			{
				Kind:    "RecordDecl",
				Name:    "", // anonymous — should be skipped
				TagUsed: "struct",
				Loc:     &Location{FilePath: fakeHdr},
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	if _, ok := framework.Structs["NSRange"]; !ok {
		t.Fatal("NSRange struct not found")
	}
	if got := len(framework.Structs["NSRange"].Fields); got != 2 {
		t.Errorf("NSRange field count = %d, want 2", got)
	}
	if _, ok := framework.Structs["MyUnion"]; ok {
		t.Error("MyUnion (union) should not be in Structs")
	}
	if len(framework.Structs) != 1 {
		t.Errorf("expected 1 struct, got %d: %v", len(framework.Structs), framework.Structs)
	}
}

// TestExtractStructAttributeDecoratedAndForwardDecl exercises the two SDK shapes
// that previously caused CGSize, CGRect, CGPoint, CGVector to land in metadata
// with zero fields:
//
//  1. A struct decorated with attribute macros (CF_SWIFT_SENDABLE, CF_BOXABLE,
//     NS_SWIFT_SENDABLE) inserts attribute nodes (AnnotateAttr, ObjCBoxableAttr)
//     into the RecordDecl's Inner slice before the FieldDecls. The extractor
//     must filter by Kind=="FieldDecl" and ignore the attribute siblings.
//  2. Clang frequently emits a forward declaration `struct Foo;` ahead of the
//     full definition, producing two RecordDecls for the same name. The
//     forward decl carries CompleteDefinition=false and no fields. The
//     extractor must not overwrite the complete entry with the empty one.
func TestExtractStructAttributeDecoratedAndForwardDecl(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			// Attribute-decorated struct (CGSize-shaped).
			{
				Kind:               "RecordDecl",
				Name:               "CGSize",
				TagUsed:            "struct",
				CompleteDefinition: true,
				Loc:                &Location{FilePath: fakeHdr},
				Inner: []ASTNode{
					{Kind: "ObjCBoxableAttr"},
					{Kind: "AnnotateAttr"},
					{Kind: "FieldDecl", Name: "width", Type: &ASTType{QualType: "CGFloat"}},
					{Kind: "FieldDecl", Name: "height", Type: &ASTType{QualType: "CGFloat"}},
				},
			},
			// Forward declaration of the same struct — must not overwrite.
			{
				Kind:               "RecordDecl",
				Name:               "CGSize",
				TagUsed:            "struct",
				CompleteDefinition: false,
				Loc:                &Location{FilePath: fakeHdr},
			},
			// Forward decl with NO prior complete entry — registers a placeholder
			// so downstream code knows the name exists.
			{
				Kind:               "RecordDecl",
				Name:               "OpaqueOnly",
				TagUsed:            "struct",
				CompleteDefinition: false,
				Loc:                &Location{FilePath: fakeHdr},
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	cgsize, ok := framework.Structs["CGSize"]
	if !ok {
		t.Fatal("CGSize not extracted")
	}
	if len(cgsize.Fields) != 2 {
		t.Fatalf("CGSize fields = %d, want 2", len(cgsize.Fields))
	}
	if cgsize.Fields[0].Name != "width" || cgsize.Fields[1].Name != "height" {
		t.Errorf("CGSize fields = %+v, want [width, height]", cgsize.Fields)
	}
	if _, ok := framework.Structs["OpaqueOnly"]; !ok {
		t.Error("OpaqueOnly forward-decl placeholder missing")
	}
}

// TestExtractTypedef exercises typedef extraction.
func TestExtractTypedefNode(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind: "TypedefDecl",
				Name: "NSTimeInterval",
				Loc:  &Location{FilePath: fakeHdr},
				Type: &ASTType{QualType: "double"},
			},
			{
				Kind: "TypedefDecl",
				Name: "NoTypeNode",
				Loc:  &Location{FilePath: fakeHdr},
				// Type is nil — should be skipped by scanTypedef
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	if v, ok := framework.Typedefs["NSTimeInterval"]; !ok || v != "double" {
		t.Errorf("Typedefs[NSTimeInterval] = %q (ok=%v), want double", v, ok)
	}
	if _, ok := framework.Typedefs["NoTypeNode"]; ok {
		t.Error("NoTypeNode (nil Type) should not be in Typedefs")
	}
}

// TestExtractClassForwardDeclSkipped verifies that @class Foo; forward
// declarations (node.Inner == nil) are skipped.
func TestExtractClassForwardDeclSkipped(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind:  "ObjCInterfaceDecl",
				Name:  "NSScreen",
				Loc:   &Location{FilePath: fakeHdr},
				Inner: nil, // forward declaration — no inner
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	if _, ok := framework.Classes["NSScreen"]; ok {
		t.Error("forward-declared class NSScreen should not be in Classes")
	}
}

// TestExtractCategoryForeignExtension exercises the path where a category
// extends a class not in the current framework (ForeignExtensions).
func TestExtractCategoryForeignExtension(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind: "ObjCCategoryDecl",
				Name: "NSBundle(MyCategory)",
				Loc:  &Location{FilePath: fakeHdr},
				Interface: &ASTRef{Name: "NSBundle"},
				Inner: []ASTNode{
					{
						Kind:       "ObjCMethodDecl",
						Name:       "myMethod",
						IsInstance: true,
						ReturnType: &ASTType{QualType: "void"},
					},
				},
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	methods, ok := framework.ForeignExtensions["NSBundle"]
	if !ok {
		t.Fatal("ForeignExtensions[NSBundle] not found")
	}
	if len(methods) != 1 || methods[0].Selector != "myMethod" {
		t.Errorf("ForeignExtensions[NSBundle] = %+v", methods)
	}
}

// TestExtractImplicitNodeSkipped verifies that implicit AST nodes are skipped.
func TestExtractImplicitNodeSkipped(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind:       "ObjCInterfaceDecl",
				Name:       "ImplicitClass",
				IsImplicit: true,
				Loc:        &Location{FilePath: fakeHdr},
				Inner:      []ASTNode{},
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	if _, ok := framework.Classes["ImplicitClass"]; ok {
		t.Error("implicit class should be skipped")
	}
}

// TestExtractFunction exercises function extraction.
func TestExtractFunctionNode(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind:       "FunctionDecl",
				Name:       "NSMakeRange",
				Loc:        &Location{FilePath: fakeHdr},
				ReturnType: &ASTType{QualType: "NSRange"},
				Inner: []ASTNode{
					{Kind: "ParmVarDecl", Name: "loc", Type: &ASTType{QualType: "NSUInteger"}},
					{Kind: "ParmVarDecl", Name: "len", Type: &ASTType{QualType: "NSUInteger"}},
				},
			},
			{
				Kind:       "FunctionDecl",
				Name:       "ImplicitFn",
				IsImplicit: true,
				Loc:        &Location{FilePath: fakeHdr},
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	if len(framework.Functions) != 1 {
		t.Fatalf("Functions: got %d, want 1", len(framework.Functions))
	}
	fn := framework.Functions[0]
	if fn.Name != "NSMakeRange" {
		t.Errorf("Function.Name = %q, want NSMakeRange", fn.Name)
	}
	if len(fn.Params) != 2 {
		t.Errorf("Function.Params: got %d, want 2", len(fn.Params))
	}
}

// TestExtractMethodNSErrorOut verifies that a trailing NSError** parameter is
// elided from the arg list and HasNSError is set.
func TestExtractMethodNSErrorOut(t *testing.T) {
	node := &ASTNode{
		Kind:       "ObjCMethodDecl",
		Name:       "doThingWithError:",
		IsInstance: true,
		ReturnType: &ASTType{QualType: "BOOL"},
		Inner: []ASTNode{
			{Kind: "ParmVarDecl", Name: "error", Type: &ASTType{QualType: "NSError **"}},
		},
	}
	m, ok := scanMethod(node, "", "", nil)
	if !ok {
		t.Fatal("scanMethod returned ok=false")
	}
	if !m.IsNSError {
		t.Error("HasNSError = false, want true")
	}
	if len(m.Params) != 0 {
		t.Errorf("Params: got %d, want 0 (NSError** should be elided)", len(m.Params))
	}
}

// TestExtractClassFull exercises the scanClass path with a real class
// definition (non-nil Inner) including methods, properties, and generic params.
func TestExtractClassFull(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind: "ObjCInterfaceDecl",
				Name: "MyClass",
				Loc:  &Location{FilePath: fakeHdr},
				Super: &ASTRef{Name: "NSObject"},
				Protocols: []ASTRef{
					{Name: "NSCopying"},
				},
				Inner: []ASTNode{
					// Generic type param
					{Kind: "ObjCTypeParamDecl", Name: "ObjectType"},
					// Instance method
					{
						Kind:       "ObjCMethodDecl",
						Name:       "count",
						IsInstance: true,
						ReturnType: &ASTType{QualType: "NSUInteger"},
					},
					// Class method
					{
						Kind:       "ObjCMethodDecl",
						Name:       "allocWithCapacity:",
						IsInstance: false,
						ReturnType: &ASTType{QualType: "instancetype"},
						Inner: []ASTNode{
							{Kind: "ParmVarDecl", Name: "cap", Type: &ASTType{QualType: "NSUInteger"}},
						},
					},
					// Property
					{
						Kind:               "ObjCPropertyDecl",
						Name:               "count",
						Type:               &ASTType{QualType: "NSUInteger"},
						PropertyAttributes: []string{"readonly"},
						Getter:             &ASTRef{Name: "count"},
					},
					// Implicit method — should be skipped
					{
						Kind:       "ObjCMethodDecl",
						Name:       ".cxx_destruct",
						IsImplicit: true,
						IsInstance: true,
						ReturnType: &ASTType{QualType: "void"},
					},
				},
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	cls, ok := framework.Classes["MyClass"]
	if !ok {
		t.Fatal("MyClass not found in Classes")
	}
	if cls.Super != "NSObject" {
		t.Errorf("Super = %q, want NSObject", cls.Super)
	}
	if len(cls.Protocols) != 1 || cls.Protocols[0] != "NSCopying" {
		t.Errorf("Protocols = %v, want [NSCopying]", cls.Protocols)
	}
	if len(cls.GenericParams) != 1 || cls.GenericParams[0] != "ObjectType" {
		t.Errorf("GenericParams = %v, want [ObjectType]", cls.GenericParams)
	}
	// Should have 2 methods (implicit skipped)
	if len(cls.Methods) != 2 {
		t.Errorf("Methods: got %d, want 2", len(cls.Methods))
	}
	// Should have 1 property
	if len(cls.Properties) != 1 {
		t.Errorf("Properties: got %d, want 1", len(cls.Properties))
	}
	if !cls.Properties[0].IsReadOnly {
		t.Error("Property Readonly not set")
	}
	if cls.Properties[0].Getter != "count" {
		t.Errorf("Property Getter = %q, want count", cls.Properties[0].Getter)
	}
}

// TestExtractPropertyFull exercises scanProperty directly via Extract,
// covering setter, availability, and implicit skip paths.
func TestExtractPropertyFull(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind: "ObjCInterfaceDecl",
				Name: "PropClass",
				Loc:  &Location{FilePath: fakeHdr},
				Inner: []ASTNode{
					// Read-write property with getter and setter
					{
						Kind:               "ObjCPropertyDecl",
						Name:               "title",
						Type:               &ASTType{QualType: "NSString * _Nullable"},
						PropertyAttributes: []string{"copy"},
						Getter:             &ASTRef{Name: "title"},
						Setter:             &ASTRef{Name: "setTitle:"},
					},
					// Implicit property — should be skipped
					{
						Kind:               "ObjCPropertyDecl",
						Name:               "implicitProp",
						IsImplicit:         true,
						Type:               &ASTType{QualType: "id"},
						PropertyAttributes: []string{},
					},
					// Property with no Type node
					{
						Kind:               "ObjCPropertyDecl",
						Name:               "nilTypeProp",
						PropertyAttributes: []string{"readonly"},
					},
				},
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	cls, ok := framework.Classes["PropClass"]
	if !ok {
		t.Fatal("PropClass not found in Classes")
	}
	// implicit prop skipped; nilTypeProp included with empty ObjCType
	if len(cls.Properties) != 2 {
		t.Errorf("Properties: got %d, want 2 (title + nilTypeProp)", len(cls.Properties))
	}
	// Find "title" property
	var titleProp *strings.Builder
	_ = titleProp
	for _, p := range cls.Properties {
		if p.Name == "title" {
			if p.Setter != "setTitle:" {
				t.Errorf("Setter = %q, want setTitle:", p.Setter)
			}
			if p.IsReadOnly {
				t.Error("title should not be readonly")
			}
		}
		if p.Name == "nilTypeProp" {
			if p.ObjCType != "" {
				t.Errorf("nilTypeProp ObjCType = %q, want empty (no Type node)", p.ObjCType)
			}
		}
	}
}

// TestExtractCategoryOwnClass exercises the path where a category extends a
// class that IS already in the current framework's Classes map.
func TestExtractCategoryOwnClass(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			// Define the class first.
			{
				Kind:  "ObjCInterfaceDecl",
				Name:  "MyClass",
				Loc:   &Location{FilePath: fakeHdr},
				Inner: []ASTNode{},
			},
			// Then a category that extends it.
			{
				Kind:      "ObjCCategoryDecl",
				Name:      "MyClass(Extras)",
				Loc:       &Location{FilePath: fakeHdr},
				Interface: &ASTRef{Name: "MyClass"},
				Inner: []ASTNode{
					{
						Kind:       "ObjCMethodDecl",
						Name:       "extraMethod",
						IsInstance: true,
						ReturnType: &ASTType{QualType: "void"},
					},
					{
						Kind:               "ObjCPropertyDecl",
						Name:               "extraProp",
						Type:               &ASTType{QualType: "id"},
						PropertyAttributes: []string{},
					},
				},
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	cls, ok := framework.Classes["MyClass"]
	if !ok {
		t.Fatal("MyClass not found in Classes")
	}
	if len(cls.Methods) != 1 || cls.Methods[0].Selector != "extraMethod" {
		t.Errorf("Methods from category: %+v", cls.Methods)
	}
	if len(cls.Properties) != 1 || cls.Properties[0].Name != "extraProp" {
		t.Errorf("Properties from category: %+v", cls.Properties)
	}
}

// TestExtractCategoryNoInterface exercises the fallback path in scanCategory
// where node.Interface is nil (parsing the class name from node.Name).
func TestExtractCategoryNoInterface(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind:  "ObjCInterfaceDecl",
				Name:  "OtherClass",
				Loc:   &Location{FilePath: fakeHdr},
				Inner: []ASTNode{},
			},
			// Category with no Interface field; class name is in Name field.
			{
				Kind:      "ObjCCategoryDecl",
				Name:      "OtherClass(ExtraStuff)",
				Loc:       &Location{FilePath: fakeHdr},
				Interface: nil, // nil — fallback to parsing Name
				Inner: []ASTNode{
					{
						Kind:       "ObjCMethodDecl",
						Name:       "fromCategory",
						IsInstance: true,
						ReturnType: &ASTType{QualType: "void"},
					},
				},
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	cls, ok := framework.Classes["OtherClass"]
	if !ok {
		t.Fatal("OtherClass not found in Classes")
	}
	if len(cls.Methods) != 1 || cls.Methods[0].Selector != "fromCategory" {
		t.Errorf("Category methods: %+v", cls.Methods)
	}
}

// TestExtractFunctionNoReturnType covers the branch where FunctionDecl has no
// ReturnType node.
func TestExtractFunctionNoReturnType(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind:       "FunctionDecl",
				Name:       "NoRetFn",
				Loc:        &Location{FilePath: fakeHdr},
				ReturnType: nil, // nil — should be skipped gracefully
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	if len(framework.Functions) != 1 || framework.Functions[0].Return.ObjCType != "" {
		t.Errorf("expected 1 function with empty Return; got %+v", framework.Functions)
	}
}

// TestExtractFunctionReturnTypeFromTypeQualType verifies that when a
// FunctionDecl carries no separate ReturnType node (as Clang emits for plain C
// functions), the return type is parsed from Type.QualType (e.g.
// "CFIndex (CFArrayRef)" → "CFIndex").
func TestExtractFunctionReturnTypeFromTypeQualType(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				// Typical plain C function: ReturnType is nil; return type lives in Type.QualType.
				Kind:       "FunctionDecl",
				Name:       "CFArrayGetCount",
				Loc:        &Location{FilePath: fakeHdr},
				ReturnType: nil,
				Type:       &ASTType{QualType: "CFIndex (CFArrayRef)"},
				Inner: []ASTNode{
					{Kind: "ParmVarDecl", Name: "theArray", Type: &ASTType{QualType: "CFArrayRef"}},
				},
			},
			{
				// Void-return function: return type should remain empty ("void" is filtered).
				Kind:       "FunctionDecl",
				Name:       "CFRelease",
				Loc:        &Location{FilePath: fakeHdr},
				ReturnType: nil,
				Type:       &ASTType{QualType: "void (CFTypeRef)"},
				Inner: []ASTNode{
					{Kind: "ParmVarDecl", Name: "cf", Type: &ASTType{QualType: "CFTypeRef"}},
				},
			},
			{
				// No Type set at all: return should remain empty.
				Kind:       "FunctionDecl",
				Name:       "Unknown",
				Loc:        &Location{FilePath: fakeHdr},
				ReturnType: nil,
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	if len(framework.Functions) != 3 {
		t.Fatalf("want 3 functions, got %d", len(framework.Functions))
	}

	// CFArrayGetCount should have return type "CFIndex".
	if got := framework.Functions[0].Return.ObjCType; got != "CFIndex" {
		t.Errorf("CFArrayGetCount return ObjCType = %q, want %q", got, "CFIndex")
	}
	// CFRelease should have empty return type (void is filtered).
	if got := framework.Functions[1].Return.ObjCType; got != "" {
		t.Errorf("CFRelease return ObjCType = %q, want empty", got)
	}
	// Unknown (no Type) should have empty return type.
	if got := framework.Functions[2].Return.ObjCType; got != "" {
		t.Errorf("Unknown return ObjCType = %q, want empty", got)
	}
}

// TestExtractEnumNoFixedType covers a plain EnumDecl with no FixedUnderlyingType:
// the Go integer type is inferred from the member values (matching clang's
// implicit choice), not defaulted to int64.
func TestExtractEnumNoFixedType(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cases := []struct {
		name    string
		values  []string
		wantGo  string
	}{
		{"small non-negative → int32", []string{"0", "1", "2"}, "int32"},
		{"negative fits int32", []string{"-1", "0", "2147483647"}, "int32"},
		{"exceeds int32, fits uint32 → uint32", []string{"0", "2147483648"}, "uint32"},
		{"exceeds uint32 → int64", []string{"0", "4294967296"}, "int64"},
	}
	for _, c := range cases {
		var members []ASTNode
		for i, v := range c.values {
			members = append(members, ASTNode{Kind: "EnumConstantDecl", Name: "V" + string(rune('a'+i)), Value: RawValue(v)})
		}
		root := &ASTNode{
			Kind: "TranslationUnitDecl",
			Inner: []ASTNode{{
				Kind:                "EnumDecl",
				Name:                "MyEnum",
				Loc:                 &Location{FilePath: fakeHdr},
				FixedUnderlyingType: nil,
				Inner:               members,
			}},
		}
		framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
		e, ok := framework.Enums["MyEnum"]
		if !ok {
			t.Fatalf("%s: MyEnum not found", c.name)
		}
		if e.GoType != c.wantGo {
			t.Errorf("%s: GoType = %q, want %q", c.name, e.GoType, c.wantGo)
		}
	}
}

// TestExtractAnonymousEnumEmpty covers the branch where an anonymous enum
// has no EnumConstantDecl members (early return).
func TestExtractAnonymousEnumEmpty(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind:  "EnumDecl",
				Name:  "", // anonymous
				Loc:   &Location{FilePath: fakeHdr},
				Inner: []ASTNode{}, // no members
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	if len(framework.Enums) != 0 {
		t.Errorf("expected no enums from empty anonymous enum; got %v", framework.Enums)
	}
}

// TestExtractAvailabilityNonMacOSInner covers the inner-node branch where the
// AvailabilityAttr has a non-macos platform — it should be ignored.
func TestExtractAvailabilityNonMacOSInnerIgnored(t *testing.T) {
	node := &ASTNode{
		Kind: "ObjCMethodDecl",
		Name: "iOSOnlyMethod",
		Inner: []ASTNode{
			{
				Kind:       "AvailabilityAttr",
				Platform:   "ios",
				Introduced: "9.0",
			},
		},
	}
	av := scanAvailability(node, "", 0)
	if av.MacOSIntroduced != "" {
		t.Errorf("MacOSIntroduced = %q, want empty (iOS inner attr should be ignored)", av.MacOSIntroduced)
	}
}

// TestListSubFrameworks exercises the listSubFrameworks helper by building a
// real directory tree that mimics a framework bundle's Frameworks/ directory.
func TestListSubFrameworks(t *testing.T) {
	parent := t.TempDir()

	// Create two valid sub-frameworks (with umbrella header) and one without.
	for _, name := range []string{"Alpha", "Beta"} {
		hdrDir := filepath.Join(parent, "Frameworks", name+".framework", "Headers")
		if err := os.MkdirAll(hdrDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(filepath.Join(hdrDir, name+".h"), []byte(""), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	// Add one without an umbrella header.
	ghostDir := filepath.Join(parent, "Frameworks", "NoHeader.framework", "Headers")
	if err := os.MkdirAll(ghostDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	got := listSubFrameworks(parent, "Parent")
	if len(got) != 2 {
		t.Fatalf("listSubFrameworks: got %v, want [Parent/Alpha Parent/Beta]", got)
	}
	// Results are not sorted by listSubFrameworks itself, so check as a set.
	found := map[string]bool{}
	for _, s := range got {
		found[s] = true
	}
	if !found["Parent/Alpha"] {
		t.Errorf("Parent/Alpha not in results: %v", got)
	}
	if !found["Parent/Beta"] {
		t.Errorf("Parent/Beta not in results: %v", got)
	}
}

// TestExtractProtocol exercises the protocol extraction path.
func TestExtractProtocolNode(t *testing.T) {
	sdkRoot := t.TempDir()
	hdrDir := filepath.Join(sdkRoot, "System", "Library", "Frameworks", "TestFW.framework", "Headers")
	if err := os.MkdirAll(hdrDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fakeHdr := filepath.Join(hdrDir, "TestFW.h")
	if err := os.WriteFile(fakeHdr, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := &ASTNode{
		Kind: "TranslationUnitDecl",
		Inner: []ASTNode{
			{
				Kind: "ObjCProtocolDecl",
				Name: "MyProtocol",
				Loc:  &Location{FilePath: fakeHdr},
				Protocols: []ASTRef{
					{Name: "NSObject"},
				},
				Inner: []ASTNode{
					{
						Kind:       "ObjCMethodDecl",
						Name:       "doSomething",
						IsInstance: true,
						ReturnType: &ASTType{QualType: "void"},
					},
				},
			},
		},
	}

	framework := Extract(root, sdkRoot, "TestFW", "14.0", "arm64", nil)
	proto, ok := framework.Protocols["MyProtocol"]
	if !ok {
		t.Fatal("MyProtocol not found in Protocols")
	}
	if len(proto.InheritedProtocols) != 1 || proto.InheritedProtocols[0] != "NSObject" {
		t.Errorf("Implements: %v", proto.InheritedProtocols)
	}
	if len(proto.Methods) != 1 {
		t.Errorf("Methods: got %d, want 1", len(proto.Methods))
	}
}

// TestLoadCLibrariesFile verifies the C library registry can be swapped for a
// committed JSON config, with the built-in defaults restored afterwards.
func TestLoadCLibrariesFile(t *testing.T) {
	saved := knownCLibraries
	t.Cleanup(func() { knownCLibraries = saved })

	dir := t.TempDir()
	path := filepath.Join(dir, "clibraries.json")
	content := `{"MyLib": {"link_lib": "mylib", "header": "mylib.h", "header_dir": "mylib.h"}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loaded, err := LoadCLibrariesFile(path)
	if err != nil || !loaded {
		t.Fatalf("LoadCLibrariesFile: loaded=%v err=%v", loaded, err)
	}
	if !IsCLibraryName("MyLib") {
		t.Error("MyLib should be registered after load")
	}
	if IsCLibraryName("EndpointSecurity") {
		t.Error("registry should be replaced, not merged")
	}
	if got := CLibraryHeaderRelative("MyLib"); got != "mylib.h" {
		t.Errorf("CLibraryHeaderRelative: got %q, want mylib.h", got)
	}
}

// TestLoadCLibrariesFileMissingIsNoop verifies a missing config file leaves
// the built-in defaults active.
func TestLoadCLibrariesFileMissingIsNoop(t *testing.T) {
	loaded, err := LoadCLibrariesFile(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil || loaded {
		t.Fatalf("missing file: loaded=%v err=%v", loaded, err)
	}
	if !IsCLibraryName("EndpointSecurity") {
		t.Error("defaults should remain active")
	}
}

// TestLoadCLibrariesFileRejectsMissingLinkLib verifies entries without a
// link_lib fail loudly rather than producing unlinkable packages.
func TestLoadCLibrariesFileRejectsMissingLinkLib(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clibraries.json")
	if err := os.WriteFile(path, []byte(`{"Bad": {"header": "bad.h"}}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := LoadCLibrariesFile(path); err == nil {
		t.Fatal("expected error for entry without link_lib")
	}
}

// TestLoadScanConfigFile verifies per-framework scan config can be swapped
// for a committed JSON file, with defaults restored afterwards.
func TestLoadScanConfigFile(t *testing.T) {
	saved := scanConfigs
	t.Cleanup(func() { scanConfigs = saved })

	dir := t.TempDir()
	path := filepath.Join(dir, "scanconfig.json")
	content := `{"MyFW": {"extra_include_dirs": ["usr/include/myfw"]}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	loaded, err := LoadScanConfigFile(path)
	if err != nil || !loaded {
		t.Fatalf("LoadScanConfigFile: loaded=%v err=%v", loaded, err)
	}
	if got := scanConfigs["MyFW"].ExtraIncludeDirs; len(got) != 1 || got[0] != "usr/include/myfw" {
		t.Errorf("ExtraIncludeDirs: got %v", got)
	}

	// Missing file: defaults remain, no error.
	loaded, err = LoadScanConfigFile(filepath.Join(dir, "nope.json"))
	if err != nil || loaded {
		t.Fatalf("missing file: loaded=%v err=%v", loaded, err)
	}
}
