package typemap

import "testing"

func TestPrimitiveGoTypeIntegers(t *testing.T) {
	cases := []struct{ qt, want string }{
		{"char", "int8"},
		{"signed char", "int8"},
		{"unsigned char", "uint8"},
		{"short", "int16"},
		{"signed short", "int16"},
		{"short int", "int16"},
		{"unsigned short", "uint16"},
		{"unsigned short int", "uint16"},
		{"int", "int32"},
		{"signed", "int32"},
		{"signed int", "int32"},
		{"unsigned", "uint32"},
		{"unsigned int", "uint32"},
		{"long", "int64"},
		{"signed long", "int64"},
		{"long int", "int64"},
		{"unsigned long", "uint64"},
		{"unsigned long int", "uint64"},
		{"long long", "int64"},
		{"signed long long", "int64"},
		{"long long int", "int64"},
		{"unsigned long long", "uint64"},
		{"unsigned long long int", "uint64"},
		{"int8_t", "int8"},
		{"int16_t", "int16"},
		{"int32_t", "int32"},
		{"int64_t", "int64"},
		{"uint8_t", "uint8"},
		{"uint16_t", "uint16"},
		{"uint32_t", "uint32"},
		{"uint64_t", "uint64"},
		{"intptr_t", "int64"},
		{"ptrdiff_t", "int64"},
		{"uintptr_t", "uint64"},
		{"size_t", "uint64"},
		{"NSInteger", "int64"},
		{"NSUInteger", "uint64"},
		{"unichar", "uint16"},
		{"wchar_t", "uint32"},
	}
	for _, c := range cases {
		if got := primitiveGoType(c.qt); got != c.want {
			t.Errorf("primitiveGoType(%q) = %q, want %q", c.qt, got, c.want)
		}
	}
}

func TestPrimitiveGoTypeFloats(t *testing.T) {
	cases := []struct{ qt, want string }{
		{"float", "float32"},
		{"double", "float64"},
		{"long double", "float64"},
		{"CGFloat", "float64"},
		{"NSTimeInterval", "float64"},
		{"float_t", "float32"},
		{"double_t", "float64"},
		{"Float32", "float32"},
		{"Float64", "float64"},
		{"CFTimeInterval", "float64"},
		{"CFAbsoluteTime", "float64"},
		{"CGRefreshRate", "float64"},
		{"GLfloat", "float32"},
		{"GLclampf", "float32"},
		{"GLdouble", "float64"},
		{"GLclampd", "float64"},
		{"ALfloat", "float32"},
		{"ALclampf", "float32"},
		{"ALdouble", "float64"},
		{"NSAnimationProgress", "float32"},
		{"NSLayoutPriority", "float32"},
		{"NSStackViewVisibilityPriority", "float32"},
		{"NSTouchBarItemPriority", "float32"},
		{"NSFontWeight", "float64"},
		{"NSFontWidth", "float64"},
		{"NSSliderAccessoryWidth", "float64"},
		{"CACurrentMediaTime", "float64"},
		{"AVCaptureDeviceTransportControlsSpeed", "float32"},
		{"CGDisplayFadeInterval", "float32"},
		{"CGDisplayReservationInterval", "float32"},
		{"CGGammaValue", "float32"},
		{"CLLocationDegrees", "float64"},
		{"CLLocationAccuracy", "float64"},
		{"CLLocationSpeed", "float64"},
		{"CLLocationDirection", "float64"},
		{"CLLocationDistance", "float64"},
		{"CLHeadingComponentValue", "float64"},
		{"cp_render_quality_t", "float32"},
		{"SRAbsoluteTime", "float64"},
		{"NLDistance", "float64"},
	}
	for _, c := range cases {
		if got := primitiveGoType(c.qt); got != c.want {
			t.Errorf("primitiveGoType(%q) = %q, want %q", c.qt, got, c.want)
		}
	}
}

func TestPrimitiveGoTypeBooleans(t *testing.T) {
	for _, qt := range []string{"BOOL", "Boolean", "bool", "_Bool"} {
		if got := primitiveGoType(qt); got != "bool" {
			t.Errorf("primitiveGoType(%q) = %q, want bool", qt, got)
		}
	}
}

func TestPrimitiveGoTypeCFIntegers(t *testing.T) {
	cases := []struct{ qt, want string }{
		{"CFIndex", "int64"},
		{"CFByteOrder", "int64"},
		{"CFOptionFlags", "uint64"},
		{"CFTypeID", "uint64"},
		{"CFHashCode", "uint64"},
		{"CFAllocatorTypeID", "uint64"},
		{"CFStringEncoding", "uint32"},
		{"CFBit", "uint32"},
	}
	for _, c := range cases {
		if got := primitiveGoType(c.qt); got != c.want {
			t.Errorf("primitiveGoType(%q) = %q, want %q", c.qt, got, c.want)
		}
	}
}

func TestPrimitiveGoTypeALvoid(t *testing.T) {
	// ALvoid is a special case: bridged as void* not truly void
	got := primitiveGoType("ALvoid")
	if got != "unsafe.Pointer" {
		t.Errorf("primitiveGoType(ALvoid) = %q, want unsafe.Pointer", got)
	}
}

func TestPrimitiveGoTypeUnknownReturnsEmpty(t *testing.T) {
	for _, qt := range []string{"NSString", "id", "void", "UnknownType", ""} {
		if got := primitiveGoType(qt); got != "" {
			t.Errorf("primitiveGoType(%q) = %q, want empty string", qt, got)
		}
	}
}
