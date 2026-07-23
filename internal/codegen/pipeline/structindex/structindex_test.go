package structindex

import (
	"testing"

	mpm "github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

func fm(name string, structs map[string]mpm.Struct) *mpm.FrameworkMeta {
	return &mpm.FrameworkMeta{Framework: name, Structs: structs}
}

func withFields(n int) mpm.Struct {
	s := mpm.Struct{}
	for i := 0; i < n; i++ {
		s.Fields = append(s.Fields, mpm.StructField{Name: "f"})
	}
	return s
}

func TestBuildOwnerAttribution(t *testing.T) {
	frameworks := []*mpm.FrameworkMeta{
		// Zebra defines Foo with fields but sorts last.
		fm("Zebra", map[string]mpm.Struct{"foo": withFields(1)}),
		// Alpha also defines Foo with fields; lexicographically smaller → owner.
		fm("Alpha", map[string]mpm.Struct{"foo": withFields(1)}),
		// Beta defines Foo with ZERO fields (opaque) — must not win.
		fm("Beta", map[string]mpm.Struct{"foo": withFields(0), "bar": withFields(0)}),
	}
	idx := map[string]string{}
	Build(frameworks, idx, map[string]string{})

	if idx["foo"] != "Alpha" {
		t.Errorf("foo owner = %q; want Alpha (lexicographically smallest with fields)", idx["foo"])
	}
	if _, ok := idx["bar"]; ok {
		t.Errorf("bar (zero fields everywhere) should not be registered; got owner %q", idx["bar"])
	}
}

func TestBuildTypedefBackfill(t *testing.T) {
	// Struct stored under underscore tag _NSRange, exposed via typedef NSRange.
	frameworks := []*mpm.FrameworkMeta{
		fm("Foundation", map[string]mpm.Struct{"_NSRange": withFields(2)}),
	}
	idx := map[string]string{}
	typedefs := map[string]string{"NSRange": "struct _NSRange"}
	Build(frameworks, idx, typedefs)

	if idx["_NSRange"] != "Foundation" {
		t.Errorf("_NSRange owner = %q; want Foundation", idx["_NSRange"])
	}
	if idx["NSRange"] != "Foundation" {
		t.Errorf("clean typedef NSRange not backfilled to owner; got %q", idx["NSRange"])
	}
}
