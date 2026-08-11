package typemap

import (
	"strings"
	"testing"
)

// newBlockedCtx builds a Context where fromFw→toFw is a blocked import cycle.
// It mutates m.BlockedImports so the qualifier sees the cycle.
func newBlockedCtx(m *Mapper, fromFw, toFw string) Context {
	m.BlockedImports = map[string]map[string]bool{
		fromFw: {toFw: true},
	}
	return newTestCtx(m, fromFw)
}

// --- qualifiedType ---

func TestQualifiedTypeSameFramework(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	got := m.qualifiedType("NSString", "NSString", ctx, nil)
	if got != "*NSString" {
		t.Errorf("qualifiedType(same-fw) = %q, want *NSString", got)
	}
}

func TestQualifiedTypeCrossFramework(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "AppKit")
	imports := make(ImportSet)
	got := m.qualifiedType("NSString", "NSString", ctx, imports)
	if got != "*foundation.NSString" {
		t.Errorf("qualifiedType(cross-fw) = %q, want *foundation.NSString", got)
	}
	if imports["foundation"] == "" {
		t.Error("expected foundation in imports")
	}
}

func TestQualifiedTypeBlockedCycle(t *testing.T) {
	m := newTestMapper()
	ctx := newBlockedCtx(m, "AppKit", "Foundation")
	got := m.qualifiedType("NSString", "NSString", ctx, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("qualifiedType(cycle) = %q, want unsafe.Pointer", got)
	}
	if len(m.Diagnostics) == 0 {
		t.Error("expected diagnostic for cycle")
	}
}

func TestQualifiedTypeUnknownOwner(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "AppKit")
	// SomeUnknown has no entry in OwnerIndex → treated as same-framework.
	got := m.qualifiedType("SomeUnknown", "SomeUnknown", ctx, nil)
	if got != "*SomeUnknown" {
		t.Errorf("qualifiedType(unknown) = %q, want *SomeUnknown", got)
	}
}

// --- qualifiedEnumType ---

func TestQualifiedEnumTypeSameFramework(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	got := m.qualifiedEnumType("NSComparisonResult", "Foundation", ctx, nil)
	if got != "NSComparisonResult" {
		t.Errorf("qualifiedEnumType(same-fw) = %q, want NSComparisonResult", got)
	}
}

func TestQualifiedEnumTypeCrossFramework(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "AppKit")
	imports := make(ImportSet)
	got := m.qualifiedEnumType("NSComparisonResult", "Foundation", ctx, imports)
	if got != "foundation.NSComparisonResult" {
		t.Errorf("qualifiedEnumType(cross-fw) = %q, want foundation.NSComparisonResult", got)
	}
	if imports["foundation"] == "" {
		t.Error("expected foundation in imports")
	}
}

func TestQualifiedEnumTypeBlockedCycle(t *testing.T) {
	m := newTestMapper()
	ctx := newBlockedCtx(m, "AppKit", "Foundation")
	got := m.qualifiedEnumType("NSComparisonResult", "Foundation", ctx, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("qualifiedEnumType(cycle) = %q, want unsafe.Pointer", got)
	}
	if len(m.Diagnostics) == 0 {
		t.Error("expected diagnostic for cycle")
	}
}

func TestQualifiedEnumTypeUnknownOwner(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "AppKit")
	got := m.qualifiedEnumType("SomeEnum", "", ctx, nil)
	if got != "SomeEnum" {
		t.Errorf("qualifiedEnumType(unknown owner) = %q, want SomeEnum", got)
	}
}

// --- qualifiedStructType ---

func TestQualifiedStructTypeSameFramework(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	got := m.qualifiedStructType("NSRect", "Foundation", false, ctx, nil)
	if got != "NSRect" {
		t.Errorf("qualifiedStructType(same-fw, no ptr) = %q, want NSRect", got)
	}
}

func TestQualifiedStructTypeWithPointer(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	got := m.qualifiedStructType("NSRect", "Foundation", true, ctx, nil)
	if got != "*NSRect" {
		t.Errorf("qualifiedStructType(same-fw, ptr) = %q, want *NSRect", got)
	}
}

