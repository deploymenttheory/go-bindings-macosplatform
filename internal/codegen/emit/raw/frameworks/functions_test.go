package rawfw

import (
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
)

// TestFunctionGoNameFor covers the C-library "Fn" collision suffix: a library
// function whose exported Go name collides with an emitted type keeps the
// suffix-resolved name (mach_time.h declares both a mach_timebase_info struct
// and function), while frameworks keep the skip-on-collision behaviour
// (FunctionGoNameFor returns the colliding name unchanged there; the skip
// happens in EmittableFunctions).
func TestFunctionGoNameFor(t *testing.T) {
	library := &meta.FrameworkMeta{
		Framework: "machtime",
		LinkLib:   "System",
		Structs: map[string]meta.Struct{
			"mach_timebase_info": {},
		},
	}
	framework := &meta.FrameworkMeta{
		Framework: "Metal",
		Structs: map[string]meta.Struct{
			"mach_timebase_info": {},
		},
	}

	cases := []struct {
		name string
		fw   *meta.FrameworkMeta
		fn   string
		want string
	}{
		{"library collision gains Fn suffix", library, "mach_timebase_info", "MachTimebaseInfoFn"},
		{"library non-collision unchanged", library, "mach_absolute_time", "MachAbsoluteTime"},
		{"framework collision NOT suffixed", framework, "mach_timebase_info", "MachTimebaseInfo"},
		{"already-exported name kept byte-identical", library, "MachTimebaseInfo", "MachTimebaseInfo"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := FunctionGoNameFor(c.fw, meta.Function{Name: c.fn})
			if got != c.want {
				t.Errorf("FunctionGoNameFor(%s, %s) = %q; want %q", c.fw.Framework, c.fn, got, c.want)
			}
		})
	}
}
