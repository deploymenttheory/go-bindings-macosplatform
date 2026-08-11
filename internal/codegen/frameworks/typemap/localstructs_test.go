package typemap

import "testing"

// TestLocalStructOwner covers the local-declaration preference for struct
// references: a framework that itself emits a field-bearing struct must
// resolve references to the local copy instead of importing the global
// StructIndex owner (which manufactured the metal⇄compositorservices and
// commonpanels⇄hitoolbox import cycles).
func TestLocalStructOwner(t *testing.T) {
	m := &Mapper{
		StructIndex: map[string]string{
			"MTLViewport": "CompositorServices",
			"_RGBColor":   "CommonPanels",
		},
		LocalStructs: map[string]map[string]bool{
			"Metal":     {"MTLViewport": true},
			"HIToolbox": {"RGBColor": true},
		},
		ModulePrefix: "example.com/frameworks",
	}

	cases := []struct {
		name        string
		qt          string
		framework   string
		want        string
		wantImports int
	}{
		{"local named", "MTLViewport", "Metal", "MTLViewport", 0},
		{"local pointer", "MTLViewport *", "Metal", "*MTLViewport", 0},
		{"non-declaring framework still qualifies", "MTLViewport", "AppKit", "compositorservices.MTLViewport", 1},
		{"owner unchanged", "MTLViewport", "CompositorServices", "MTLViewport", 0},
		{"underscore declaration matches exported keying", "_RGBColor", "HIToolbox", "RGBColor", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			imports := ImportSet{}
			got := m.GoType(c.qt, Context{Framework: c.framework}, imports)
			if got != c.want {
				t.Errorf("GoType(%q, %s) = %q; want %q", c.qt, c.framework, got, c.want)
			}
			if len(imports) != c.wantImports {
				t.Errorf("GoType(%q, %s) imports = %v; want %d entries", c.qt, c.framework, imports, c.wantImports)
			}
		})
	}
}