func TestQualifiedStructTypeCrossFramework(t *testing.T) {
	m := newTestMapper()
	// Add CGRect owned by CoreGraphics so we can test cross-fw.
	m.OwnerIndex["CGRect"] = "CoreGraphics"
	ctx := newTestCtx(m, "AppKit")
	imports := make(ImportSet)
	got := m.qualifiedStructType("CGRect", "CoreGraphics", false, ctx, imports)
	if got != "coregraphics.CGRect" {
		t.Errorf("qualifiedStructType(cross-fw) = %q, want coregraphics.CGRect", got)
	}
	if imports["coregraphics"] == "" {
		t.Error("expected coregraphics in imports")
	}
}

func TestQualifiedStructTypeBlockedCycle(t *testing.T) {
	m := newTestMapper()
	ctx := newBlockedCtx(m, "AppKit", "Foundation")
	got := m.qualifiedStructType("NSRect", "Foundation", false, ctx, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("qualifiedStructType(cycle) = %q, want unsafe.Pointer", got)
	}
	if len(m.Diagnostics) == 0 {
		t.Error("expected diagnostic for cycle")
	}
}

// --- qualifiedCFType ---

func TestQualifiedCFTypeSameFramework(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "CoreFoundation")
	got := m.qualifiedCFType("CFStringRef", ctx, nil)
	if got != "*CFStringRef" {
		t.Errorf("qualifiedCFType(CoreFoundation) = %q, want *CFStringRef", got)
	}
}

func TestQualifiedCFTypeCrossFramework(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	imports := make(ImportSet)
	got := m.qualifiedCFType("CFStringRef", ctx, imports)
	if got != "*corefoundation.CFStringRef" {
		t.Errorf("qualifiedCFType(cross-fw) = %q, want *corefoundation.CFStringRef", got)
	}
	if imports["corefoundation"] == "" {
		t.Error("expected corefoundation in imports")
	}
}

func TestQualifiedCFTypeBlockedCycle(t *testing.T) {
	m := newTestMapper()
	ctx := newBlockedCtx(m, "Foundation", "CoreFoundation")
	got := m.qualifiedCFType("CFStringRef", ctx, nil)
	if got != "unsafe.Pointer" {
		t.Errorf("qualifiedCFType(cycle) = %q, want unsafe.Pointer", got)
	}
	if len(m.Diagnostics) == 0 {
		t.Error("expected diagnostic for cycle")
	}
}

// --- qualifiedProtocolType ---

func TestQualifiedProtocolTypeSingleKnown(t *testing.T) {
	m := newTestMapper()
	m.ProtocolIndex = map[string]string{"NSCopying": "Foundation"}
	ctx := newTestCtx(m, "Foundation")
	got := m.qualifiedProtocolType([]string{"NSCopying"}, ctx, nil)
	if got != "NSCopying" {
		t.Errorf("qualifiedProtocolType(single, same-fw) = %q, want NSCopying", got)
	}
}

func TestQualifiedProtocolTypeCrossFramework(t *testing.T) {
	m := newTestMapper()
	m.ProtocolIndex = map[string]string{"NSCopying": "Foundation"}
	ctx := newTestCtx(m, "AppKit")
	got := m.qualifiedProtocolType([]string{"NSCopying"}, ctx, nil)
	if got != "foundation.NSCopying" {
		t.Errorf("qualifiedProtocolType(cross-fw) = %q, want foundation.NSCopying", got)
	}
}

func TestQualifiedProtocolTypeMultiple(t *testing.T) {
	m := newTestMapper()
	m.ProtocolIndex = map[string]string{
		"NSCopying":   "Foundation",
		"NSCacheItem": "Foundation",
	}
	ctx := newTestCtx(m, "Foundation")
	got := m.qualifiedProtocolType([]string{"NSCopying", "NSCacheItem"}, ctx, nil)
	if !strings.HasPrefix(got, "interface {") {
		t.Errorf("qualifiedProtocolType(multi) = %q, want inline interface", got)
	}
}

func TestQualifiedProtocolTypeUnregistered(t *testing.T) {
	m := newTestMapper()
	m.ProtocolIndex = map[string]string{}
	ctx := newTestCtx(m, "Foundation")
	got := m.qualifiedProtocolType([]string{"UnknownProto"}, ctx, nil)
	if got != "" {
		t.Errorf("qualifiedProtocolType(unregistered) = %q, want empty", got)
	}
}

