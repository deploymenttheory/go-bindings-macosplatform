package rawfw

import (
	"bytes"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

func TestSDK27EnumLimits(t *testing.T) {
	for _, tc := range []struct {
		name, base, value, want string
		negative                bool
	}{
		{"signed maximum", "int64", "9223372036854775807", "int64", true},
		{"hex signed maximum", "int64", "0x7fffffffffffffff", "int64", true},
		{"unsigned high bit", "int64", "9223372036854775808", "uint64", false},
		{"hex unsigned high bit", "int64", "0x8000000000000000", "uint64", false},
		{"signed 32 bit maximum", "int32", "2147483647", "int32", true},
		{"unsigned 32 bit high bit", "int32", "2147483648", "uint32", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			members := []meta.EnumMember{{Name: "FixtureLimit", Value: tc.value}}
			if tc.negative {
				members = append(members, meta.EnumMember{Name: "FixtureDefault", Value: "-1"})
			}
			if got := upgradeEnumTypeIfOverflow(tc.base, members); got != tc.want {
				t.Fatalf("type = %s, want %s", got, tc.want)
			}
			var out bytes.Buffer
			out.WriteString("package fixture\nimport \"fmt\"\n")
			err := EmitEnums(&out, &meta.FrameworkMeta{Framework: "Fixture", Enums: map[string]meta.Enum{"FixtureAction": {GoType: tc.base, Members: members}}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "fixture.go", out.Bytes(), 0)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := (&types.Config{Importer: importer.Default()}).Check("fixture", fset, []*ast.File{file}, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFunctionLinkAliasPreservesGoAPI(t *testing.T) {
	var out bytes.Buffer
	framework := &meta.FrameworkMeta{Framework: "Fixture", Functions: []meta.Function{{Name: "FixturePerform", LinkName: "FixturePerform_v2"}}}
	_, registrations, err := EmitFunctions(&out, framework, &typemap.Mapper{}, "_lib", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(registrations) != 1 || !strings.Contains(registrations[0].Line, `"FixturePerform_v2"`) {
		t.Fatalf("registration does not resolve the assembly alias: %+v", registrations)
	}
	if !strings.Contains(out.String(), "func FixturePerform(") || strings.Contains(out.String(), "func FixturePerform_v2(") {
		t.Fatalf("link alias changed the Go API: %s", out.String())
	}
}

func TestCFunctionProtocolHandles(t *testing.T) {
	mapper := &typemap.Mapper{
		ProtocolIndex: map[string]string{"FixtureDevice": "Fixture"},
		TypedefIndex:  map[string]string{"FixtureDeviceRef": "id<FixtureDevice>"},
	}
	framework := &meta.FrameworkMeta{Framework: "Fixture", Functions: []meta.Function{{
		Name: "FixtureRoundTrip",
		Params: []meta.Param{
			{Name: "device", ObjCType: "id<FixtureDevice>"},
			{Name: "observer", ObjCType: "id<FixtureDevice> _Nullable * _Nonnull"},
		},
		Return: meta.ReturnType{ObjCType: "FixtureDeviceRef"},
	}}}
	var out bytes.Buffer
	if _, _, err := EmitFunctions(&out, framework, mapper, "_lib", nil, nil); err != nil {
		t.Fatal(err)
	}
	for _, signature := range []string{
		"func(objc.ID, *objc.ID) objc.ID",
		"func FixtureRoundTrip(device objc.ID, observer *objc.ID) objc.ID",
	} {
		if !strings.Contains(out.String(), signature) {
			t.Fatalf("missing ABI-safe signature %q: %s", signature, out.String())
		}
	}
}

func TestUnderscoreFrameworkFilesAreCompiled(t *testing.T) {
	dir := t.TempDir()
	if err := WriteGoFile(filepath.Join(dir, "_fixture_enums.go"), []byte("package _fixture\n type Action int64\n const Default Action = -1\n")); err != nil {
		t.Fatal(err)
	}
	pkg, err := build.Default.ImportDir(dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg.GoFiles) != 1 || pkg.GoFiles[0] != "sdk_fixture_enums.go" {
		t.Fatalf("compiled files = %v", pkg.GoFiles)
	}
}
