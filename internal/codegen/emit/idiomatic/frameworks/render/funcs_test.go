package render

import (
	"sort"
	"strings"
	"testing"
)

// TestFuncMapExact enforces P7: the render FuncMap holds exactly the pure
// formatting helpers it is supposed to, so type-deciding logic cannot creep into
// the render layer disguised as a template helper. Adding a helper means updating
// this list deliberately.
func TestFuncMapExact(t *testing.T) {
	want := []string{"cfuncRetOutValues", "comment", "join", "retOutValues", "retOutZeros", "wrap"}
	var got []string
	for k := range funcMap {
		got = append(got, k)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("render FuncMap keys = %v, want %v", got, want)
	}
}
