//go:build darwin

package purego

import (
	"fmt"
	"sync/atomic"
	"unsafe"
)

// NSXPCConnectionPrivileged is the NSXPCConnectionOptions bit requesting a
// privileged (root) mach-service connection.
const NSXPCConnectionPrivileged uint64 = 1 << 12

// listenerDelegateCounter makes each listener delegate class name unique, so two
// HandleNewConnections calls do not collide on NewDelegate's class reuse (which
// would silently bind the second listener to the first's accept closure).
var listenerDelegateCounter atomic.Uint64

// keepAlive retains references to delegate objects whose ObjC owners hold them
// weakly (NSXPCListener.delegate is a weak property), preventing the only +1 we
// own from being released and the object deallocated for the process lifetime.
var keepAlive []ID

// ── Connection (client or per-connection server side) ──────────────────────────

// XPCConn wraps an NSXPCConnection.
type XPCConn struct{ id ID }

// WrapConn adopts an existing NSXPCConnection pointer (e.g. the one handed to a
// listener's accept handler).
func WrapConn(id ID) XPCConn { return XPCConn{id: id} }

// ID returns the underlying NSXPCConnection object pointer.
func (c XPCConn) ID() ID { return c.id }

// DialMachService opens a connection to a registered mach service. Set
// privileged for a service that must run as root.
func DialMachService(name string, privileged bool) XPCConn {
	var opts uint64
	if privileged {
		opts = NSXPCConnectionPrivileged
	}
	alloc := Send[ID](ID(GetClass("NSXPCConnection")), RegisterName("alloc"))
	id := Send[ID](alloc, RegisterName("initWithMachServiceName:options:"), NSString(name), opts)
	return XPCConn{id: id}
}

// DialService opens a connection to an XPC service bundle by name.
func DialService(name string) XPCConn {
	alloc := Send[ID](ID(GetClass("NSXPCConnection")), RegisterName("alloc"))
	id := Send[ID](alloc, RegisterName("initWithServiceName:"), NSString(name))
	return XPCConn{id: id}
}

// interfaceForProtocol builds an NSXPCInterface from an XPCProtocol.
func interfaceForProtocol(p XPCProtocol) (ID, error) {
	proto, err := p.BuildProtocol()
	if err != nil {
		return 0, err
	}
	iface := Send[ID](ID(GetClass("NSXPCInterface")), RegisterName("interfaceWithProtocol:"), unsafe.Pointer(proto))
	return iface, nil
}

// SetRemoteInterface declares the protocol the remote object implements so
// outgoing proxy calls can be marshalled.
func (c XPCConn) SetRemoteInterface(p XPCProtocol) error {
	iface, err := interfaceForProtocol(p)
	if err != nil {
		return err
	}
	Send[ID](c.id, RegisterName("setRemoteObjectInterface:"), iface)
	return nil
}

// SetExported registers obj (from NewExportedObject) as the local object that
// handles incoming calls, declaring protocol p as the exported interface.
func (c XPCConn) SetExported(p XPCProtocol, obj ID) error {
	iface, err := interfaceForProtocol(p)
	if err != nil {
		return err
	}
	Send[ID](c.id, RegisterName("setExportedInterface:"), iface)
	Send[ID](c.id, RegisterName("setExportedObject:"), obj)
	return nil
}

// SetInvalidationHandler installs a handler called when the connection is invalidated.
func (c XPCConn) SetInvalidationHandler(fn func()) {
	blk := NewBlock(func(_ Block) { fn() })
	Send[ID](c.id, RegisterName("setInvalidationHandler:"), blk)
}

// SetInterruptionHandler installs a handler called when the connection is interrupted.
func (c XPCConn) SetInterruptionHandler(fn func()) {
	blk := NewBlock(func(_ Block) { fn() })
	Send[ID](c.id, RegisterName("setInterruptionHandler:"), blk)
}

// Resume activates the connection. A connection delivers no messages until resumed.
func (c XPCConn) Resume() { Send[ID](c.id, RegisterName("resume")) }

// Invalidate tears the connection down; no further messages are sent or received.
func (c XPCConn) Invalidate() { Send[ID](c.id, RegisterName("invalidate")) }

// Suspend balances a later Resume; calls must be balanced.
func (c XPCConn) Suspend() { Send[ID](c.id, RegisterName("suspend")) }

// ── Remote proxy (outgoing calls) ───────────────────────────────────────────────

// XPCProxy wraps a remote-object proxy returned by RemoteProxy.
type XPCProxy struct{ id ID }

