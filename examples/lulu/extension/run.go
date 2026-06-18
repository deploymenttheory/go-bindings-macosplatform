//go:build darwin

package extension

import (
	rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/lulu/rules"
)

// Run is the network-extension entry point. It loads the rule set, registers the
// filter provider class and the XPC daemon, enters NEProvider system-extension
// mode, and runs the main run loop forever. This is the Go equivalent of LuLu's
// Extension/main.m calling [NEProvider startSystemExtensionMode].
//
// NOTE: macOS only loads this as a system extension when it is the principal
// class of a signed .systemextension bundle embedded in a signed app with the
// com.apple.developer.networking.networkextension entitlement. See the README.
func Run(rulesPath string) error {
	eng := rules.New(rulesPath)
	if err := eng.Load(); err != nil {
		return err
	}
	if err := RegisterProvider(eng); err != nil {
		return err
	}
	if err := StartDaemon(eng); err != nil {
		return err
	}

	// Enter NE system-extension mode and run the run loop (callbacks/flows are
	// delivered on it).
	rt.Send[rt.ID](classID("NEProvider"), rt.RegisterName("startSystemExtensionMode"))
	runLoop := rt.Send[rt.ID](classID("NSRunLoop"), rt.RegisterName("currentRunLoop"))
	rt.Send[rt.ID](runLoop, rt.RegisterName("run"))
	return nil
}
