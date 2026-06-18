//go:build darwin

package purego

import "fmt"

// XPCArgKind classifies one argument of an XPC protocol method for both ObjC
// type-encoding (so NSXPCInterface can build the wire NSMethodSignature) and
// Go-side marshalling. XPC methods always return void; arguments are limited to
// the property-list object types plus a single trailing reply block.
type XPCArgKind int

const (
	// XPCObject is an ObjC object pointer (id / NSString* / NSData* /
	// NSDictionary* / NSNumber* / NSArray* …). Encodes as "@".
	XPCObject XPCArgKind = iota
	// XPCInt64 encodes as "q".
	XPCInt64
	// XPCUInt64 encodes as "Q".
	XPCUInt64
	// XPCBool encodes as "B".
	XPCBool
	// XPCDouble encodes as "d".
	XPCDouble
	// XPCReplyBlock is a reply handler block (void(^)(...)). Encodes as "@?".
	// At most one reply block may appear, and it must be the last argument.
	XPCReplyBlock
)

// encoding returns the ObjC @encode string fragment for one argument kind.
func (k XPCArgKind) encoding() string {
	switch k {
	case XPCObject:
		return "@"
	case XPCInt64:
		return "q"
	case XPCUInt64:
		return "Q"
	case XPCBool:
		return "B"
	case XPCDouble:
		return "d"
	case XPCReplyBlock:
		return "@?"
	default:
		return "@"
	}
}

// XPCMethod describes one method of an XPC service protocol.
type XPCMethod struct {
	// Selector is the full ObjC selector, e.g. "processRequest:withReply:".
	Selector string
	// Args lists the argument kinds in selector order, excluding the implicit
	// self and _cmd. A reply handler, if present, is the last entry and must be
	// XPCReplyBlock.
	Args []XPCArgKind
	// ReplyArgs describes the parameters of the reply block (in order), when
	// Args ends in XPCReplyBlock. Used by callers to build/marshal the block.
	ReplyArgs []XPCArgKind
}

// encodeXPCMethod returns the ObjC method type encoding for an XPC method.
// XPC methods always return void; the encoding is "v@:" (void return, self,
// _cmd) followed by one fragment per argument. Frame-offset numbers are omitted
// — protocol_addMethodDescription accepts the numberless form.
func encodeXPCMethod(m XPCMethod) string {
	enc := "v@:"
	for _, a := range m.Args {
		enc += a.encoding()
	}
	return enc
}

// XPCProtocol describes a Go-defined XPC service @protocol. It is turned into a
// real ObjC Protocol at runtime via BuildProtocol so it can be handed to
// NSXPCInterface (which requires a registered Protocol*).
type XPCProtocol struct {
	// Name is the ObjC protocol name, e.g. "XPCDaemonProtocol". It must be
	// unique within the process.
	Name    string
	Methods []XPCMethod
}

// BuildProtocol allocates and registers an ObjC Protocol matching p, returning
// its pointer for use with NSXPCInterface +interfaceWithProtocol:. It is
// idempotent: if a protocol with p.Name is already registered (e.g. on a second
// call), the existing one is returned. The returned pointer stays valid for the
// life of the process.
func (p XPCProtocol) BuildProtocol() (*Protocol, error) {
	if existing := GetProtocol(p.Name); existing != nil {
		return existing, nil
	}
	proto := AllocateProtocol(p.Name)
	if proto == nil {
		// AllocateProtocol returns nil if a protocol with this name already
		// exists but was not yet visible to GetProtocol, or on allocation
		// failure. Re-check once.
		if existing := GetProtocol(p.Name); existing != nil {
			return existing, nil
		}
		return nil, fmt.Errorf("xpc: AllocateProtocol(%q) failed", p.Name)
	}
	for _, method := range p.Methods {
		// All XPC methods are required instance methods.
		proto.AddMethodDescription(RegisterName(method.Selector), encodeXPCMethod(method), true, true)
	}
	proto.Register()
	return proto, nil
}
