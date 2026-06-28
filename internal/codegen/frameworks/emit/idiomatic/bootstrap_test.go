//go:build darwin

package idiomatic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
)

func TestFrameworkDylibPath(t *testing.T) {
	cases := []struct {
		name      string
		framework *meta.FrameworkMeta
		want      string
	}{
		{
			name:      "plain framework",
			framework: &meta.FrameworkMeta{Framework: "Virtualization"},
			want:      "/System/Library/Frameworks/Virtualization.framework/Virtualization",
		},
		{
			name:      "sub-framework uses parent umbrella",
			framework: &meta.FrameworkMeta{Framework: "HIToolbox", ParentFramework: "Carbon"},
			want:      "/System/Library/Frameworks/Carbon.framework/Carbon",
		},
		{
			name:      "link library",
			framework: &meta.FrameworkMeta{Framework: "EndpointSecurity", LinkLib: "EndpointSecurity"},
			want:      "/usr/lib/libEndpointSecurity.dylib",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := frameworkDylibPath(c.framework); got != c.want {
				t.Errorf("frameworkDylibPath = %q, want %q", got, c.want)
			}
		})
	}
}

func TestEmitRuntimeBootstrap(t *testing.T) {
	dir := t.TempDir()
	framework := &meta.FrameworkMeta{Framework: "Virtualization"}
	if err := emitRuntimeBootstrap(dir, "virtualization", framework); err != nil {
		t.Fatalf("emitRuntimeBootstrap: %v", err)
	}
	path := filepath.Join(dir, "virtualization_runtime_generated.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read bootstrap: %v", err)
	}
	got := string(src)
	for _, want := range []string{
		"package virtualization",
		"func _class(name string) objc.Class",
		"_loadOnce.Do(_loadLibrary)",
		"/System/Library/Frameworks/Virtualization.framework/Virtualization",
		"//go:build darwin",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("bootstrap missing %q\n--- got ---\n%s", want, got)
		}
	}
}
