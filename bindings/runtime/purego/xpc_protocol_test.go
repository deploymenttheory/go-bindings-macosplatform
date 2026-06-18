//go:build darwin

package purego

import "testing"

func TestEncodeXPCMethod(t *testing.T) {
	cases := []struct {
		name string
		m    XPCMethod
		want string
	}{
		{
			name: "no args",
			m:    XPCMethod{Selector: "ping"},
			want: "v@:",
		},
		{
			name: "single object",
			m:    XPCMethod{Selector: "log:", Args: []XPCArgKind{XPCObject}},
			want: "v@:@",
		},
		{
			name: "object + reply block",
			m: XPCMethod{
				Selector:  "processRequest:withReply:",
				Args:      []XPCArgKind{XPCObject, XPCReplyBlock},
				ReplyArgs: []XPCArgKind{XPCObject},
			},
			want: "v@:@@?",
		},
		{
			name: "mixed scalars",
			m:    XPCMethod{Selector: "setLimit:enabled:rate:", Args: []XPCArgKind{XPCUInt64, XPCBool, XPCDouble}},
			want: "v@:QBd",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := encodeXPCMethod(tc.m); got != tc.want {
				t.Errorf("encodeXPCMethod(%s) = %q, want %q", tc.m.Selector, got, tc.want)
			}
		})
	}
}

func TestBuildProtocolIdempotent(t *testing.T) {
	p := XPCProtocol{
		Name: "GoTestXPCProto_BuildProtocolIdempotent",
		Methods: []XPCMethod{
			{Selector: "doThing:", Args: []XPCArgKind{XPCObject}},
			{Selector: "fetch:withReply:", Args: []XPCArgKind{XPCObject, XPCReplyBlock}, ReplyArgs: []XPCArgKind{XPCObject}},
		},
	}
	proto, err := p.BuildProtocol()
	if err != nil {
		t.Fatalf("BuildProtocol: %v", err)
	}
	if proto == nil {
		t.Fatal("BuildProtocol returned nil protocol")
	}
	// Registered protocols are discoverable by name.
	if got := GetProtocol(p.Name); got == nil {
		t.Fatalf("GetProtocol(%q) returned nil after BuildProtocol", p.Name)
	}
	// Idempotent: a second build returns the already-registered protocol.
	proto2, err := p.BuildProtocol()
	if err != nil {
		t.Fatalf("second BuildProtocol: %v", err)
	}
	if !proto.Equals(proto2) {
		t.Errorf("second BuildProtocol returned a different protocol pointer")
	}
}
