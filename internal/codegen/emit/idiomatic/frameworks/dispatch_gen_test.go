//go:build darwin

package idiofw

import "testing"

func TestClassifyGoType(t *testing.T) {
	enums := map[string]bool{"ComparisonResult": true, "VirtualMachineState": true}
	objects := map[string]bool{"String": true, "VirtualMachine": true}
	isEnum := func(n string) bool { return enums[n] }
	isObject := func(n string) bool { return objects[n] }

	cases := []struct {
		typ  string
		want objKind
	}{
		{"", kindVoid},
		{"void", kindVoid},
		{"string", kindString},
		{"bool", kindBool},
		{"int", kindScalar},
		{"uint", kindScalar},
		{"float64", kindScalar},
		{"*String", kindObject},
		{"*virtualization.VirtualMachine", kindObject},
		{"obj.Object", kindObject},
		{"foundation.Object", kindObject},
		{"ComparisonResult", kindEnum},
		{"foundation.ComparisonResult", kindEnum},
		{"VirtualMachine", kindObject}, // bare wrapper name via isObject
		{"UnknownThing", kindScalar},   // unknown bare name -> scalar fallback
	}
	for _, c := range cases {
		if got := classifyGoType(c.typ, isEnum, isObject); got != c.want {
			t.Errorf("classifyGoType(%q) = %v, want %v", c.typ, got, c.want)
		}
	}
}

func TestMarshalArg(t *testing.T) {
	cases := []struct {
		name string
		k    objKind
		in   string
		want string
	}{
		{"string", kindString, "name", "purego.NSString(name)"},
		{"object", kindObject, "cfg", "objref.IDOf(cfg)"},
		{"scalar", kindScalar, "n", "n"},
		{"bool", kindBool, "flag", "flag"},
		{"enum", kindEnum, "mode", "mode"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := marshalArg(c.in, c.k); got != c.want {
				t.Errorf("marshalArg(%q, %v) = %q, want %q", c.in, c.k, got, c.want)
			}
		})
	}
}

func TestSendReturnType(t *testing.T) {
	cases := []struct {
		k    objKind
		typ  string
		want string
	}{
		{kindString, "string", "objc.ID"},
		{kindObject, "*String", "objc.ID"},
		{kindScalar, "int", "int"},
		{kindBool, "bool", "bool"},
		{kindEnum, "ComparisonResult", "ComparisonResult"},
		{kindVoid, "", ""},
	}
	for _, c := range cases {
		if got := sendReturnType(c.k, c.typ); got != c.want {
			t.Errorf("sendReturnType(%v, %q) = %q, want %q", c.k, c.typ, got, c.want)
		}
	}
}

func TestMarshalReturn(t *testing.T) {
	cases := []struct {
		name     string
		k        objKind
		recv     string
		typ      string
		wrapExpr string
		want     string
	}{
		{"void", kindVoid, "_r", "", "", ""},
		{"string", kindString, "_r", "string", "", "if _r == 0 {\nreturn \"\"\n}\nreturn purego.GoString(_r)"},
		{"object", kindObject, "_r", "*String", "stringFromID(%s)", "return stringFromID(_r)"},
		{"object obj.Wrap", kindObject, "_id", "obj.Object", "obj.Wrap(%s)", "return obj.Wrap(_id)"},
		{"scalar", kindScalar, "_r", "int", "", "return _r"},
		{"bool", kindBool, "_r", "bool", "", "return _r"},
		{"enum", kindEnum, "_r", "ComparisonResult", "", "return _r"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := marshalReturn(c.k, c.recv, c.typ, c.wrapExpr); got != c.want {
				t.Errorf("marshalReturn(%v, %q, %q, %q) = %q, want %q",
					c.k, c.recv, c.typ, c.wrapExpr, got, c.want)
			}
		})
	}
}

func TestZeroValue(t *testing.T) {
	cases := []struct {
		k    objKind
		typ  string
		want string
	}{
		{kindVoid, "", ""},
		{kindString, "string", `""`},
		{kindObject, "*String", "nil"},
		{kindObject, "obj.Object", "nil"},
		{kindObject, "", "nil"},
		{kindArray, "[]string", "nil"},
		{kindObject, "map[string]obj.Object", "nil"},
		// Distinct struct-wrapper handle types have no nil; their zero is the empty
		// struct literal, bare or package-qualified.
		{kindObject, "CFArrayRef", "CFArrayRef{}"},
		{kindObject, "coregraphics.CGImageRef", "coregraphics.CGImageRef{}"},
		{kindBool, "bool", "false"},
		{kindEnum, "ComparisonResult", "ComparisonResult(0)"},
		{kindScalar, "int", "0"},
	}
	for _, c := range cases {
		if got := zeroValue(c.k, c.typ); got != c.want {
			t.Errorf("zeroValue(%v, %q) = %q, want %q", c.k, c.typ, got, c.want)
		}
	}
}

func TestSelectorIdent(t *testing.T) {
	cases := []struct {
		selector string
		want     string
	}{
		{"count", "Count"},
		{"objectAtIndex:", "ObjectAtIndex"},
		{"initWithURL:readOnly:error:", "InitWithURLReadOnlyError"},
		{"componentsSeparatedByString:", "ComponentsSeparatedByString"},
		{"", ""},
		{":", ""},
		{"a::b:", "AB"},
	}
	for _, c := range cases {
		if got := selectorIdent(c.selector); got != c.want {
			t.Errorf("selectorIdent(%q) = %q, want %q", c.selector, got, c.want)
		}
	}
}

func TestSelectorVarName(t *testing.T) {
	got := selectorVarName("String", "componentsSeparatedByString:")
	want := "_selStringComponentsSeparatedByString"
	if got != want {
		t.Errorf("selectorVarName = %q, want %q", got, want)
	}
	// Same selector, different class -> distinct var (no collision).
	other := selectorVarName("Array", "componentsSeparatedByString:")
	if other == got {
		t.Errorf("selectorVarName should differ by class: %q == %q", other, got)
	}
}

func TestCapitalizeFirst(t *testing.T) {
	cases := map[string]string{
		"":         "",
		"a":        "A",
		"readOnly": "ReadOnly",
		"URL":      "URL",
		"é":        "É",
	}
	for in, want := range cases {
		if got := capitalizeFirst(in); got != want {
			t.Errorf("capitalizeFirst(%q) = %q, want %q", in, got, want)
		}
	}
}
