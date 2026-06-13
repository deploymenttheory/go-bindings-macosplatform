package naming

import "testing"

func TestExportedFunctionName(t *testing.T) {
	cases := []struct {
		symbol string
		want   string
	}{
		// Already exported — must be byte-identical.
		{"CFArrayCreate", "CFArrayCreate"},
		{"SecItemAdd", "SecItemAdd"},
		{"CSSM_CL_CertSign", "CSSM_CL_CertSign"},
		{"CGColorCreateGenericGrayGamma2_2", "CGColorCreateGenericGrayGamma2_2"},
		{"AudioChannelLayoutTag_GetNumberOfChannels", "AudioChannelLayoutTag_GetNumberOfChannels"},

		// snake_case → PascalCase.
		{"vmnet_start_interface", "VmnetStartInterface"},
		{"vmnet_read", "VmnetRead"},
		{"sandbox_init", "SandboxInit"},
		{"os_log_create", "OsLogCreate"},

		// Lowercase camel with underscore segments.
		{"vImageBoxConvolve_ARGB8888", "VImageBoxConvolveARGB8888"},

		// Leading underscore.
		{"_MPIsFullyInitialized", "MPIsFullyInitialized"},
		{"_SparseCGIterate_Double", "SparseCGIterateDouble"},

		// Single lowercase word.
		{"proc_listpids", "ProcListpids"},

		// Degenerate inputs.
		{"", ""},
		{"_", ""},
		{"__", ""},
	}

	for _, c := range cases {
		if got := ExportedFunctionName(c.symbol); got != c.want {
			t.Errorf("ExportedFunctionName(%q) = %q, want %q", c.symbol, got, c.want)
		}
	}
}

func TestExportedTypeName(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"NSRange", "NSRange"},
		{"CGRect", "CGRect"},
		{"vmpktdesc", "Vmpktdesc"},
		{"vmnet_interface", "VmnetInterface"},
	}
	for _, c := range cases {
		if got := ExportedTypeName(c.name); got != c.want {
			t.Errorf("ExportedTypeName(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}