// --- BlockedImportNote ---

func TestBlockedImportNoteBlocked(t *testing.T) {
	m := newTestMapper()
	ctx := newBlockedCtx(m, "AppKit", "Foundation")
	note := m.BlockedImportNote("NSString *", ctx)
	if note == "" {
		t.Error("BlockedImportNote: expected non-empty for blocked cycle")
	}
	if !strings.Contains(note, "import cycle") {
		t.Errorf("BlockedImportNote: got %q, want 'import cycle' in message", note)
	}
}

func TestBlockedImportNoteNotBlocked(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "AppKit")
	if note := m.BlockedImportNote("NSString *", ctx); note != "" {
		t.Errorf("BlockedImportNote(not blocked) = %q, want empty", note)
	}
}

func TestBlockedImportNoteSameFramework(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "Foundation")
	if note := m.BlockedImportNote("NSString *", ctx); note != "" {
		t.Errorf("BlockedImportNote(same-fw) = %q, want empty", note)
	}
}

func TestBlockedImportNoteNonClass(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "AppKit")
	if note := m.BlockedImportNote("int64_t", ctx); note != "" {
		t.Errorf("BlockedImportNote(non-class) = %q, want empty", note)
	}
}

// --- capitaliseFirst ---

func TestCapitaliseFirstLower(t *testing.T) {
	if got := capitaliseFirst("hello"); got != "Hello" {
		t.Errorf("capitaliseFirst(hello) = %q, want Hello", got)
	}
}

func TestCapitaliseFirstAlreadyUpper(t *testing.T) {
	if got := capitaliseFirst("Hello"); got != "Hello" {
		t.Errorf("capitaliseFirst(Hello) = %q, want Hello", got)
	}
}

func TestCapitaliseFirstEmpty(t *testing.T) {
	if got := capitaliseFirst(""); got != "" {
		t.Errorf("capitaliseFirst('') = %q, want empty", got)
	}
}

func TestCapitaliseFirstSingleChar(t *testing.T) {
	if got := capitaliseFirst("a"); got != "A" {
		t.Errorf("capitaliseFirst(a) = %q, want A", got)
	}
}

func TestCapitaliseFirstNonAlpha(t *testing.T) {
	if got := capitaliseFirst("_foo"); got != "_foo" {
		t.Errorf("capitaliseFirst(_foo) = %q, want _foo", got)
	}
}

// --- qualifier struct direct tests (independently of Mapper) ---

func newTestQualifier(fw, modulePrefix string, owner map[string]string, blocked map[string]map[string]bool) qualifier {
	diags := make([]string, 0)
	return qualifier{
		framework:      fw,
		modulePrefix:   modulePrefix,
		frameworkOwner: owner,
		blockedImports: blocked,
		usedImports:    make(map[string]string),
		diagnostics:    &diags,
	}
}

func TestQualifierClassTypeSameFramework(t *testing.T) {
	q := newTestQualifier("Foundation", "example.com/fw", map[string]string{"NSString": "Foundation"}, nil)
	if got := q.classType("NSString", "NSString"); got != "*NSString" {
		t.Errorf("classType(same-fw) = %q, want *NSString", got)
	}
}

func TestQualifierClassTypeCrossFramework(t *testing.T) {
	q := newTestQualifier("AppKit", "example.com/fw", map[string]string{"NSString": "Foundation"}, nil)
	if got := q.classType("NSString", "NSString"); got != "*foundation.NSString" {
		t.Errorf("classType(cross-fw) = %q, want *foundation.NSString", got)
	}
	if q.usedImports["foundation"] == "" {
		t.Error("expected foundation in usedImports")
	}
}

func TestQualifierClassTypeBlocked(t *testing.T) {
	blocked := map[string]map[string]bool{"AppKit": {"Foundation": true}}
	q := newTestQualifier("AppKit", "example.com/fw", map[string]string{"NSString": "Foundation"}, blocked)
	if got := q.classType("NSString", "NSString"); got != "unsafe.Pointer" {
		t.Errorf("classType(blocked) = %q, want unsafe.Pointer", got)
	}
	if len(*q.diagnostics) == 0 {
		t.Error("expected diagnostic for cycle")
	}
}

