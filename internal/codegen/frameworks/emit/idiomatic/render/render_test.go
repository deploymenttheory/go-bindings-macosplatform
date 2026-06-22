package render

import (
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/idiomatic/view"
)

// TestStructs locks the rendered form of value structs: a documented struct, an
// undocumented one, and field order — matching what the previous string-builder
// produced (so the migration is byte-identical after gofmt).
func TestStructs(t *testing.T) {
	got, err := Structs([]view.Struct{
		{
			GoName: "CGPoint",
			Doc:    "A structure that contains a point in a two-dimensional coordinate system.",
			Fields: []view.Field{{GoName: "X", GoType: "float64"}, {GoName: "Y", GoType: "float64"}},
		},
		{
			GoName: "CGRect",
			Fields: []view.Field{{GoName: "Origin", GoType: "CGPoint"}, {GoName: "Size", GoType: "CGSize"}},
		},
	})
	if err != nil {
		t.Fatalf("Structs: %v", err)
	}

	want := "// A structure that contains a point in a two-dimensional coordinate system.\n" +
		"type CGPoint struct {\n" +
		"X float64\n" +
		"Y float64\n" +
		"}\n\n" +
		"type CGRect struct {\n" +
		"Origin CGPoint\n" +
		"Size CGSize\n" +
		"}\n\n"

	if string(got) != want {
		t.Errorf("Structs mismatch:\n got %q\nwant %q", got, want)
	}
}

// TestComment covers the doc-comment helper, including the empty case and a
// multi-line block.
func TestComment(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"one line", "// one line"},
		{"first\nsecond", "// first\n// second"},
	}
	for _, c := range cases {
		if got := comment(c.in); got != c.want {
			t.Errorf("comment(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
