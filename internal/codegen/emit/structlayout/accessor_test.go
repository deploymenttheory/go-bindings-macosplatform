package structlayout

import "testing"

func TestAccessorPlan(t *testing.T) {
	names := []string{"mChunkType", "_reserved", "", "mChunkSize"}
	offsets := []int{0, 4, 8, 4}
	goTypes := []string{"uint32", "uint8", "uint8", "int64"}
	got := AccessorPlan(names, offsets, goTypes)
	// The underscore-prefixed and anonymous members carry no accessor.
	if len(got) != 2 {
		t.Fatalf("got %d accessors, want 2: %+v", len(got), got)
	}
	if got[0] != (Accessor{"mChunkType", 0, "uint32"}) || got[1] != (Accessor{"mChunkSize", 4, "int64"}) {
		t.Errorf("unexpected plan: %+v", got)
	}
	// Mismatched lengths yield no plan (defensive).
	if AccessorPlan(names, offsets[:1], goTypes) != nil {
		t.Error("mismatched lengths should yield nil")
	}
}
