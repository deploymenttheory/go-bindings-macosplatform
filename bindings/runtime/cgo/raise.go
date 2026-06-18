//go:build darwin

package cgo

import "unsafe"

// OnException is called by RaiseIfException immediately before panicking with
// an ObjC exception. Wire this to your structured logger at application startup.
// Called from any goroutine; must be goroutine-safe.
var OnException func(reason string)

// OnCallbackPanic is called when a panic is recovered inside a generated
// //export callback (goCallBlock_* or goCallIMP_*). Wire this to your
// structured logger at application startup. Called from any goroutine.
var OnCallbackPanic func(name string, r any, stack []byte)

// RaiseIfException converts a caught ObjC NSException pointer to a Go panic.
// It is a no-op when exc is nil. The exception object is released by
// ExceptionInfoFromPtr. OnException (if set) is invoked with the reason first,
// giving the application a chance to log through its structured logger before
// the panic unwinds the stack.
func RaiseIfException(exc unsafe.Pointer) {
	if exc == nil {
		return
	}
	info := ExceptionInfoFromPtr(exc)
	if h := OnException; h != nil {
		h(info.Reason)
	}
	panic("ObjC exception [" + info.Name + "]: " + info.Reason)
}
