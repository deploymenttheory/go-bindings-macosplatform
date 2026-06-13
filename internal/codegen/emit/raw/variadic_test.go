package raw

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
		"func NSArrayOf[T objc.Object](ctx context.Context, objects ...T) *NSArray[T]",
		"func NSMutableArrayOf[T objc.Object](ctx context.Context, objects ...T) *NSMutableArray[T]",
		"func NSSetOf[T objc.Object](ctx context.Context, objects ...T) *NSSet[T]",
		"func NSMutableSetOf[T objc.Object](ctx context.Context, objects ...T) *NSMutableSet[T]",
		"func NSDictionaryOf(ctx context.Context, objectsAndKeys ...objc.Object) *NSDictionary[objc.Object]",
		"func NSMutableDictionaryOf(ctx context.Context, objectsAndKeys ...objc.Object) *NSMutableDictionary",
		"goBindings_NSDictionaryFromPairs",
		"NSMutableArrayArrayWithCapacity(ctx,",
		"NSMutableSetSetWithCapacity(ctx,",
		"AddObject(ctx, o)",
		"NewNSArrayT[T]",
		"NewNSSetT[T]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in variadic wrappers output:\n%s", want, out)
		}
	}
}
