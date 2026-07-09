package idiofw

import "testing"

// TestRepairMethodNames locks the conservative second-pass renames: error
// labels, redundant Options labels, and With<TypeEcho> segments are stripped
// only when the shorter name is free, and curated names are never touched.
func TestRepairMethodNames(t *testing.T) {
	methods := []methodModel{
		// Error-returning plain method drops its Error label.
		{kind: kindPlain, goName: "DataWithContentsOfFileOptionsError", plainHasError: true,
			plainParams: []plainParamModel{{goName: "path", goType: "string"}, {goName: "options", goType: "DataReadingOptions"}}},
		// …then the Options rule chains on the same method.
		// (options param type ends in Options → the label is redundant.)

		// Error-returning slice method drops its Error label.
		{kind: kindSlice, goName: "ContentsOfDirectoryAtPathError", sliceHasError: true},

		// With<TypeEcho>: the sole param's type repeats the name's tail.
		{kind: kindPlain, goName: "CommonPrefixWithString",
			plainParams: []plainParamModel{{goName: "str", goType: "string"}}},

		// A wrapper-typed echo also strips ("Data" echoes *Data).
		{kind: kindPlain, goName: "AppendWithData",
			plainParams: []plainParamModel{{goName: "data", goType: "*Data"}}},

		// A non-echoing With segment is kept.
		{kind: kindPlain, goName: "EncodeWithCoder",
			plainParams: []plainParamModel{{goName: "coder", goType: "obj.Object"}}},

		// The stripped name is already taken → keep the original.
		{kind: kindPlain, goName: "Validate", plainParams: nil},
		{kind: kindPlain, goName: "ValidateError", plainHasError: true},

		// A curated name is never repaired.
		{kind: kindPlain, goName: "SaveError", plainHasError: true, nameCurated: true},
	}
	repairMethodNames(methods)

	want := []string{
		"DataWithContentsOfFile", // Error stripped, then Options stripped
		"ContentsOfDirectoryAtPath",
		"CommonPrefix",
		"Append",
		"EncodeWithCoder",
		"Validate",
		"ValidateError", // "Validate" taken
		"SaveError",     // curated
	}
	for i, w := range want {
		if methods[i].goName != w {
			t.Errorf("methods[%d].goName = %q; want %q", i, methods[i].goName, w)
		}
	}
}

func TestRepairConstructorNames(t *testing.T) {
	ctors := []constructorModel{
		{goName: "NewStringWithContentsOfFileEncodingError", hasNSError: true},
		{goName: "NewMachineError", hasNSError: true, nameCurated: true}, // curated: untouched
		{goName: "NewData"}, // no error: untouched
	}
	repairConstructorNames(ctors)
	want := []string{
		"NewStringWithContentsOfFileEncoding",
		"NewMachineError",
		"NewData",
	}
	for i, w := range want {
		if ctors[i].goName != w {
			t.Errorf("ctors[%d].goName = %q; want %q", i, ctors[i].goName, w)
		}
	}
}

func TestSafeParamNameSubstitutions(t *testing.T) {
	cases := map[string]string{
		"string_":    "str",
		"bytes_":     "data",
		"len_":       "length",
		"id_":        "identifier",
		"error_":     "err",
		"obj":        "object",
		"rt":         "rt_",
		"anything":   "anything",
		"identifier": "identifier",
	}
	for in, want := range cases {
		if got := safeParamName(in); got != want {
			t.Errorf("safeParamName(%q) = %q; want %q", in, got, want)
		}
	}
}
