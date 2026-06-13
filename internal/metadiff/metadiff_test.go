package metadiff

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

func baseFramework() *meta.FrameworkMeta {
	return &meta.FrameworkMeta{
		Framework:  "Foundation",
		SDKVersion: "26.5",
		Arch:       "arm64",
		Classes: map[string]meta.Class{
			"NSString": {
				Super: "NSObject",
				Methods: []meta.Method{
					{Selector: "length", Return: meta.ReturnType{ObjCType: "NSUInteger"}},
					{Selector: "stringWithFormat:", IsClassMethod: true,
						Params: []meta.Param{{Name: "format", ObjCType: "NSString *"}},
						Return: meta.ReturnType{ObjCType: "instancetype"}},
				},
			},
		},
		Enums: map[string]meta.Enum{
			"NSComparisonResult": {
				GoType: "int64",
				Members: []meta.EnumMember{
					{Name: "NSOrderedAscending", Value: "-1"},
					{Name: "NSOrderedSame", Value: "0"},
				},
			},
		},
		Functions: []meta.Function{
			{Name: "NSMakeRange",
				Params: []meta.Param{{Name: "loc", ObjCType: "NSUInteger"}, {Name: "len", ObjCType: "NSUInteger"}},
				Return: meta.ReturnType{ObjCType: "NSRange"}},
		},
		Externs: []meta.Extern{
			{Name: "NSFoundationVersionNumber", ObjCType: "double"},
		},
	}
}

// clone produces an independent deep-enough copy for mutation in tests.
func clone(framework *meta.FrameworkMeta) *meta.FrameworkMeta {
	out := *framework
	out.Classes = map[string]meta.Class{}
	for name, class := range framework.Classes {
		methods := make([]meta.Method, len(class.Methods))
		copy(methods, class.Methods)
		class.Methods = methods
		out.Classes[name] = class
	}
	out.Enums = map[string]meta.Enum{}
	for name, enum := range framework.Enums {
		members := make([]meta.EnumMember, len(enum.Members))
		copy(members, enum.Members)
		enum.Members = members
		out.Enums[name] = enum
	}
	out.Functions = make([]meta.Function, len(framework.Functions))
	copy(out.Functions, framework.Functions)
	out.Externs = make([]meta.Extern, len(framework.Externs))
	copy(out.Externs, framework.Externs)
	return &out
}

func TestIdenticalTreesAreEmpty(t *testing.T) {
	report := Compare(
		[]*meta.FrameworkMeta{baseFramework()},
		[]*meta.FrameworkMeta{baseFramework()},
	)
	if !report.IsEmpty() {
		t.Errorf("expected empty report for identical trees, got %+v", report)
	}
}

func TestFrameworkAddedAndRemoved(t *testing.T) {
	oldTree := []*meta.FrameworkMeta{baseFramework()}
	added := baseFramework()
	added.Framework = "NewKit"
	report := Compare(oldTree, []*meta.FrameworkMeta{added})

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
	newFramework.Classes["NSNewThing"] = meta.Class{Super: "NSObject"}
	nsstring := newFramework.Classes["NSString"]
	nsstring.Methods = append(nsstring.Methods, meta.Method{Selector: "uppercaseString", Return: meta.ReturnType{ObjCType: "NSString *"}})
	nsstring.Methods[0].Return = meta.ReturnType{ObjCType: "NSInteger"} // length: NSUInteger → NSInteger
	newFramework.Classes["NSString"] = nsstring

	report := Compare([]*meta.FrameworkMeta{oldFramework}, []*meta.FrameworkMeta{newFramework})
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
	nsstring.Methods = append(nsstring.Methods, meta.Method{
		Selector: "stringWithFormat:", IsClassMethod: false,
		Return: meta.ReturnType{ObjCType: "instancetype"},
	})
	newFramework.Classes["NSString"] = nsstring

	report := Compare([]*meta.FrameworkMeta{oldFramework}, []*meta.FrameworkMeta{newFramework})
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
	enum.Members = append(enum.Members, meta.EnumMember{Name: "NSOrderedDescending", Value: "1"})
	enum.Members = enum.Members[1:] // drop NSOrderedAscending
	newFramework.Enums["NSComparisonResult"] = enum

	report := Compare([]*meta.FrameworkMeta{oldFramework}, []*meta.FrameworkMeta{newFramework})
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
	newFramework.Functions[0].Return = meta.ReturnType{ObjCType: "void"}
	newFramework.Functions = append(newFramework.Functions, meta.Function{Name: "NSNewFunc", Return: meta.ReturnType{ObjCType: "void"}})
	newFramework.Externs = nil

	report := Compare([]*meta.FrameworkMeta{oldFramework}, []*meta.FrameworkMeta{newFramework})
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
	makeOverloads := func(order []string) *meta.FrameworkMeta {
		f := baseFramework()
		f.Functions = nil
		for _, t := range order {
			f.Functions = append(f.Functions, meta.Function{
				Name:   "SparseCleanup",
				Params: []meta.Param{{Name: "x", ObjCType: t}},
				Return: meta.ReturnType{ObjCType: "void"},
			})
		}
		return f
	}
	report := Compare(
		[]*meta.FrameworkMeta{makeOverloads([]string{"TypeA", "TypeB"})},
		[]*meta.FrameworkMeta{makeOverloads([]string{"TypeB", "TypeA"})},
	)
	if !report.IsEmpty() {
		t.Errorf("identical overload sets must not diff, got %+v", report.Changed)
	}

	// A genuinely new overload must surface as one change for the name.
	report = Compare(
		[]*meta.FrameworkMeta{makeOverloads([]string{"TypeA"})},
		[]*meta.FrameworkMeta{makeOverloads([]string{"TypeA", "TypeC"})},
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

	report := Compare([]*meta.FrameworkMeta{oldFramework}, []*meta.FrameworkMeta{newFramework})
	d := report.Changed[0]
	if len(d.DeprecationChanges) != 1 || !strings.Contains(d.DeprecationChanges[0], "(none) → 27.0") {
		t.Errorf("DeprecationChanges: got %v", d.DeprecationChanges)
	}
}

func TestProvenanceChangesAreNotSemantic(t *testing.T) {
	oldFramework := baseFramework()
	newFramework := clone(baseFramework())
	newFramework.SchemaVersion = meta.CurrentSchemaVersion
	newFramework.ClangVersion = "Apple clang version 99"
	newFramework.XcodeVersion = "Xcode 99"

	report := Compare([]*meta.FrameworkMeta{oldFramework}, []*meta.FrameworkMeta{newFramework})
	if !report.IsEmpty() {
		t.Errorf("provenance-only changes must not appear in the diff, got %+v", report.Changed)
	}
}

func TestMarkdownOutput(t *testing.T) {
	oldFramework := baseFramework()
	newFramework := clone(baseFramework())
	newFramework.Classes["NSNewThing"] = meta.Class{Super: "NSObject"}

	report := Compare([]*meta.FrameworkMeta{oldFramework}, []*meta.FrameworkMeta{newFramework})
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
		[]*meta.FrameworkMeta{baseFramework()},
		[]*meta.FrameworkMeta{baseFramework()},
	)
	var buf bytes.Buffer
	report.WriteMarkdown(&buf)
	if !strings.Contains(buf.String(), "No semantic changes") {
		t.Errorf("empty report should say so:\n%s", buf.String())
	}
}