// RemoteProxy returns the remote object proxy for the connection. onError, if
// non-nil, is invoked with a Go error when a message cannot be delivered.
func (c XPCConn) RemoteProxy(onError func(error)) XPCProxy {
	errBlk := NewBlock(func(_ Block, nserr ID) {
		if onError != nil {
			onError(NSErrorToError(nserr))
		}
	})
	proxy := Send[ID](c.id, RegisterName("remoteObjectProxyWithErrorHandler:"), errBlk)
	return XPCProxy{id: proxy}
}

// ID returns the underlying proxy object pointer.
func (p XPCProxy) ID() ID { return p.id }

// CallOneWay sends a fire-and-forget XPC method (void return, no reply block).
// args are passed straight through to objc message dispatch (objc.ID for
// objects, NSString(...) for strings, primitives for scalars).
func (p XPCProxy) CallOneWay(selector string, args ...any) {
	Send[ID](p.id, RegisterName(selector), args...)
}

// CallWithReply sends an XPC method whose final argument is a reply block. reply
// must be an IMP-shaped Go func whose first parameter is the block itself,
// followed by the reply block's own parameters, e.g.
//
//	func(_ purego.Block, result objc.ID) { ... }
//
// args are the leading non-block arguments. The call returns immediately; reply
// fires asynchronously when the peer responds. The reply closure is retained in
// the block cache for the round trip.
func (p XPCProxy) CallWithReply(selector string, reply any, args ...any) {
	blk := NewBlock(reply)
	all := append(append([]any{}, args...), blk)
	Send[ID](p.id, RegisterName(selector), all...)
}

// ── Exported object (incoming calls) ────────────────────────────────────────────

// XPCExport binds Go handlers to the selectors of an XPC service protocol.
type XPCExport struct {
	Protocol XPCProtocol
	// Handlers maps each selector to an IMP-shaped Go func:
	//
	//	func(self objc.ID, _cmd objc.SEL, <args...>)
	//
	// Object arguments arrive as objc.ID; a reply-block argument arrives as
	// objc.Block (invoke it with purego.InvokeBlock to send the reply).
	Handlers map[string]any
}

// NewExportedObject registers an ObjC class named className that conforms to
// x.Protocol and whose methods dispatch to x.Handlers, then returns a fresh
// instance to hand to XPCConn.SetExported. className must be process-unique.
func NewExportedObject(className string, x XPCExport) (ID, error) {
	proto, err := x.Protocol.BuildProtocol()
	if err != nil {
		return 0, err
	}
	handlers := make([]DelegateHandler, 0, len(x.Handlers))
	for selector, fn := range x.Handlers {
		handlers = append(handlers, DelegateHandler{Selector: selector, Fn: fn})
	}
	return NewDelegate(className, GetClass("NSObject"), []*Protocol{proto}, handlers...)
}

// ── Listener (daemon side) ──────────────────────────────────────────────────────

// XPCListener wraps an NSXPCListener.
type XPCListener struct{ id ID }

// NewMachServiceListener creates a listener that vends a registered mach service.
func NewMachServiceListener(name string) XPCListener {
	alloc := Send[ID](ID(GetClass("NSXPCListener")), RegisterName("alloc"))
	id := Send[ID](alloc, RegisterName("initWithMachServiceName:"), NSString(name))
	return XPCListener{id: id}
}

// ID returns the underlying NSXPCListener object pointer.
func (l XPCListener) ID() ID { return l.id }

// HandleNewConnections installs a delegate whose accept handler runs for each
// inbound connection. Configure the connection's exported interface/object
// inside accept and call Resume; return true to take the connection, false to
// reject it. The delegate object is retained for the process lifetime because
// NSXPCListener holds its delegate weakly.
func (l XPCListener) HandleNewConnections(accept func(XPCConn) bool) error {
	className := fmt.Sprintf("GoXPCListenerDelegate_%d", listenerDelegateCounter.Add(1))
	delegate, err := NewDelegate(
		className,
		GetClass("NSObject"),
		nil,
		DelegateHandler{
			Selector: "listener:shouldAcceptNewConnection:",
			Fn: func(_ ID, _ SEL, _ ID, conn ID) bool {
				return accept(WrapConn(conn))
			},
		},
	)
	if err != nil {
		return err
	}
	keepAlive = append(keepAlive, Retain(delegate))
	Send[ID](l.id, RegisterName("setDelegate:"), delegate)
	return nil
}

// Resume activates the listener.
func (l XPCListener) Resume() { Send[ID](l.id, RegisterName("resume")) }

// Invalidate tears the listener down.
func (l XPCListener) Invalidate() { Send[ID](l.id, RegisterName("invalidate")) }
