package metadiff

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

func baseFramework() *macosplatformmetadata.FrameworkMeta {
	return &macosplatformmetadata.FrameworkMeta{
		Framework:  "Foundation",
		SDKVersion: "26.5",
		Arch:       "arm64",
		Classes: map[string]macosplatformmetadata.Class{
			"NSString": {
				Super: "NSObject",
				Methods: []macosplatformmetadata.Method{
					{Selector: "length", Return: macosplatformmetadata.ReturnType{ObjCType: "NSUInteger"}},
					{Selector: "stringWithFormat:", IsClassMethod: true,
						Params: []macosplatformmetadata.Param{{Name: "format", ObjCType: "NSString *"}},
						Return: macosplatformmetadata.ReturnType{ObjCType: "instancetype"}},
				},
			},
		},
		Enums: map[string]macosplatformmetadata.Enum{
			"NSComparisonResult": {
				GoType: "int64",
				Members: []macosplatformmetadata.EnumMember{
					{Name: "NSOrderedAscending", Value: "-1"},
					{Name: "NSOrderedSame", Value: "0"},
				},
			},
		},
		Functions: []macosplatformmetadata.Function{
			{Name: "NSMakeRange",
				Params: []macosplatformmetadata.Param{{Name: "loc", ObjCType: "NSUInteger"}, {Name: "len", ObjCType: "NSUInteger"}},
				Return: macosplatformmetadata.ReturnType{ObjCType: "NSRange"}},
		},
		Externs: []macosplatformmetadata.Extern{
			{Name: "NSFoundationVersionNumber", ObjCType: "double"},
		},
	}
}

// clone produces an independent deep-enough copy for mutation in tests.
func clone(framework *macosplatformmetadata.FrameworkMeta) *macosplatformmetadata.FrameworkMeta {
	out := *framework
	out.Classes = map[string]macosplatformmetadata.Class{}
	for name, class := range framework.Classes {
		methods := make([]macosplatformmetadata.Method, len(class.Methods))
		copy(methods, class.Methods)
		class.Methods = methods
		out.Classes[name] = class
	}
	out.Enums = map[string]macosplatformmetadata.Enum{}
	for name, enum := range framework.Enums {
		members := make([]macosplatformmetadata.EnumMember, len(enum.Members))
		copy(members, enum.Members)
		enum.Members = members
		out.Enums[name] = enum
	}
	out.Functions = make([]macosplatformmetadata.Function, len(framework.Functions))
	copy(out.Functions, framework.Functions)
	out.Externs = make([]macosplatformmetadata.Extern, len(framework.Externs))
	copy(out.Externs, framework.Externs)
	return &out
}

func TestIdenticalTreesAreEmpty(t *testing.T) {
	report := Compare(
		[]*macosplatformmetadata.FrameworkMeta{baseFramework()},
		[]*macosplatformmetadata.FrameworkMeta{baseFramework()},
	)
	if !report.IsEmpty() {
		t.Errorf("expected empty report for identical trees, got %+v", report)
	}
}

func TestFrameworkAddedAndRemoved(t *testing.T) {
	oldTree := []*macosplatformmetadata.FrameworkMeta{baseFramework()}
	added := baseFramework()
	added.Framework = "NewKit"
	report := Compare(oldTree, []*macosplatformmetadata.FrameworkMeta{added})

	if len(report.FrameworksAdded) != 1 || report.FrameworksAdded[0] != "NewKit" {
		t.Errorf("FrameworksAdded: got %v, want [NewKit]", report.FrameworksAdded)
	}
	if len(report.FrameworksRemoved) != 1 || report.FrameworksRemoved[0] != "Foundation" {
		t.Errorf("FrameworksRemoved: got %v, want [Foundation]", report.FrameworksRemoved)
	}
}

