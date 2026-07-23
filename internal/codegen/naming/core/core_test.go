package core

import "testing"

func TestCapitalise(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"", ""}, {"foo", "Foo"}, {"Foo", "Foo"}, {"éa", "Éa"},
	} {
		if got := Capitalise(c.in); got != c.want {
			t.Errorf("Capitalise(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestMethodName(t *testing.T) {
	for _, c := range []struct{ sel, want string }{
		{"objectAtIndex:", "ObjectAtIndex"},
		{"writeToURL:error:", "WriteToURLError"},
		{"count", "Count"},
		{"enumerateObjectsUsingBlock:", "EnumerateObjectsUsing"},
		{"doThingWithBlock:", "DoThingWith"},
	} {
		if got := MethodName(c.sel); got != c.want {
			t.Errorf("MethodName(%q) = %q; want %q", c.sel, got, c.want)
		}
	}
}

func TestExportedFunctionAndTypeName(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"", ""},
		{"vmnet_start_interface", "VmnetStartInterface"},
		{"audit_token_t", "AuditTokenT"},
		{"CFArrayCreate", "CFArrayCreate"}, // already exported → unchanged
		{"_MPIsFullyInitialized", "MPIsFullyInitialized"},
	} {
		if got := ExportedFunctionName(c.in); got != c.want {
			t.Errorf("ExportedFunctionName(%q) = %q; want %q", c.in, got, c.want)
		}
		if got := ExportedTypeName(c.in); got != c.want {
			t.Errorf("ExportedTypeName(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestGoTypeName(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"", ""}, {"audit_token_t", "Audit_token_t"}, {"NSRange", "NSRange"},
	} {
		if got := GoTypeName(c.in); got != c.want {
			t.Errorf("GoTypeName(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestProtocolGoTypeName(t *testing.T) {
	owners := map[string]string{"NSObject": "Foundation"}
	if got := ProtocolGoTypeName("NSObject", owners); got != "NSObjectProtocol" {
		t.Errorf("ProtocolGoTypeName(NSObject, class-owner) = %q; want NSObjectProtocol", got)
	}
	if got := ProtocolGoTypeName("NSCopying", owners); got != "NSCopying" {
		t.Errorf("ProtocolGoTypeName(NSCopying) = %q; want NSCopying", got)
	}
	if got := ProtocolGoTypeName("NSObject", nil); got != "NSObject" {
		t.Errorf("ProtocolGoTypeName(NSObject, nil) = %q; want NSObject", got)
	}
}

func TestPackageName(t *testing.T) {
	if got := PackageName("CoreML"); got != "coreml" {
		t.Errorf("PackageName(CoreML) = %q; want coreml", got)
	}
}
