package typemap

import (
	"reflect"
	"testing"
)

func TestResolveFuncPtrSignature(t *testing.T) {
	m := &Mapper{
		TypedefIndex: map[string]string{
			"cc_ccache_release_f": "cc_int32 (*)(cc_ccache_t)",
		},
	}
	cases := []struct {
		in      string
		wantOK  bool
		wantRet string
		wantPar []string
	}{
		{"cc_int32 (*)(cc_ccache_t)", true, "cc_int32", []string{"cc_ccache_t"}},
		{"cc_int32 (*)(cc_ccache_t, cc_uint32 *)", true, "cc_int32", []string{"cc_ccache_t", "cc_uint32 *"}},
		{"void (*)(void)", true, "void", nil},
		{"cc_ccache_release_f", true, "cc_int32", []string{"cc_ccache_t"}}, // via typedef
		{"int", false, "", nil}, // not a function pointer
		{"NSString *", false, "", nil},
	}
	for _, c := range cases {
		sig, ok := m.ResolveFuncPtrSignature(c.in)
		if ok != c.wantOK {
			t.Errorf("ResolveFuncPtrSignature(%q) ok = %v, want %v", c.in, ok, c.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if sig.ReturnObjCType != c.wantRet {
			t.Errorf("ResolveFuncPtrSignature(%q) ret = %q, want %q", c.in, sig.ReturnObjCType, c.wantRet)
		}
		if !reflect.DeepEqual(sig.ParamObjCTypes, c.wantPar) {
			t.Errorf("ResolveFuncPtrSignature(%q) params = %v, want %v", c.in, sig.ParamObjCTypes, c.wantPar)
		}
	}
}
