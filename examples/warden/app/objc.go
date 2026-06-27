//go:build darwin

// Package app implements Warden's controlling-app side: activating the network
// system extension and talking to its XPC daemon. Mirrors Warden's App/.
package app

import (
	rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
)

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