func TestQualifierEnumTypeSameFramework(t *testing.T) {
	q := newTestQualifier("Foundation", "example.com/fw", nil, nil)
	if got := q.enumType("NSComparisonResult", "Foundation"); got != "NSComparisonResult" {
		t.Errorf("enumType(same-fw) = %q, want NSComparisonResult", got)
	}
}

func TestQualifierEnumTypeCrossFramework(t *testing.T) {
	q := newTestQualifier("AppKit", "example.com/fw", nil, nil)
	if got := q.enumType("NSComparisonResult", "Foundation"); got != "foundation.NSComparisonResult" {
		t.Errorf("enumType(cross-fw) = %q, want foundation.NSComparisonResult", got)
	}
}

// TestQualifierEnumTypeLocalDeclaration covers the local-declaration
// preference: an enum owned (by the global index) elsewhere but ALSO declared
// by the current framework resolves to the local copy, not a cross-package
// import (xpc_listener_create_flags_t declared by both xpc and oslog).
func TestQualifierEnumTypeLocalDeclaration(t *testing.T) {
	q := newTestQualifier("xpc", "example.com/fw", nil, nil)
	q.localEnums = map[string]bool{"xpc_listener_create_flags_t": true}
	got := q.enumType("xpc_listener_create_flags_t", "oslog")
	if got != "XpcListenerCreateFlagsT" {
		t.Errorf("enumType(local-decl) = %q, want XpcListenerCreateFlagsT (no oslog. qualifier)", got)
	}
	if q.usedImports["oslog"] != "" {
		t.Error("local enum must not import the global owner package")
	}
}

func TestQualifierStructTypeSameFramework(t *testing.T) {
	q := newTestQualifier("Foundation", "example.com/fw", nil, nil)
	if got := q.structType("NSRect", "Foundation", false); got != "NSRect" {
		t.Errorf("structType(same-fw) = %q, want NSRect", got)
	}
}

func TestQualifierStructTypeWithPointer(t *testing.T) {
	q := newTestQualifier("Foundation", "example.com/fw", nil, nil)
	if got := q.structType("NSRect", "Foundation", true); got != "*NSRect" {
		t.Errorf("structType(with ptr) = %q, want *NSRect", got)
	}
}

func TestQualifierCFTypeSameFramework(t *testing.T) {
	q := newTestQualifier("CoreFoundation", "example.com/fw", nil, nil)
	if got := q.cfType("CFStringRef"); got != "*CFStringRef" {
		t.Errorf("cfType(CoreFoundation) = %q, want *CFStringRef", got)
	}
}

func TestQualifierCFTypeCrossFramework(t *testing.T) {
	q := newTestQualifier("Foundation", "example.com/fw", nil, nil)
	if got := q.cfType("CFStringRef"); got != "*corefoundation.CFStringRef" {
		t.Errorf("cfType(cross-fw) = %q, want *corefoundation.CFStringRef", got)
	}
	if q.usedImports["corefoundation"] == "" {
		t.Error("expected corefoundation in usedImports")
	}
}

func TestQualifierProtocolTypeSingle(t *testing.T) {
	// frameworkOwner must NOT contain NSCopying as a class — otherwise
	// naming.ProtocolGoTypeName adds a "Protocol" disambiguation suffix.
	q := newTestQualifier("Foundation", "example.com/fw", nil, nil)
	q.knownProtocols = map[string]string{"NSCopying": "Foundation"}
	if got := q.protocolType([]string{"NSCopying"}); got != "NSCopying" {
		t.Errorf("protocolType(single, same-fw) = %q, want NSCopying", got)
	}
}

func TestQualifierProtocolTypeBlockedDropsConstraint(t *testing.T) {
	blocked := map[string]map[string]bool{"AppKit": {"Foundation": true}}
	q := newTestQualifier("AppKit", "example.com/fw", nil, blocked)
	q.knownProtocols = map[string]string{"NSCopying": "Foundation"}
	// Blocked cycle → protocolGoName returns "" → constraint dropped → empty result
	if got := q.protocolType([]string{"NSCopying"}); got != "" {
		t.Errorf("protocolType(blocked) = %q, want empty", got)
	}
}
