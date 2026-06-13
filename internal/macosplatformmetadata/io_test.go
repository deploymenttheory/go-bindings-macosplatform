package macosplatformmetadata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// minimalFM returns a FrameworkMeta with all required scalar fields set.
func minimalFM() *FrameworkMeta {
	return &FrameworkMeta{
		Framework:  "Foundation",
		SDKVersion: "14.0",
		Arch:       "arm64",
	}
}

// TestWriteAndReadRoundTrip writes a populated FrameworkMeta and reads it back,
// verifying every top-level field is preserved exactly.
func TestWriteAndReadRoundTrip(t *testing.T) {
	framework := &FrameworkMeta{
		Framework:       "Foundation",
		SDKVersion:      "14.0",
		Arch:            "arm64",
		IsSwiftOnly:       false,
		ParentFramework: "",
		Classes: map[string]Class{
			"NSObject": {
				Super:     "",
				Protocols: []string{"NSCopying"},
				Methods: []Method{
					{
						Selector:      "alloc",
						IsClassMethod: true,
						Return:        ReturnType{ObjCType: "id"},
					},
				},
			},
		},
		Protocols: map[string]Protocol{
			"NSCopying": {
				Methods: []Method{
					{Selector: "copyWithZone:", Return: ReturnType{ObjCType: "id"}},
				},
			},
		},
		Enums: map[string]Enum{
			"NSComparisonResult": {
				GoType: "int64",
				Members: []EnumMember{
					{Name: "NSOrderedAscending", Value: "-1"},
				},
			},
		},
		Structs: map[string]Struct{
			"NSRange": {
				Fields: []StructField{
					{Name: "location", ObjCType: "NSUInteger", GoType: "uint"},
					{Name: "length", ObjCType: "NSUInteger", GoType: "uint"},
				},
			},
		},
		Functions: []Function{
			{
				Name:   "NSMakeRange",
				Return: ReturnType{ObjCType: "NSRange"},
				Params: []Param{
					{Name: "loc", ObjCType: "NSUInteger"},
				},
			},
		},
		Externs: []Extern{
			{Name: "NSFoundationVersionNumber", ObjCType: "double", GoType: "float64"},
		},
		BlockTypes: map[string]BlockType{
			"CompletionBlock": {
				Params:   []Param{{Name: "error", ObjCType: "NSError *"}},
				Return: ReturnType{ObjCType: "void"},
			},
		},
		Typedefs: map[string]string{
			"NSTimeInterval": "double",
		},
		ForeignExtensions: map[string][]Method{
			"NSBundle": {
				{Selector: "loadAppleScriptObjectiveCScripts", Return: ReturnType{ObjCType: "void"}},
			},
		},
	}

	dir := t.TempDir()
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}

	path := filepath.Join(dir, "Foundation-arm64-14.0.gometa.json")
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if got.Framework != framework.Framework {
		t.Errorf("Framework: got %q, want %q", got.Framework, framework.Framework)
	}
	if got.SDKVersion != framework.SDKVersion {
		t.Errorf("SDKVersion: got %q, want %q", got.SDKVersion, framework.SDKVersion)
	}
	if got.Arch != framework.Arch {
		t.Errorf("Arch: got %q, want %q", got.Arch, framework.Arch)
	}
	if len(got.Classes) != 1 {
		t.Errorf("Classes: got %d, want 1", len(got.Classes))
	}
	if len(got.Protocols) != 1 {
		t.Errorf("Protocols: got %d, want 1", len(got.Protocols))
	}
	if len(got.Enums) != 1 {
		t.Errorf("Enums: got %d, want 1", len(got.Enums))
	}
	if len(got.Structs) != 1 {
		t.Errorf("Structs: got %d, want 1", len(got.Structs))
	}
	if len(got.Functions) != 1 {
		t.Errorf("Functions: got %d, want 1", len(got.Functions))
	}
	if len(got.Externs) != 1 {
		t.Errorf("Externs: got %d, want 1", len(got.Externs))
	}
	if len(got.BlockTypes) != 1 {
		t.Errorf("BlockTypes: got %d, want 1", len(got.BlockTypes))
	}
	if len(got.Typedefs) != 1 {
		t.Errorf("Typedefs: got %d, want 1", len(got.Typedefs))
	}
	if len(got.ForeignExtensions) != 1 {
		t.Errorf("ForeignExtensions: got %d, want 1", len(got.ForeignExtensions))
	}
}

