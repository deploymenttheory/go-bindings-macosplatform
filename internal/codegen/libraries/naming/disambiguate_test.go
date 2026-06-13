package naming_test

import (
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

func methods(sels ...string) []macosplatformmetadata.Method {
	out := make([]macosplatformmetadata.Method, len(sels))
	for i, s := range sels {
		out[i].Selector = s
	}
	return out
}

func countMap(ms []macosplatformmetadata.Method) map[string]int {
	c := make(map[string]int)
	for _, m := range ms {
		c[naming.MethodName(m.Selector)]++
	}
	return c
}

// TestRule1_ZeroArgOwnsCleanName: when "open" and "open:" both map to "Open",
// the zero-arg form keeps the clean name.
func TestRule1_ZeroArgOwnsCleanName(t *testing.T) {
	ms := methods("open", "open:")
	counts := countMap(ms)

	r, skip := naming.ResolveGoMethodName("Open", ms[0], ms, counts)
	if skip {
		t.Fatal("zero-arg form should not be skipped")
	}
	if r != "Open" {
		t.Errorf("zero-arg: got %q, want Open", r)
	}

	r, skip = naming.ResolveGoMethodName("Open", ms[1], ms, counts)
	if skip {
		t.Fatal("non-IBAction one-arg form should not be skipped")
	}
	if r != "Open1" {
		t.Errorf("one-arg: got %q, want Open1", r)
	}
}

// TestRule2_IBActionSkipped: a one-arg method with bare `id` sender is skipped
// when a zero-arg sibling exists.
func TestRule2_IBActionSkipped(t *testing.T) {
	goBack := macosplatformmetadata.Method{Selector: "goBack"}
	goBackSender := macosplatformmetadata.Method{
		Selector: "goBack:",
		Params:     []macosplatformmetadata.Param{{ObjCType: "id"}},
	}
	ms := []macosplatformmetadata.Method{goBack, goBackSender}
	counts := map[string]int{"GoBack": 2}

	_, skip := naming.ResolveGoMethodName("GoBack", goBackSender, ms, counts)
	if !skip {
		t.Error("IBAction one-arg form should be skipped")
	}

	r, skip := naming.ResolveGoMethodName("GoBack", goBack, ms, counts)
	if skip {
		t.Error("zero-arg form should not be skipped")
	}
	if r != "GoBack" {
		t.Errorf("zero-arg: got %q, want GoBack", r)
	}
}

// TestRule3_GenuineCollision: two selectors producing the same Go name get
// the colon-count suffix.
func TestRule3_GenuineCollision(t *testing.T) {
	m1 := macosplatformmetadata.Method{Selector: "transformContextForBox:"}
	m2 := macosplatformmetadata.Method{Selector: "transformContext:forBox:"}
	ms := []macosplatformmetadata.Method{m1, m2}
	counts := map[string]int{"TransformContextForBox": 2}

	r1, skip1 := naming.ResolveGoMethodName("TransformContextForBox", m1, ms, counts)
	r2, skip2 := naming.ResolveGoMethodName("TransformContextForBox", m2, ms, counts)

	if skip1 || skip2 {
		t.Error("neither method should be skipped in a genuine collision")
	}
	if r1 != "TransformContextForBox1" {
		t.Errorf("m1: got %q, want TransformContextForBox1", r1)
	}
	if r2 != "TransformContextForBox2" {
		t.Errorf("m2: got %q, want TransformContextForBox2", r2)
	}
}

// TestUniqueNamePassthrough: goNameCount == 1 always returns the base name.
func TestUniqueNamePassthrough(t *testing.T) {
	m := macosplatformmetadata.Method{Selector: "doSomething"}
	ms := []macosplatformmetadata.Method{m}
	counts := map[string]int{"DoSomething": 1}

	r, skip := naming.ResolveGoMethodName("DoSomething", m, ms, counts)
	if skip || r != "DoSomething" {
		t.Errorf("unique method: got %q skip=%v, want DoSomething skip=false", r, skip)
	}
}

// TestClassMethodPassthrough: class methods skip all three rules and always
// return the base name unchanged. The classes.go secondary dedup (_2/_3)
// handles any remaining collision.
func TestClassMethodPassthrough(t *testing.T) {
	m := macosplatformmetadata.Method{Selector: "sharedManager", IsClassMethod: true}
	sibling := macosplatformmetadata.Method{Selector: "sharedManager:", IsClassMethod: true}
	ms := []macosplatformmetadata.Method{m, sibling}
	counts := map[string]int{"SharedManager": 2}

	r, skip := naming.ResolveGoMethodName("SharedManager", m, ms, counts)
	if skip {
		t.Error("class method should not be skipped")
	}
	if r != "SharedManager" {
		t.Errorf("class method: got %q, want SharedManager (unchanged)", r)
	}
}

// TestIsIBActionSender covers all annotated id spellings.
func TestIsIBActionSender(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"id", true},
		{"id _Nullable", true},
		{"id _Nonnull", true},
		{"_Nullable id", true},
		{"_Nonnull id", true},
		{"NSObject *", false},
		{"", false},
		{"NSString *", false},
	}
	for _, c := range cases {
		if got := naming.IsIBActionSender(c.in); got != c.want {
			t.Errorf("IsIBActionSender(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestFindSiblingMethod confirms correct selector and class-method matching.
func TestFindSiblingMethod(t *testing.T) {
	inst := macosplatformmetadata.Method{Selector: "open", IsClassMethod: false}
	cls := macosplatformmetadata.Method{Selector: "open", IsClassMethod: true}
	ms := []macosplatformmetadata.Method{inst, cls}

	if got, ok := naming.FindSiblingMethod(ms, "open", false); !ok || got.Selector != "open" || got.IsClassMethod {
		t.Errorf("FindSiblingMethod instance: ok=%v got=%v", ok, got)
	}
	if got, ok := naming.FindSiblingMethod(ms, "open", true); !ok || !got.IsClassMethod {
		t.Errorf("FindSiblingMethod class: ok=%v got=%v", ok, got)
	}
	if _, ok := naming.FindSiblingMethod(ms, "close", false); ok {
		t.Error("FindSiblingMethod should return false for unknown selector")
	}
}
