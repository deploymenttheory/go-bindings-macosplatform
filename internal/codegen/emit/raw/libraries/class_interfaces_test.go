package rawlib

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

func ifaceTestMapper() *typemap.Mapper {
	return &typemap.Mapper{
		GenericClasses: map[string]bool{
			"NSArray": true,
		},
		OwnerIndex: map[string]string{
			"NSObject": "Foundation",
			"NSArray":  "Foundation",
		},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
		TypedefIndex:   map[string]string{},
		StructIndex:    map[string]string{},
	}
}

// TestClassInterfacesEmpty verifies no output for a framework with no classes.
func TestClassInterfacesEmpty(t *testing.T) {
	m := ifaceTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{Framework: "Foundation", Classes: map[string]macosplatformmetadata.Class{}}
	var buf bytes.Buffer
	if err := EmitClassInterfaces(&buf, "foundation", framework, m, map[string]bool{}, map[string]macosplatformmetadata.Class{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for empty framework; got:\n%s", buf.String())
	}
}

// TestClassInterfacesUnavailable verifies unavailable classes are skipped.
func TestClassInterfacesUnavailable(t *testing.T) {
	m := ifaceTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSFoo": {Availability: macosplatformmetadata.Availability{IsUnavailable: true}},
		},
	}
	var buf bytes.Buffer
	if err := EmitClassInterfaces(&buf, "foundation", framework, m, map[string]bool{}, map[string]macosplatformmetadata.Class{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for unavailable class; got:\n%s", buf.String())
	}
}

// TestClassInterfacesGenericClassSkipped verifies generic classes are skipped.
func TestClassInterfacesGenericClassSkipped(t *testing.T) {
	m := ifaceTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSArray": {GenericParams: []string{"ObjectType"}},
		},
	}
	var buf bytes.Buffer
	if err := EmitClassInterfaces(&buf, "foundation", framework, m, map[string]bool{}, map[string]macosplatformmetadata.Class{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "NSArrayable") {
		t.Errorf("generic class should be skipped; got:\n%s", buf.String())
	}
}

// TestClassInterfacesBasicClass verifies a basic class emits an interface.
func TestClassInterfacesBasicClass(t *testing.T) {
	m := ifaceTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {},
		},
	}
	var buf bytes.Buffer
	if err := EmitClassInterfaces(&buf, "foundation", framework, m, map[string]bool{"NSObject": true}, map[string]macosplatformmetadata.Class{"NSObject": {}}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "NSObjectable") {
		t.Errorf("expected NSObjectable interface; got:\n%s", out)
	}
	if !strings.Contains(out, "var _ NSObjectable = (*NSObject)(nil)") {
		t.Errorf("expected compile-time check; got:\n%s", out)
	}
}

// TestClassInterfacesWithMethod verifies methods appear in the interface.
func TestClassInterfacesWithMethod(t *testing.T) {
	m := ifaceTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {
				Methods: []macosplatformmetadata.Method{
					{Selector: "description", Return: macosplatformmetadata.ReturnType{ObjCType: "NSString *"}},
				},
			},
		},
	}
	knownClasses := map[string]bool{"NSObject": true}
	var buf bytes.Buffer
	if err := EmitClassInterfaces(&buf, "foundation", framework, m, knownClasses, map[string]macosplatformmetadata.Class{"NSObject": framework.Classes["NSObject"]}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Description") {
		t.Errorf("expected Description method in interface; got:\n%s", out)
	}
}

// TestClassInterfacesRootClassEmbedsObjcObject verifies root classes embed cgo.Object.
func TestClassInterfacesRootClassEmbedsObjcObject(t *testing.T) {
	m := ifaceTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {}, // no super → root
		},
	}
	var buf bytes.Buffer
	if err := EmitClassInterfaces(&buf, "foundation", framework, m, map[string]bool{"NSObject": true}, map[string]macosplatformmetadata.Class{"NSObject": {}}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "cgo.Object") {
		t.Errorf("root class interface should embed cgo.Object; got:\n%s", out)
	}
}

// TestClassInterfacesSameFwSuperEmbeds verifies same-fw super interface is embedded.
func TestClassInterfacesSameFwSuperEmbeds(t *testing.T) {
	m := ifaceTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject":  {},
			"NSMyClass": {Super: "NSObject"},
		},
	}
	allClasses := map[string]macosplatformmetadata.Class{
		"NSObject":  {},
		"NSMyClass": {Super: "NSObject"},
	}
	knownClasses := map[string]bool{"NSObject": true, "NSMyClass": true}
	var buf bytes.Buffer
	if err := EmitClassInterfaces(&buf, "foundation", framework, m, knownClasses, allClasses); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "NSObjectable") {
		t.Errorf("subclass should embed NSObjectable; got:\n%s", out)
	}
}

// TestClassInterfacesSkipsProtocolConflict verifies class with protocol name collision is skipped.
func TestClassInterfacesSkipsProtocolConflict(t *testing.T) {
	m := ifaceTestMapper()
	// Protocol "NSFooable" already exists — class "NSFoo" would collide.
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSFoo": {},
		},
		Protocols: map[string]macosplatformmetadata.Protocol{
			"NSFooable": {},
		},
	}
	var buf bytes.Buffer
	if err := EmitClassInterfaces(&buf, "foundation", framework, m, map[string]bool{}, map[string]macosplatformmetadata.Class{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "NSFooable") {
		t.Errorf("class with conflicting protocol name should be skipped; got:\n%s", buf.String())
	}
}

