//go:build darwin

// Package app implements LuLu's controlling-app side: activating the network
// system extension and talking to its XPC daemon. Mirrors LuLu's App/.
package app

import (
	"unsafe"

	rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
)

func classID(name string) rt.ID { return rt.ID(rt.GetClass(name)) }

// nsDataBytes copies an NSData's bytes into a Go slice.
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

// bytesToNSData builds an NSData (copying b).
func bytesToNSData(b []byte) rt.ID {
	cls := classID("NSData")
	if len(b) == 0 {
		return rt.Send[rt.ID](cls, rt.RegisterName("data"))
	}
	return rt.Send[rt.ID](cls, rt.RegisterName("dataWithBytes:length:"),
		unsafe.Pointer(&b[0]), uint64(len(b)))
}

// mainQueue returns the GCD main queue object (the address of _dispatch_main_q,
// which is what dispatch_get_main_queue() returns) for APIs that take a
// dispatch_queue_t, resolved without pulling in the CGo dispatch package.
var cachedMainQueue rt.ID

func mainQueue() rt.ID {
	if cachedMainQueue != 0 {
		return cachedMainQueue
	}
	h, err := rt.Dlopen("/usr/lib/libSystem.B.dylib", rt.RTLD_GLOBAL|rt.RTLD_LAZY)
	if err != nil {
		return 0
	}
	addr, err := rt.Dlsym(h, "_dispatch_main_q")
	if err != nil {
		return 0
	}
	cachedMainQueue = rt.ID(addr)
	return cachedMainQueue
}
