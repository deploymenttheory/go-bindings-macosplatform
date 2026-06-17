//go:build darwin

package purego

import "github.com/ebitengine/purego/objc"

// DelegateHandler binds one ObjC selector to a Go implementation. Fn must have
// the IMP shape: its first parameter is the receiver (objc.ID self), its second
// the selector (objc.SEL _cmd), followed by the method's own arguments. For
// example, -[id<VZVirtualMachineDelegate> guestDidStopVirtualMachine:] is:
//
//	DelegateHandler{Selector: "guestDidStopVirtualMachine:",
//	    Fn: func(self objc.ID, _cmd objc.SEL, vm objc.ID) { ... }}
type DelegateHandler struct {
	Selector string
	Fn       any
}

// NewDelegate registers an ObjC class named className (subclassing super and
// conforming to protocols) whose methods are the supplied Go funcs, then returns
// a freshly alloc/init'd instance. If a class with className already exists it is
// reused (re-registering would fail), so pick a stable, unique name per delegate
// shape. This replaces the hand-rolled RegisterClass + alloc/init dance consumers
// previously needed to implement delegate protocols from Go.
func NewDelegate(
	className string,
	super Class,
	protocols []*Protocol,
	handlers ...DelegateHandler,
) (ID, error) {
	cls := objc.GetClass(className)
	if cls == 0 {
		methods := make([]objc.MethodDef, 0, len(handlers))
		for _, h := range handlers {
			methods = append(methods, objc.MethodDef{
				Cmd: objc.RegisterName(h.Selector),
				Fn:  h.Fn,
			})
		}
		registered, err := objc.RegisterClass(className, super, protocols, nil, methods)
		if err != nil {
			return 0, err
		}
		cls = registered
	}
	obj := objc.ID(cls).Send(objc.RegisterName("alloc"))
	return obj.Send(objc.RegisterName("init")), nil
}
