package idiolib

import "testing"

func TestSnakeToPascal(t *testing.T) {
	cases := map[string]string{
		"send_message": "SendMessage",
		"message_size": "MessageSize",
		"create":       "Create",
		"__os_log":     "OsLog",
		"":             "",
		"xpc_object_t": "XpcObjectT",
	}
	for in, want := range cases {
		if got := snakeToPascal(in); got != want {
			t.Errorf("snakeToPascal(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHandleGoName(t *testing.T) {
	cases := []struct {
		base, prefix, want string
	}{
		{"xpc_connection_t", "xpc", "Connection"},
		{"xpc_object_t", "xpc", "Object"},
		{"dispatch_queue_t", "dispatch", "Queue"},
		{"es_client_t", "es", "Client"},
		{"audit_token_t", "proc", "AuditToken"}, // prefix doesn't match → full name
	}
	for _, c := range cases {
		if got := handleGoName(c.base, c.prefix); got != c.want {
			t.Errorf("handleGoName(%q, %q) = %q, want %q", c.base, c.prefix, got, c.want)
		}
	}
}

func TestMethodGoName(t *testing.T) {
	if got := methodGoName("xpc_connection_send_message", "xpc_connection_t", "xpc"); got != "SendMessage" {
		t.Errorf("methodGoName = %q, want SendMessage", got)
	}
	// Falls back to library-prefix strip when the symbol lacks the handle prefix.
	if got := methodGoName("xpc_connection_create", "xpc_object_t", "xpc"); got != "ConnectionCreate" {
		t.Errorf("methodGoName fallback = %q, want ConnectionCreate", got)
	}
}

func TestQualifyRawTokens(t *testing.T) {
	cases := map[string]string{
		"int32":          "int32",
		"uint64":         "uint64",
		"unsafe.Pointer": "unsafe.Pointer",
		"Es_message_t":   "raw.Es_message_t",
		"*Es_message_t":  "*raw.Es_message_t",
		"[]RusageInfo":   "[]raw.RusageInfo",
		"bsd.Timespec":   "bsd.Timespec",
		"*bsd.Timespec":  "*bsd.Timespec",
		"string":         "string",
	}
	for in, want := range cases {
		if got := qualifyRawTokens(in); got != want {
			t.Errorf("qualifyRawTokens(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCleanTypeToken(t *testing.T) {
	cases := map[string]string{
		"DISPATCH_RETURNS_RETAINED dispatch_data_t  _Nullable": "dispatch_data_t",
		"XPC_RETURNS_RETAINED xpc_object_t  _Nonnull":          "xpc_object_t",
		"void *":        "void",
		"const char *":  "char",
		"OSStatus":      "OSStatus",
		"kern_return_t": "kern_return_t",
	}
	for in, want := range cases {
		if got := cleanTypeToken(in); got != want {
			t.Errorf("cleanTypeToken(%q) = %q, want %q", in, got, want)
		}
	}
}