func TestClassAndMethodChanges(t *testing.T) {
	oldFramework := baseFramework()
	newFramework := clone(baseFramework())

	// Add a class, add a method, change a signature.
	newFramework.Classes["NSNewThing"] = macosplatformmetadata.Class{Super: "NSObject"}
	nsstring := newFramework.Classes["NSString"]
	nsstring.Methods = append(nsstring.Methods, macosplatformmetadata.Method{Selector: "uppercaseString", Return: macosplatformmetadata.ReturnType{ObjCType: "NSString *"}})
	nsstring.Methods[0].Return = macosplatformmetadata.ReturnType{ObjCType: "NSInteger"} // length: NSUInteger → NSInteger
	newFramework.Classes["NSString"] = nsstring

	report := Compare([]*macosplatformmetadata.FrameworkMeta{oldFramework}, []*macosplatformmetadata.FrameworkMeta{newFramework})
	if len(report.Changed) != 1 {
		t.Fatalf("expected 1 changed framework, got %d", len(report.Changed))
	}
	d := report.Changed[0]
	if len(d.ClassesAdded) != 1 || d.ClassesAdded[0] != "NSNewThing" {
		t.Errorf("ClassesAdded: got %v", d.ClassesAdded)
	}
	if len(d.MethodsAdded) != 1 || d.MethodsAdded[0] != "NSString -uppercaseString" {
		t.Errorf("MethodsAdded: got %v", d.MethodsAdded)
	}
	if len(d.MethodChanges) != 1 || d.MethodChanges[0].Name != "NSString -length" {
		t.Errorf("MethodChanges: got %v", d.MethodChanges)
	}
	if d.MethodChanges[0].Old == d.MethodChanges[0].New {
		t.Errorf("signature change should differ: %+v", d.MethodChanges[0])
	}
}

func TestClassVsInstanceMethodDistinct(t *testing.T) {
	oldFramework := baseFramework()
	newFramework := clone(baseFramework())
	// Add an instance method with the same selector as an existing class method.
	nsstring := newFramework.Classes["NSString"]
	nsstring.Methods = append(nsstring.Methods, macosplatformmetadata.Method{
		Selector: "stringWithFormat:", IsClassMethod: false,
		Return: macosplatformmetadata.ReturnType{ObjCType: "instancetype"},
	})
	newFramework.Classes["NSString"] = nsstring

	report := Compare([]*macosplatformmetadata.FrameworkMeta{oldFramework}, []*macosplatformmetadata.FrameworkMeta{newFramework})
	if len(report.Changed) != 1 || len(report.Changed[0].MethodsAdded) != 1 {
		t.Fatalf("expected the instance-method addition to be detected: %+v", report)
	}
	if report.Changed[0].MethodsAdded[0] != "NSString -stringWithFormat:" {
		t.Errorf("MethodsAdded: got %v", report.Changed[0].MethodsAdded)
	}
}

func TestEnumChanges(t *testing.T) {
	oldFramework := baseFramework()
	newFramework := clone(baseFramework())
	enum := newFramework.Enums["NSComparisonResult"]
	enum.GoType = "uint64"
	enum.Members = append(enum.Members, macosplatformmetadata.EnumMember{Name: "NSOrderedDescending", Value: "1"})
	enum.Members = enum.Members[1:] // drop NSOrderedAscending
	newFramework.Enums["NSComparisonResult"] = enum

	report := Compare([]*macosplatformmetadata.FrameworkMeta{oldFramework}, []*macosplatformmetadata.FrameworkMeta{newFramework})
	if len(report.Changed) != 1 {
		t.Fatalf("expected 1 changed framework, got %+v", report)
	}
	d := report.Changed[0]
	if len(d.EnumBaseTypeChanges) != 1 || d.EnumBaseTypeChanges[0].Old != "int64" || d.EnumBaseTypeChanges[0].New != "uint64" {
		t.Errorf("EnumBaseTypeChanges: got %v", d.EnumBaseTypeChanges)
	}
	if len(d.EnumMembersAdded) != 1 || d.EnumMembersAdded[0] != "NSComparisonResult.NSOrderedDescending" {
		t.Errorf("EnumMembersAdded: got %v", d.EnumMembersAdded)
	}
	if len(d.EnumMembersRemoved) != 1 || d.EnumMembersRemoved[0] != "NSComparisonResult.NSOrderedAscending" {
		t.Errorf("EnumMembersRemoved: got %v", d.EnumMembersRemoved)
	}
}

func TestFunctionAndExternChanges(t *testing.T) {
	oldFramework := baseFramework()
	newFramework := clone(baseFramework())
	newFramework.Functions[0].Return = macosplatformmetadata.ReturnType{ObjCType: "void"}
	newFramework.Functions = append(newFramework.Functions, macosplatformmetadata.Function{Name: "NSNewFunc", Return: macosplatformmetadata.ReturnType{ObjCType: "void"}})
	newFramework.Externs = nil

	report := Compare([]*macosplatformmetadata.FrameworkMeta{oldFramework}, []*macosplatformmetadata.FrameworkMeta{newFramework})
	d := report.Changed[0]
	if len(d.FunctionChanges) != 1 || d.FunctionChanges[0].Name != "NSMakeRange" {
		t.Errorf("FunctionChanges: got %v", d.FunctionChanges)
	}
	if len(d.FunctionsAdded) != 1 || d.FunctionsAdded[0] != "NSNewFunc" {
		t.Errorf("FunctionsAdded: got %v", d.FunctionsAdded)
	}
	if len(d.ExternsRemoved) != 1 || d.ExternsRemoved[0] != "NSFoundationVersionNumber" {
		t.Errorf("ExternsRemoved: got %v", d.ExternsRemoved)
	}
}

