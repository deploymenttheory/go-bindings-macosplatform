package naming

import "testing"

func TestLowerLeadingAcronym(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// Single leading uppercase — normal camelCase.
		{"ObjectAtIndex", "objectAtIndex"},
		{"InitWithURL", "initWithURL"},
		{"IsLoaded", "isLoaded"},

		// Leading acronym followed by a new camelCase word.
		{"URLForAuxiliaryExecutable", "urlForAuxiliaryExecutable"},
		{"HTTPBody", "httpBody"},
		{"JSONObject", "jsonObject"},
		{"URLForResourceWithExtension", "urlForResourceWithExtension"},

		// Leading acronym followed by a single-char lowercase suffix then uppercase.
		{"URLsForResources", "urlsForResources"},

		// Standalone acronym (no word follows).
		{"URL", "url"},
		{"HTTP", "http"},

		// Already lowercase — unchanged.
		{"objectAtIndex", "objectAtIndex"},

		// Empty — unchanged.
		{"", ""},
	}

	for _, c := range cases {
		got := lowerLeadingAcronym(c.in)
		if got != c.want {
			t.Errorf("lowerLeadingAcronym(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBridgeFuncName(t *testing.T) {
	cases := []struct {
		framework, class, selector string
		isClass                    bool
		want                       string
	}{
		{"Foundation", "NSArray", "objectAtIndex:", false, "foundation_NSArray_objectAtIndex"},
		{"Foundation", "NSBundle", "URLForAuxiliaryExecutable:", false, "foundation_NSBundle_urlForAuxiliaryExecutable"},
		{"Foundation", "NSBundle", "URLsForResourcesWithExtension:subdirectory:", false, "foundation_NSBundle_urlsForResourcesWithExtensionSubdirectory"},
		{"Foundation", "NSDate", "timeIntervalSinceReferenceDate", true, "foundation_NSDate_timeIntervalSinceReferenceDate_cls"},
		{"AppKit", "NSWindow", "initWithContentRect:styleMask:backing:defer:", false, "appkit_NSWindow_initWithContentRectStyleMaskBackingDefer"},
	}

	for _, c := range cases {
		got := BridgeFuncName(c.framework, c.class, c.selector, c.isClass)
		if got != c.want {
			t.Errorf("BridgeFuncName(%q, %q, %q, %v) = %q, want %q",
				c.framework, c.class, c.selector, c.isClass, got, c.want)
		}
	}
}

func TestMethodName(t *testing.T) {
	cases := []struct {
		selector, want string
	}{
		// No colon — plain property getter.
		{"count", "Count"},
		// Single colon.
		{"objectAtIndex:", "ObjectAtIndex"},
		// Multiple colons joined without separator.
		{"setObject:forKey:", "SetObjectForKey"},
		// "UsingBlock" suffix is rewritten to "Using".
		{"enumerateObjectsUsingBlock:", "EnumerateObjectsUsing"},
		// initWith* family.
		{"initWithFrame:", "InitWithFrame"},
		// isEqual family.
		{"isEqual:", "IsEqual"},
		// performBlock — WithBlock suffix rewritten to With.
		{"performBlock:", "PerformBlock"},
		// Three-part selector: colons join all parts.
		{"initWithContentsOfURL:encoding:error:", "InitWithContentsOfURLEncodingError"},
		// "WithBlock" suffix variant.
		{"animateWithDuration:animations:withBlock:", "AnimateWithDurationAnimationsWith"},
	}

	for _, c := range cases {
		got := MethodName(c.selector)
		if got != c.want {
			t.Errorf("MethodName(%q) = %q, want %q", c.selector, got, c.want)
		}
	}
}

func TestArgName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// Already lowercase — no change.
		{"index", "index"},
		// Leading uppercase lowercased (acts like an acronym first letter).
		{"URL", "uRL"},
		// "self" is not a Go keyword.
		{"self", "self"},
		// Go keywords get a trailing underscore.
		{"type", "type_"},
		{"default", "default_"},
		{"return", "return_"},
		{"range", "range_"},
		{"error", "error_"},
		{"func", "func_"},
		// Empty name falls back to "arg".
		{"", "arg"},
		// Leading uppercase lowercased.
		{"Count", "count"},
		{"URLString", "uRLString"},
	}

	for _, c := range cases {
		got := ParamName(c.in)
		if got != c.want {
			t.Errorf("ParamName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPackageName(t *testing.T) {
	cases := []struct {
		framework, want string
	}{
		{"Foundation", "foundation"},
		{"CoreML", "coreml"},
		{"AppKit", "appkit"},
		{"AVFoundation", "avfoundation"},
		{"", ""},
	}

	for _, c := range cases {
		got := PackageName(c.framework)
		if got != c.want {
			t.Errorf("PackageName(%q) = %q, want %q", c.framework, got, c.want)
		}
	}
}

func TestGoTypeName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// Already exported — unchanged.
		{"NSString", "NSString"},
		// Lowercase first letter — capitalised.
		{"nsstring", "Nsstring"},
		{"abc", "Abc"},
		// Empty — unchanged.
		{"", ""},
	}

	for _, c := range cases {
		got := GoTypeName(c.in)
		if got != c.want {
			t.Errorf("GoTypeName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBlockTypeName(t *testing.T) {
	cases := []struct {
		className, selector, want string
	}{
		// MethodName → "EnumerateObjectsUsing"; TrimSuffix("Using") → "EnumerateObjects"
		{
			"NSArray",
			"enumerateObjectsUsingBlock:",
			"NSArrayEnumerateObjectsBlock",
		},
		// MethodName → "EnumerateKeysAndObjectsUsing"; TrimSuffix("Using") → "EnumerateKeysAndObjects"
		{
			"NSDictionary",
			"enumerateKeysAndObjectsUsingBlock:",
			"NSDictionaryEnumerateKeysAndObjectsBlock",
		},
		// MethodName → "PerformBlock"; TrimSuffix("Block") → "Perform"
		{
			"NSObject",
			"performBlock:",
			"NSObjectPerformBlock",
		},
	}

	for _, c := range cases {
		got := BlockTypeName(c.className, c.selector)
		if got != c.want {
			t.Errorf("BlockTypeName(%q, %q) = %q, want %q", c.className, c.selector, got, c.want)
		}
	}
}