// TestClassInterfacesSkipsClassMethods verifies class methods are not in the interface.
func TestClassInterfacesSkipsClassMethods(t *testing.T) {
	m := ifaceTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {
				Methods: []macosplatformmetadata.Method{
					{Selector: "new", IsClassMethod: true, Return: macosplatformmetadata.ReturnType{ObjCType: "instancetype"}},
					{Selector: "description", Return: macosplatformmetadata.ReturnType{ObjCType: "NSString *"}},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitClassInterfaces(&buf, "foundation", framework, m, map[string]bool{"NSObject": true}, map[string]macosplatformmetadata.Class{"NSObject": framework.Classes["NSObject"]}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Class methods appear as NSObjectNew() at package level, not in the interface.
	if strings.Contains(out, "NSObjectNew") {
		t.Errorf("class method should not appear in interface; got:\n%s", out)
	}
}

// TestClassInterfacesSkipsInitMethods verifies init methods are excluded.
func TestClassInterfacesSkipsInitMethods(t *testing.T) {
	m := ifaceTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject": {
				Methods: []macosplatformmetadata.Method{
					{Selector: "init", IsInit: true, Return: macosplatformmetadata.ReturnType{IsInstancetype: true}},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := EmitClassInterfaces(&buf, "foundation", framework, m, map[string]bool{"NSObject": true}, map[string]macosplatformmetadata.Class{"NSObject": framework.Classes["NSObject"]}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "\tInit(") {
		t.Errorf("init methods should be excluded from interface; got:\n%s", out)
	}
}

// TestClassInterfacesInheritedMethodsSkipped verifies methods promoted from a parent are not repeated.
func TestClassInterfacesInheritedMethodsSkipped(t *testing.T) {
	m := ifaceTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSObject":  {Methods: []macosplatformmetadata.Method{{Selector: "description", Return: macosplatformmetadata.ReturnType{ObjCType: "NSString *"}}}},
			"NSMyClass": {Super: "NSObject"},
		},
	}
	allClasses := map[string]macosplatformmetadata.Class{
		"NSObject":  framework.Classes["NSObject"],
		"NSMyClass": framework.Classes["NSMyClass"],
	}
	knownClasses := map[string]bool{"NSObject": true, "NSMyClass": true}
	var buf bytes.Buffer
	if err := EmitClassInterfaces(&buf, "foundation", framework, m, knownClasses, allClasses); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// NSMyClass should embed NSObjectable but NOT re-declare Description.
	if !strings.Contains(out, "NSObjectable") {
		t.Errorf("subclass interface should embed NSObjectable; got:\n%s", out)
	}
	// Count occurrences of "Description(" — should only appear once (in NSObject interface).
	count := strings.Count(out, "Description(")
	if count > 1 {
		t.Errorf("Description method should not be repeated in subclass interface; found %d times:\n%s", count, out)
	}
}

// TestClassInterfacesNSCodingClass verifies NSSecureCoding class gets SerializeToArchive.
func TestClassInterfacesNSCodingClass(t *testing.T) {
	m := ifaceTestMapper()
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "Foundation",
		Classes: map[string]macosplatformmetadata.Class{
			"NSString": {
				Protocols: []string{"NSSecureCoding"},
			},
		},
		Protocols: map[string]macosplatformmetadata.Protocol{},
	}
	allClasses := map[string]macosplatformmetadata.Class{"NSString": framework.Classes["NSString"]}
	var buf bytes.Buffer
	if err := EmitClassInterfaces(&buf, "foundation", framework, m, map[string]bool{"NSString": true}, allClasses); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "SerializeToArchive") {
		t.Errorf("NSSecureCoding class should have SerializeToArchive; got:\n%s", out)
	}
}

// TestClassInterfacesCrossFwSuperEmbeds verifies cross-framework super uses qualified embed.
func TestClassInterfacesCrossFwSuperEmbeds(t *testing.T) {
	m := &typemap.Mapper{
		GenericClasses: map[string]bool{},
		OwnerIndex: map[string]string{
			"NSView":   "AppKit",
			"NSObject": "Foundation",
		},
		ModulePrefix:   "github.com/example/fw",
		BlockedImports: map[string]map[string]bool{},
		TypedefIndex:   map[string]string{},
		StructIndex:    map[string]string{},
	}
	// NSView's super (NSObject) is in Foundation — cross-fw.
	framework := &macosplatformmetadata.FrameworkMeta{
		Framework: "AppKit",
		Classes: map[string]macosplatformmetadata.Class{
			"NSView": {Super: "NSObject"},
		},
	}
	allClasses := map[string]macosplatformmetadata.Class{
		"NSObject": {},
		"NSView":   {Super: "NSObject"},
	}
	knownClasses := map[string]bool{"NSObject": true, "NSView": true}
	var buf bytes.Buffer
	if err := EmitClassInterfaces(&buf, "appkit", framework, m, knownClasses, allClasses); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "foundation.NSObjectable") {
		t.Errorf("cross-fw super should be qualified; got:\n%s", out)
	}
}
