package typemap

// primitiveGoType maps C/ObjC primitive type names to Go types.
// Returns empty string if not a known primitive.
func primitiveGoType(n string) string {
	switch n {
	case "BOOL", "Boolean":
		return "bool"
	case "bool", "_Bool":
		return "bool"
	case "char", "signed char":
		return "int8"
	case "unsigned char":
		return "uint8"
	case "short", "signed short", "short int":
		return "int16"
	case "unsigned short", "unsigned short int":
		return "uint16"
	case "int", "signed", "signed int":
		return "int32"
	case "unsigned", "unsigned int":
		return "uint32"
	case "long", "signed long", "long int":
		return "int64"
	case "unsigned long", "unsigned long int":
		return "uint64"
	case "long long", "signed long long", "long long int":
		return "int64"
	case "unsigned long long", "unsigned long long int":
		return "uint64"
	case "float":
		return "float32"
	case "double", "long double":
		return "float64"
	case "int8_t":
		return "int8"
	case "int16_t":
		return "int16"
	case "int32_t":
		return "int32"
	case "int64_t":
		return "int64"
	case "uint8_t":
		return "uint8"
	case "uint16_t":
		return "uint16"
	case "uint32_t":
		return "uint32"
	case "uint64_t":
		return "uint64"
	case "intptr_t", "ptrdiff_t":
		return "int64"
	case "uintptr_t", "size_t":
		return "uint64"
	case "NSInteger":
		return "int64"
	case "NSUInteger":
		return "uint64"
	case "CGFloat":
		return "float64"
	case "NSTimeInterval":
		return "float64"
	case "unichar":
		return "uint16"
	case "wchar_t":
		return "uint32"
	// CoreFoundation scalar typedefs
	case "CFIndex", "CFByteOrder":
		return "int64"
	case "CFOptionFlags", "CFTypeID", "CFHashCode", "CFAllocatorTypeID":
		return "uint64"
	case "CFTimeInterval", "CFAbsoluteTime", "Float64":
		return "float64"
	case "Float32":
		return "float32"
	case "CFStringEncoding", "CFBit":
		return "uint32"
	// CoreGraphics float typedefs
	case "CGDisplayFadeInterval", "CGDisplayReservationInterval", "CGGammaValue":
		return "float32"
	case "CGRefreshRate":
		return "float64"
	// OpenGL / OpenAL scalar typedefs
	case "GLfloat", "GLclampf", "ALfloat", "ALclampf":
		return "float32"
	case "GLdouble", "GLclampd", "ALdouble":
		return "float64"
	case "ALvoid":
		return "unsafe.Pointer"
	// AppKit / UIKit float typedefs
	case "NSAnimationProgress", "NSLayoutPriority", "NSStackViewVisibilityPriority", "NSTouchBarItemPriority":
		return "float32"
	case "NSFontWeight", "NSFontWidth", "NSSliderAccessoryWidth":
		return "float64"
	case "CACurrentMediaTime":
		return "float64"
	// AVFoundation scalar typedefs
	case "AVCaptureDeviceTransportControlsSpeed":
		return "float32"
	// C99 floating-point typedefs (math.h)
	case "float_t":
		return "float32"
	case "double_t":
		return "float64"
	// CoreLocation scalar typedefs (all typedef double)
	case "CLLocationDegrees", "CLLocationAccuracy", "CLLocationSpeed",
		"CLLocationSpeedAccuracy", "CLLocationDirection",
		"CLLocationDirectionAccuracy", "CLLocationDistance":
		return "float64"
	case "CLHeadingComponentValue":
		return "float64"
	// CompositorServices scalar typedef (typedef float)
	case "cp_render_quality_t":
		return "float32"
	// SensorKit time typedef (typedef double)
	case "SRAbsoluteTime":
		return "float64"
	// NaturalLanguage distance typedef (typedef double)
	case "NLDistance":
		return "float64"
	}
	return ""
}
