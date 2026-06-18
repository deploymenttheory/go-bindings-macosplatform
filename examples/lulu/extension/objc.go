//go:build darwin

package extension

import (
	"unsafe"

	rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
)

// classID returns an ObjC class object as an ID for class-method dispatch.
func classID(name string) rt.ID { return rt.ID(rt.GetClass(name)) }

// nsDataBytes copies the bytes of an NSData into a Go slice.
func nsDataBytes(d rt.ID) []byte {
	if d == 0 {
		return nil
	}
	n := rt.Send[uint64](d, rt.RegisterName("length"))
	if n == 0 {
		return nil
	}
	p := rt.Send[unsafe.Pointer](d, rt.RegisterName("bytes"))
	if p == nil {
		return nil
	}
	return append([]byte(nil), unsafe.Slice((*byte)(p), int(n))...)
}

// bytesToNSData builds an NSData (copying b) for an XPC reply.
func bytesToNSData(b []byte) rt.ID {
	cls := classID("NSData")
	if len(b) == 0 {
		return rt.Send[rt.ID](cls, rt.RegisterName("data"))
	}
	return rt.Send[rt.ID](cls, rt.RegisterName("dataWithBytes:length:"),
		unsafe.Pointer(&b[0]), uint64(len(b)))
}
