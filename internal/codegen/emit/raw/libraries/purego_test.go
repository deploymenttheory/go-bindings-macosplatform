package rawlib

import (
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
)

// TestPuregoArg covers the wrapper-argument classification for the purego
// backend: the func-var parameter type purego registers and the call
// expression, mirroring goCGoArgExpr branch for branch.
func TestPuregoArg(t *testing.T) {
	m := typemap.New()

	cases := []struct {
		name        string
		goType      string
		wantVarType string
		wantCall    string
		wantOK      bool
	}{
		{"scalar", "int32", "int32", "x", true},
		{"string is purego-native", "string", "string", "x", true},
		{"unsafe pointer", "unsafe.Pointer", "unsafe.Pointer", "x", true},
		{"pointer to primitive", "*uint64", "*uint64", "x", true},
		{"handle wrapper extracts Ptr", "*XarT", "unsafe.Pointer", "_ptr_x", true},
		{"objc object unsupported", "cgo.Object", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var preambles, keepAlives []string
			varType, call, ok := puregoArg(c.goType, "x", m, &preambles, &keepAlives)
			if ok != c.wantOK || varType != c.wantVarType || call != c.wantCall {
				t.Errorf("puregoArg(%q) = (%q, %q, %v); want (%q, %q, %v)",
					c.goType, varType, call, ok, c.wantVarType, c.wantCall, c.wantOK)
			}
			if c.goType == "*XarT" && len(keepAlives) != 1 {
				t.Errorf("handle wrapper: keepAlives = %v; want the wrapper kept alive", keepAlives)
			}
		})
	}
}

// TestSplitGoFuncType covers decomposition of a Go func type into its
// parameter types and return, including the depth-aware comma split that keeps
// nested types intact.
func TestSplitGoFuncType(t *testing.T) {
	cases := []struct {
		in         string
		wantParams []string
		wantRet    string
		wantOK     bool
	}{
		{"func()", nil, "", true},
		{"func(unsafe.Pointer)", []string{"unsafe.Pointer"}, "", true},
		{"func(unsafe.Pointer, unsafe.Pointer)", []string{"unsafe.Pointer", "unsafe.Pointer"}, "", true},
		{"func(uint64, unsafe.Pointer) bool", []string{"uint64", "unsafe.Pointer"}, "bool", true},
		{"func(a map[string]int, b int)", []string{"a map[string]int", "b int"}, "", true},
		{"func(f func(int, int) bool) error", []string{"f func(int, int) bool"}, "error", true},
		{"unsafe.Pointer", nil, "", false},
	}
	for _, c := range cases {
		params, ret, ok := splitGoFuncType(c.in)
		if ok != c.wantOK || ret != c.wantRet || !equalStrs(params, c.wantParams) {
			t.Errorf("splitGoFuncType(%q) = (%v, %q, %v); want (%v, %q, %v)",
				c.in, params, ret, ok, c.wantParams, c.wantRet, c.wantOK)
		}
	}
}

func equalStrs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestExternPuregoInitExpr covers the translation of CGo extern init shapes to
// Dlsym-based ones.
func TestExternPuregoInitExpr(t *testing.T) {
	cases := []struct {
		goType, cgoExpr, want string
	}{
		{"uint32", "*(*uint32)(C.machinit_extern_mach_task_self_())", "*(*uint32)(unsafe.Pointer(_addr))"},
		{"unsafe.Pointer", "*(*unsafe.Pointer)(C.xpc_extern_foo())", "*(*unsafe.Pointer)(unsafe.Pointer(_addr))"},
		{"unsafe.Pointer", "C.dispatch_extern_bar()", "unsafe.Pointer(_addr)"}, // struct-value global: address itself
		{"string", "", ""}, // unsupported shape stays zero-valued
	}
	for _, c := range cases {
		if got := externPuregoInitExpr(c.goType, c.cgoExpr); got != c.want {
			t.Errorf("externPuregoInitExpr(%q, %q) = %q; want %q", c.goType, c.cgoExpr, got, c.want)
		}
	}
}