// TestWriteCreatesDirectory verifies that Write creates the output directory
// when it does not already exist.
func TestWriteCreatesDirectory(t *testing.T) {
	base := t.TempDir()
	newDir := filepath.Join(base, "deeply", "nested", "dir")

	framework := minimalFM()
	if err := Write(framework, newDir); err != nil {
		t.Fatalf("Write to non-existent dir: %v", err)
	}

	if _, err := os.Stat(newDir); err != nil {
		t.Errorf("expected directory %s to exist after Write, but stat failed: %v", newDir, err)
	}
}

// TestWriteFileName verifies that the file produced by Write follows the
// pattern {framework}-{arch}-{sdk}.gometa.json.
func TestWriteFileName(t *testing.T) {
	dir := t.TempDir()
	framework := &FrameworkMeta{
		Framework:  "AppKit",
		Arch:       "arm64",
		SDKVersion: "15.2",
	}
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}

	want := "AppKit-arm64-15.2.gometa.json"
	path := filepath.Join(dir, want)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file %q to exist; stat error: %v", want, err)
	}

	// Confirm no other .json files were created.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected exactly 1 file, got %d", len(entries))
	}
}

// TestReadNonexistentFile verifies that Read returns an error for a path that
// does not exist.
func TestReadNonexistentFile(t *testing.T) {
	_, err := Read("/no/such/file/nope.gometa.json")
	if err == nil {
		t.Fatal("expected error reading non-existent file, got nil")
	}
}

