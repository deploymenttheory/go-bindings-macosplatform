package typemap

import "testing"

func TestIsStructCTypeKnown(t *testing.T) {
	known := []string{
		"NSRect", "NSRange", "CGRect", "CGSize", "CGPoint",
		"CMTime", "CATransform3D", "MTLSize", "MTLRegion",
		"simd_float4", "vector_float3", "matrix_float4x4",
		"CLLocationCoordinate2D", "CVTimeStamp",
	}
	for _, name := range known {
		if !IsStructCType(name) {
			t.Errorf("IsStructCType(%q) = false, want true", name)
		}
	}
}

func TestIsStructCTypeUnknown(t *testing.T) {
	for _, name := range []string{"NSString", "id", "CGFloat", "NSInteger", "UnknownStruct", ""} {
		if IsStructCType(name) {
			t.Errorf("IsStructCType(%q) = true, want false", name)
		}
	}
}

func TestIsStructCTypeNormalisesInput(t *testing.T) {
	// Normalise strips qualifiers; the underlying name should still match.
	if !IsStructCType("NSRect") {
		t.Error("IsStructCType(NSRect) = false, want true")
	}
}
