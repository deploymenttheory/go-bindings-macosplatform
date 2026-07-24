package typemap

// cfTypedefSet is the complete set of CoreFoundation opaque reference typedef
// names. These appear in function/method signatures as bare names (e.g.
// "CFStringRef") because the pointer is implicit in the CF typedef definition.
var cfTypedefSet = map[string]bool{
	"CFAllocatorRef":               true,
	"CFArrayRef":                   true,
	"CFAttributedStringRef":        true,
	"CFBagRef":                     true,
	"CFBinaryHeapRef":              true,
	"CFBitVectorRef":               true,
	"CFBooleanRef":                 true,
	"CFBundleRef":                  true,
	"CFCalendarRef":                true,
	"CFCharacterSetRef":            true,
	"CFDataRef":                    true,
	"CFDateFormatterRef":           true,
	"CFDateRef":                    true,
	"CFDictionaryRef":              true,
	"CFErrorRef":                   true,
	"CFFileDescriptorRef":          true,
	"CFFileSecurityRef":            true,
	"CFLocaleRef":                  true,
	"CFMachPortRef":                true,
	"CFMessagePortRef":             true,
	"CFMutableArrayRef":            true,
	"CFMutableAttributedStringRef": true,
	"CFMutableBagRef":              true,
	"CFMutableBitVectorRef":        true,
	"CFMutableCharacterSetRef":     true,
	"CFMutableDataRef":             true,
	"CFMutableDictionaryRef":       true,
	"CFMutableSetRef":              true,
	"CFMutableStringRef":           true,
	"CFNotificationCenterRef":      true,
	"CFNullRef":                    true,
	"CFNumberFormatterRef":         true,
	"CFNumberRef":                  true,
	"CFPlugInInstanceRef":          true,
	"CFPlugInRef":                  true,
	"CFReadStreamRef":              true,
	"CFRunLoopObserverRef":         true,
	"CFRunLoopRef":                 true,
	"CFRunLoopSourceRef":           true,
	"CFRunLoopTimerRef":            true,
	"CFSetRef":                     true,
	"CFSocketRef":                  true,
	"CFStringRef":                  true,
	"CFStringTokenizerRef":         true,
	"CFTimeZoneRef":                true,
	"CFTreeRef":                    true,
	"CFURLEnumeratorRef":           true,
	"CFURLRef":                     true,
	"CFUUIDRef":                    true,
	"CFUserNotificationRef":        true,
	"CFWriteStreamRef":             true,
	"CFXMLNodeRef":                 true,
	"CFXMLParserRef":               true,
}

// IsCoreFoundationOpaqueRef reports whether name is one of the well-known
// CoreFoundation opaque-pointer typedefs (CFStringRef, CFArrayRef, …) that
// the type mapper routes to *corefoundation.<Name>. Exposed so the pipeline
// can attribute these typedefs to CoreFoundation in cycle detection.
//
// Pipeline-specific: the frameworks typemap has a same-named function with a
// DIFFERENT implementation (a CF/CG-suffix pattern match, not this fixed
// cfTypedefSet lookup) and different results. They are not interchangeable — see
// the package doc's SCOPE note.
func IsCoreFoundationOpaqueRef(name string) bool {
	return cfTypedefSet[name]
}