// TestReadInvalidJSON verifies that Read returns a wrapped error when the file
// contains invalid JSON.
func TestReadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.gometa.json")
	if err := os.WriteFile(path, []byte("{this is not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Read(path)
	if err == nil {
		t.Fatal("expected error reading invalid JSON, got nil")
	}
	// The error must contain a useful description.
	if !strings.Contains(err.Error(), "parsing") {
		t.Errorf("error message should mention 'parsing'; got: %v", err)
	}
}

// TestWritePreservesClasses verifies that a class with methods, arguments, and
// properties survives a round-trip intact.
func TestWritePreservesClasses(t *testing.T) {
	framework := minimalFM()
	framework.Classes = map[string]Class{
		"NSString": {
			Super:     "NSObject",
			Protocols: []string{"NSCopying", "NSSecureCoding"},
			Methods: []Method{
				{
					Selector:      "stringWithFormat:",
					IsClassMethod: true,
					IsVariadic:    true,
					Params: []Param{
						{Name: "format", ObjCType: "NSString *"},
					},
					Return: ReturnType{ObjCType: "NSString *", IsInstancetype: true},
				},
				{
					Selector: "length",
					Return:   ReturnType{ObjCType: "NSUInteger"},
				},
			},
			Properties: []Property{
				{Name: "length", ObjCType: "NSUInteger", IsReadOnly: true},
			},
		},
	}

	dir := t.TempDir()
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(filepath.Join(dir, "Foundation-arm64-14.0.gometa.json"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	cls, ok := got.Classes["NSString"]
	if !ok {
		t.Fatal("NSString class missing after round-trip")
	}
	if cls.Super != "NSObject" {
		t.Errorf("Super: got %q, want %q", cls.Super, "NSObject")
	}
	if len(cls.Protocols) != 2 {
		t.Errorf("Protocols: got %d, want 2", len(cls.Protocols))
	}
	if len(cls.Methods) != 2 {
		t.Errorf("Methods: got %d, want 2", len(cls.Methods))
	}
	m0 := cls.Methods[0]
	if !m0.IsClassMethod {
		t.Errorf("IsClassMethod not preserved for %s", m0.Selector)
	}
	if !m0.IsVariadic {
		t.Errorf("IsVariadic not preserved for %s", m0.Selector)
	}
	if len(m0.Params) != 1 || m0.Params[0].ObjCType != "NSString *" {
		t.Errorf("Arg ObjCType not preserved; got %+v", m0.Params)
	}
	if len(cls.Properties) != 1 || !cls.Properties[0].IsReadOnly {
		t.Errorf("Property Readonly not preserved; got %+v", cls.Properties)
	}
}

// TestWritePreservesProtocols verifies that a protocol with methods survives a
// round-trip.
func TestWritePreservesProtocols(t *testing.T) {
	framework := minimalFM()
	framework.Protocols = map[string]Protocol{
		"NSCopying": {
			InheritedProtocols: []string{"NSObject"},
			Methods: []Method{
				{
					Selector: "copyWithZone:",
					Params:     []Param{{Name: "zone", ObjCType: "NSZone *"}},
					Return:   ReturnType{ObjCType: "id", IsNullable: true},
				},
			},
			Availability: Availability{MacOSIntroduced: "10.0"},
		},
	}

	dir := t.TempDir()
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(filepath.Join(dir, "Foundation-arm64-14.0.gometa.json"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	proto, ok := got.Protocols["NSCopying"]
	if !ok {
		t.Fatal("NSCopying protocol missing after round-trip")
	}
	if len(proto.InheritedProtocols) != 1 || proto.InheritedProtocols[0] != "NSObject" {
		t.Errorf("Implements: got %v, want [NSObject]", proto.InheritedProtocols)
	}
	if len(proto.Methods) != 1 {
		t.Errorf("Methods: got %d, want 1", len(proto.Methods))
	}
	if proto.Methods[0].Return.IsNullable != true {
		t.Errorf("Return.IsNullable not preserved")
	}
	if proto.Availability.MacOSIntroduced != "10.0" {
		t.Errorf("Availability.MacOSIntroduced: got %q, want %q", proto.Availability.MacOSIntroduced, "10.0")
	}
}

// TestWritePreservesEnums verifies that enum types with members survive a
// round-trip.
func TestWritePreservesEnums(t *testing.T) {
	framework := minimalFM()
	framework.Enums = map[string]Enum{
		"NSComparisonResult": {
			GoType: "int64",
			Members: []EnumMember{
				{Name: "NSOrderedAscending", Value: "-1"},
				{Name: "NSOrderedSame", Value: "0"},
				{Name: "NSOrderedDescending", Value: "1"},
			},
			Availability: Availability{MacOSIntroduced: "10.0"},
		},
		"__anon_consts": {
			GoType: "uint64",
			IsAnon: true,
			Members: []EnumMember{
				{Name: "NSNotFound", Value: "9223372036854775807"},
			},
		},
	}

	dir := t.TempDir()
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(filepath.Join(dir, "Foundation-arm64-14.0.gometa.json"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	e, ok := got.Enums["NSComparisonResult"]
	if !ok {
		t.Fatal("NSComparisonResult enum missing after round-trip")
	}
	if e.GoType != "int64" {
		t.Errorf("GoType: got %q, want int64", e.GoType)
	}
	if len(e.Members) != 3 {
		t.Errorf("Members: got %d, want 3", len(e.Members))
	}
	if e.Availability.MacOSIntroduced != "10.0" {
		t.Errorf("Availability not preserved")
	}

	anon, ok := got.Enums["__anon_consts"]
	if !ok {
		t.Fatal("__anon_consts enum missing after round-trip")
	}
	if !anon.IsAnon {
		t.Errorf("IsAnon not preserved")
	}
}

// TestWritePreservesStructs verifies that struct field details survive a
// round-trip.
func TestWritePreservesStructs(t *testing.T) {
	framework := minimalFM()
	framework.Structs = map[string]Struct{
		"NSRange": {
			Fields: []StructField{
				{Name: "location", ObjCType: "NSUInteger", GoType: "uint"},
				{Name: "length", ObjCType: "NSUInteger", GoType: "uint"},
			},
			Availability: Availability{MacOSIntroduced: "10.0"},
		},
		"NSPoint": {
			Fields: []StructField{
				{Name: "x", ObjCType: "CGFloat", GoType: "float64"},
				{Name: "y", ObjCType: "CGFloat", GoType: "float64"},
			},
		},
	}

	dir := t.TempDir()
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(filepath.Join(dir, "Foundation-arm64-14.0.gometa.json"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	s, ok := got.Structs["NSRange"]
	if !ok {
		t.Fatal("NSRange struct missing after round-trip")
	}
	if len(s.Fields) != 2 {
		t.Errorf("Fields: got %d, want 2", len(s.Fields))
	}
	if s.Fields[0].Name != "location" || s.Fields[0].GoType != "uint" {
		t.Errorf("Field[0] not preserved: %+v", s.Fields[0])
	}
	if s.Availability.MacOSIntroduced != "10.0" {
		t.Errorf("Availability not preserved")
	}

	if _, ok := got.Structs["NSPoint"]; !ok {
		t.Error("NSPoint struct missing after round-trip")
	}
}

// TestWritePreservesFunctions verifies that the Functions slice survives a
// round-trip with all fields intact.
func TestWritePreservesFunctions(t *testing.T) {
	framework := minimalFM()
	framework.Functions = []Function{
		{
			Name: "NSMakeRange",
			Params: []Param{
				{Name: "loc", ObjCType: "NSUInteger"},
				{Name: "len", ObjCType: "NSUInteger"},
			},
			Return:   ReturnType{ObjCType: "NSRange"},
			IsInline: false,
		},
		{
			Name:     "NSLog",
			Params:     []Param{{Name: "format", ObjCType: "NSString *"}},
			Return:   ReturnType{ObjCType: "void"},
			IsInline: false,
		},
	}

	dir := t.TempDir()
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(filepath.Join(dir, "Foundation-arm64-14.0.gometa.json"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if len(got.Functions) != 2 {
		t.Fatalf("Functions: got %d, want 2", len(got.Functions))
	}
	fn := got.Functions[0]
	if fn.Name != "NSMakeRange" {
		t.Errorf("Function[0].Name: got %q, want NSMakeRange", fn.Name)
	}
	if len(fn.Params) != 2 {
		t.Errorf("Function[0].Params: got %d, want 2", len(fn.Params))
	}
	if fn.Params[0].ObjCType != "NSUInteger" {
		t.Errorf("Function[0].Params[0].ObjCType: got %q, want NSUInteger", fn.Params[0].ObjCType)
	}
}

// TestWritePreservesExterns verifies that the Externs slice survives a
// round-trip.
func TestWritePreservesExterns(t *testing.T) {
	framework := minimalFM()
	framework.Externs = []Extern{
		{Name: "NSFoundationVersionNumber", ObjCType: "double", GoType: "float64"},
		{Name: "NSBundleDidLoadNotification", ObjCType: "NSNotificationName", GoType: "string"},
	}

	dir := t.TempDir()
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(filepath.Join(dir, "Foundation-arm64-14.0.gometa.json"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if len(got.Externs) != 2 {
		t.Fatalf("Externs: got %d, want 2", len(got.Externs))
	}
	// Verify specific fields on the first extern.
	e0 := got.Externs[0]
	if e0.Name != "NSFoundationVersionNumber" {
		t.Errorf("Externs[0].Name: got %q, want NSFoundationVersionNumber", e0.Name)
	}
	if e0.GoType != "float64" {
		t.Errorf("Externs[0].GoType: got %q, want float64", e0.GoType)
	}
}

// TestWritePreservesBlockTypes verifies that BlockType entries (including their
// argument and return-type details) survive a round-trip.
func TestWritePreservesBlockTypes(t *testing.T) {
	framework := minimalFM()
	framework.BlockTypes = map[string]BlockType{
		"VoidErrorBlock": {
			Params: []Param{
				{Name: "error", ObjCType: "NSError *", IsNullable: true},
			},
			Return: ReturnType{ObjCType: "void"},
		},
		"BoolBlock": {
			Params:   []Param{},
			Return: ReturnType{ObjCType: "BOOL"},
		},
	}

	dir := t.TempDir()
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(filepath.Join(dir, "Foundation-arm64-14.0.gometa.json"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if len(got.BlockTypes) != 2 {
		t.Fatalf("BlockTypes: got %d, want 2", len(got.BlockTypes))
	}
	bt, ok := got.BlockTypes["VoidErrorBlock"]
	if !ok {
		t.Fatal("VoidErrorBlock missing after round-trip")
	}
	if len(bt.Params) != 1 || !bt.Params[0].IsNullable {
		t.Errorf("BlockType arg Nullable not preserved: %+v", bt.Params)
	}
	if bt.Return.ObjCType != "void" {
		t.Errorf("BlockType Return.ObjCType: got %q, want void", bt.Return.ObjCType)
	}
}

// TestWritePreservesTypedefs verifies that the Typedefs map[string]string
// survives a round-trip with all entries intact.
func TestWritePreservesTypedefs(t *testing.T) {
	framework := minimalFM()
	framework.Typedefs = map[string]string{
		"NSTimeInterval": "double",
		"NSInteger":      "long",
		"NSUInteger":     "unsigned long",
	}

	dir := t.TempDir()
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(filepath.Join(dir, "Foundation-arm64-14.0.gometa.json"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if len(got.Typedefs) != 3 {
		t.Fatalf("Typedefs: got %d, want 3", len(got.Typedefs))
	}
	for k, wantV := range framework.Typedefs {
		if gotV := got.Typedefs[k]; gotV != wantV {
			t.Errorf("Typedefs[%q]: got %q, want %q", k, gotV, wantV)
		}
	}
}

// TestWritePreservesForeignExtensions verifies that the ForeignExtensions map
// (class name → []Method) survives a round-trip with method details preserved.
func TestWritePreservesForeignExtensions(t *testing.T) {
	framework := minimalFM()
	framework.ForeignExtensions = map[string][]Method{
		"NSBundle": {
			{
				Selector: "loadAppleScriptObjectiveCScripts",
				Return:   ReturnType{ObjCType: "void"},
			},
		},
		"NSObject": {
			{
				Selector:      "scriptingProperties",
				IsClassMethod: false,
				Return:        ReturnType{ObjCType: "NSDictionary *"},
			},
		},
	}

	dir := t.TempDir()
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(filepath.Join(dir, "Foundation-arm64-14.0.gometa.json"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if len(got.ForeignExtensions) != 2 {
		t.Fatalf("ForeignExtensions: got %d, want 2", len(got.ForeignExtensions))
	}
	methods, ok := got.ForeignExtensions["NSBundle"]
	if !ok {
		t.Fatal("NSBundle ForeignExtensions entry missing after round-trip")
	}
	if len(methods) != 1 || methods[0].Selector != "loadAppleScriptObjectiveCScripts" {
		t.Errorf("ForeignExtensions[NSBundle] not preserved: %+v", methods)
	}
}

// TestWritePreservesParentFramework verifies that the ParentFramework field is
// preserved across a round-trip.
func TestWritePreservesParentFramework(t *testing.T) {
	framework := &FrameworkMeta{
		Framework:       "HIToolbox",
		Arch:            "arm64",
		SDKVersion:      "14.0",
		ParentFramework: "Carbon",
	}

	dir := t.TempDir()
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(filepath.Join(dir, "HIToolbox-arm64-14.0.gometa.json"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if got.ParentFramework != "Carbon" {
		t.Errorf("ParentFramework: got %q, want Carbon", got.ParentFramework)
	}
	if got.Framework != "HIToolbox" {
		t.Errorf("Framework: got %q, want HIToolbox", got.Framework)
	}
}

// TestWritePreservesSwiftOnly verifies that SwiftOnly=true survives a
// round-trip; omitempty must not strip it.
func TestWritePreservesSwiftOnly(t *testing.T) {
	framework := &FrameworkMeta{
		Framework:  "RealityKit",
		Arch:       "arm64",
		SDKVersion: "17.0",
		IsSwiftOnly:  true,
	}

	dir := t.TempDir()
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(filepath.Join(dir, "RealityKit-arm64-17.0.gometa.json"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if !got.IsSwiftOnly {
		t.Errorf("IsSwiftOnly: got false, want true")
	}
}

// TestWriteErrorOnUnwritablePath verifies that Write returns an error when the
// output file cannot be created because the target directory path is already
// occupied by a regular file (making MkdirAll fail on the leaf component).
func TestWriteErrorOnUnwritablePath(t *testing.T) {
	base := t.TempDir()

	// Create a regular file where Write expects to create a directory.
	blocker := filepath.Join(base, "notadir")
	if err := os.WriteFile(blocker, []byte("blocker"), 0o644); err != nil {
		t.Fatalf("setup: WriteFile: %v", err)
	}

	// Ask Write to use blocker/subdir as outDir — MkdirAll must fail because
	// blocker is a file, not a directory.
	outDir := filepath.Join(blocker, "subdir")
	framework := minimalFM()
	err := Write(framework, outDir)
	if err == nil {
		t.Fatal("expected error from Write when outDir path is unwritable, got nil")
	}
}

// TestWriteErrorOnReadOnlyDir verifies that Write returns an error when the
// output directory exists but is not writable (WriteFile fails).
func TestWriteErrorOnReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root — permission checks do not apply")
	}
	dir := t.TempDir()
	// Remove write permission so os.WriteFile fails.
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	framework := minimalFM()
	err := Write(framework, dir)
	if err == nil {
		t.Fatal("expected error from Write on read-only directory, got nil")
	}
}

// TestReadWriteAvailability verifies that all Availability struct fields are
// preserved through a round-trip, using availability embedded in a Class.
func TestReadWriteAvailability(t *testing.T) {
	avail := Availability{
		MacOSIntroduced: "10.12",
		MacOSDeprecated: "14.0",
		DeprecationMsg:  "use NewAPI instead",
		ReplacedBy:      "NewClass",
	}
	framework := minimalFM()
	framework.Classes = map[string]Class{
		"OldClass": {
			Availability: avail,
		},
	}
	framework.Enums = map[string]Enum{
		"OldEnum": {
			GoType:       "int64",
			Availability: avail,
			Members: []EnumMember{
				{
					Name:         "OldEnumVal",
					Value:        "0",
					Availability: avail,
				},
			},
		},
	}

	dir := t.TempDir()
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(filepath.Join(dir, "Foundation-arm64-14.0.gometa.json"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	cls := got.Classes["OldClass"]
	checkAvail := func(label string, a Availability) {
		t.Helper()
		if a.MacOSIntroduced != avail.MacOSIntroduced {
			t.Errorf("%s MacOSIntroduced: got %q, want %q", label, a.MacOSIntroduced, avail.MacOSIntroduced)
		}
		if a.MacOSDeprecated != avail.MacOSDeprecated {
			t.Errorf("%s MacOSDeprecated: got %q, want %q", label, a.MacOSDeprecated, avail.MacOSDeprecated)
		}
		if a.DeprecationMsg != avail.DeprecationMsg {
			t.Errorf("%s DeprecationMsg: got %q, want %q", label, a.DeprecationMsg, avail.DeprecationMsg)
		}
		if a.ReplacedBy != avail.ReplacedBy {
			t.Errorf("%s ReplacedBy: got %q, want %q", label, a.ReplacedBy, avail.ReplacedBy)
		}
	}
	checkAvail("Class", cls.Availability)

	enum := got.Enums["OldEnum"]
	checkAvail("Enum", enum.Availability)
	if len(enum.Members) != 1 {
		t.Fatalf("expected 1 enum member, got %d", len(enum.Members))
	}
	checkAvail("EnumMember", enum.Members[0].Availability)
}

// TestWriteStampsSchemaVersion verifies that Write stamps CurrentSchemaVersion
// regardless of what the caller set, and that the value survives a round-trip.
func TestWriteStampsSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	framework := minimalFM()
	framework.SchemaVersion = 0 // Write must overwrite this with the current version.

	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(filepath.Join(dir, "Foundation-arm64-14.0.gometa.json"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("SchemaVersion: got %d, want %d", got.SchemaVersion, CurrentSchemaVersion)
	}
}

// TestWritePreservesToolchainProvenance verifies that ClangVersion and
// XcodeVersion survive a round-trip.
func TestWritePreservesToolchainProvenance(t *testing.T) {
	dir := t.TempDir()
	framework := minimalFM()
	framework.ClangVersion = "Apple clang version 21.0.0 (clang-2100.3.9.2)"
	framework.XcodeVersion = "Xcode 26.0 Build version 17A321"

	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(filepath.Join(dir, "Foundation-arm64-14.0.gometa.json"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.ClangVersion != framework.ClangVersion {
		t.Errorf("ClangVersion: got %q, want %q", got.ClangVersion, framework.ClangVersion)
	}
	if got.XcodeVersion != framework.XcodeVersion {
		t.Errorf("XcodeVersion: got %q, want %q", got.XcodeVersion, framework.XcodeVersion)
	}
}

// TestReadRejectsFutureSchemaVersion verifies that Read refuses metadata
// written by a newer generator generation.
func TestReadRejectsFutureSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Future-arm64-99.0.gometa.json")
	content := []byte(`{"framework":"Future","sdk_version":"99.0","arch":"arm64","schema_version":2147483647}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Read(path)
	if err == nil {
		t.Fatal("expected error reading future schema version, got nil")
	}
	if !strings.Contains(err.Error(), "newer than this generator supports") {
		t.Errorf("error should mention unsupported newer version; got: %v", err)
	}
}

// TestReadAcceptsLegacyUnversionedMetadata verifies that metadata written
// before the schema_version field existed (deserialising to 0) is accepted
// while MinSupportedSchemaVersion remains 0.
func TestReadAcceptsLegacyUnversionedMetadata(t *testing.T) {
	if MinSupportedSchemaVersion > 0 {
		t.Skip("legacy unversioned metadata is no longer within the supported window")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "Legacy-arm64-26.5.gometa.json")
	content := []byte(`{"framework":"Legacy","sdk_version":"26.5","arch":"arm64"}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read legacy metadata: %v", err)
	}
	if got.SchemaVersion != 0 {
		t.Errorf("SchemaVersion: got %d, want 0 for legacy file", got.SchemaVersion)
	}
}

// TestEnumMemberValueRoundTrip verifies that EnumMember.Value string values
// survive a JSON round-trip without precision loss (the previous interface{}
// field type caused large uint64 values to be decoded as float64, losing bits).
func TestEnumMemberValueRoundTrip(t *testing.T) {
	dir := t.TempDir()
	framework := minimalFM()
	framework.Enums = map[string]Enum{
		"BigEnum": {
			GoType: "uint64",
			Members: []EnumMember{
				{Name: "MaxUint64", Value: "18446744073709551615"},
				{Name: "HexVal", Value: "0xFFFFFFFF"},
				{Name: "NegVal", Value: "-1"},
			},
		},
	}
	if err := Write(framework, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	files, _ := os.ReadDir(dir)
	if len(files) == 0 {
		t.Fatal("no files written")
	}
	got, err := Read(filepath.Join(dir, files[0].Name()))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	members := got.Enums["BigEnum"].Members
	if len(members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(members))
	}
	for i, want := range []string{"18446744073709551615", "0xFFFFFFFF", "-1"} {
		if members[i].Value != want {
			t.Errorf("members[%d].Value = %q, want %q", i, members[i].Value, want)
		}
	}
}