func TestOverloadedFunctionsNotFalselyChanged(t *testing.T) {
	// C overloadable functions repeat the same name with different signatures
	// (vecLib's SparseCleanup has 34 variants). Identical multisets must not
	// report changes regardless of declaration order.
	makeOverloads := func(order []string) *macosplatformmetadata.FrameworkMeta {
		f := baseFramework()
		f.Functions = nil
		for _, t := range order {
			f.Functions = append(f.Functions, macosplatformmetadata.Function{
				Name:   "SparseCleanup",
				Params: []macosplatformmetadata.Param{{Name: "x", ObjCType: t}},
				Return: macosplatformmetadata.ReturnType{ObjCType: "void"},
			})
		}
		return f
	}
	report := Compare(
		[]*macosplatformmetadata.FrameworkMeta{makeOverloads([]string{"TypeA", "TypeB"})},
		[]*macosplatformmetadata.FrameworkMeta{makeOverloads([]string{"TypeB", "TypeA"})},
	)
	if !report.IsEmpty() {
		t.Errorf("identical overload sets must not diff, got %+v", report.Changed)
	}

	// A genuinely new overload must surface as one change for the name.
	report = Compare(
		[]*macosplatformmetadata.FrameworkMeta{makeOverloads([]string{"TypeA"})},
		[]*macosplatformmetadata.FrameworkMeta{makeOverloads([]string{"TypeA", "TypeC"})},
	)
	if len(report.Changed) != 1 || len(report.Changed[0].FunctionChanges) != 1 {
		t.Errorf("expected 1 overload-set change, got %+v", report.Changed)
	}
}

func TestDeprecationChangeReported(t *testing.T) {
	oldFramework := baseFramework()
	newFramework := clone(baseFramework())
	nsstring := newFramework.Classes["NSString"]
	nsstring.Availability.MacOSDeprecated = "27.0"
	newFramework.Classes["NSString"] = nsstring

	report := Compare([]*macosplatformmetadata.FrameworkMeta{oldFramework}, []*macosplatformmetadata.FrameworkMeta{newFramework})
	d := report.Changed[0]
	if len(d.DeprecationChanges) != 1 || !strings.Contains(d.DeprecationChanges[0], "(none) → 27.0") {
		t.Errorf("DeprecationChanges: got %v", d.DeprecationChanges)
	}
}

func TestProvenanceChangesAreNotSemantic(t *testing.T) {
	oldFramework := baseFramework()
	newFramework := clone(baseFramework())
	newFramework.SchemaVersion = macosplatformmetadata.CurrentSchemaVersion
	newFramework.ClangVersion = "Apple clang version 99"
	newFramework.XcodeVersion = "Xcode 99"

	report := Compare([]*macosplatformmetadata.FrameworkMeta{oldFramework}, []*macosplatformmetadata.FrameworkMeta{newFramework})
	if !report.IsEmpty() {
		t.Errorf("provenance-only changes must not appear in the diff, got %+v", report.Changed)
	}
}

func TestMarkdownOutput(t *testing.T) {
	oldFramework := baseFramework()
	newFramework := clone(baseFramework())
	newFramework.Classes["NSNewThing"] = macosplatformmetadata.Class{Super: "NSObject"}

	report := Compare([]*macosplatformmetadata.FrameworkMeta{oldFramework}, []*macosplatformmetadata.FrameworkMeta{newFramework})
	var buf bytes.Buffer
	report.WriteMarkdown(&buf)
	out := buf.String()
	for _, want := range []string{"# Metadata diff", "## Summary", "## Foundation", "NSNewThing"} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown missing %q:\n%s", want, out)
		}
	}
}

func TestMarkdownEmptyReport(t *testing.T) {
	report := Compare(
		[]*macosplatformmetadata.FrameworkMeta{baseFramework()},
		[]*macosplatformmetadata.FrameworkMeta{baseFramework()},
	)
	var buf bytes.Buffer
	report.WriteMarkdown(&buf)
	if !strings.Contains(buf.String(), "No semantic changes") {
		t.Errorf("empty report should say so:\n%s", buf.String())
	}
}
