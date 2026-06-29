//go:build darwin

package shared

import (
	"unsafe"

	rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
)

// ADOPTION: these helpers use the *runtime* (rt), not the idiomatic Foundation
// wrappers, on purpose. They marshal NSData across the NSXPC boundary — and XPC
// has no idiomatic form, so its payloads have to be handled as raw object ids
// (rt.ID) anyway. Converting to/from an idiomatic *foundation.Data here would only
// add a wrap-then-unwrap round trip. Rule of thumb: use the idiomatic layer for
// real framework-class calls; stay on the runtime for the plumbing (XPC, dispatch,
// subclassing) the idiomatic layer doesn't model. See examples/README.md.

// ClassID returns an ObjC class object as an ID for class-method dispatch
// (e.g. ClassID("NSData") to send +dataWithBytes:length:). Both the app and the
// extension marshal NSData over XPC, so these helpers live here rather than being
// copied into each side.
func ClassID(name string) rt.ID { return rt.ID(rt.GetClass(name)) }

// NSDataBytes copies an NSData's bytes into a freshly allocated Go slice, so the
// result stays valid after the NSData is released. A nil or empty NSData yields nil.
func NSDataBytes(d rt.ID) []byte {
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

// BytesToNSData builds an NSData that copies b. An empty b yields an empty NSData
// (+[NSData data]) rather than nil, which XPC reply blocks expect.
func BytesToNSData(b []byte) rt.ID {
	cls := ClassID("NSData")
	if len(b) == 0 {
		return rt.Send[rt.ID](cls, rt.RegisterName("data"))
	}
	return rt.Send[rt.ID](cls, rt.RegisterName("dataWithBytes:length:"),
		unsafe.Pointer(&b[0]), uint64(len(b)))
}
