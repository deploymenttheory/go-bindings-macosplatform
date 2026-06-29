package rawlib

import (
	"bytes"
	"strings"
	"testing"
)

func TestFoundationVariadicWrappers(t *testing.T) {
	var buf bytes.Buffer
	if err := EmitFoundationVariadicWrappers(&buf, "foundation"); err != nil {
		t.Fatalf("FoundationVariadicWrappers error: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"package foundation",
		"func NSArrayOf[T cgo.Object](objects ...T) *NSArray[T]",
		"func NSMutableArrayOf[T cgo.Object](objects ...T) *NSMutableArray[T]",
		"func NSSetOf[T cgo.Object](objects ...T) *NSSet[T]",
		"func NSMutableSetOf[T cgo.Object](objects ...T) *NSMutableSet[T]",
		"func NSDictionaryOf(objectsAndKeys ...cgo.Object) *NSDictionary[cgo.Object]",
		"func NSMutableDictionaryOf(objectsAndKeys ...cgo.Object) *NSMutableDictionary",
		"goBindings_NSDictionaryFromPairs",
		"NSMutableArrayArrayWithCapacity(",
		"NSMutableSetSetWithCapacity(",
		"AddObject(o)",
		"NewNSArrayT[T]",
		"NewNSSetT[T]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in variadic wrappers output:\n%s", want, out)
		}
	}
}
