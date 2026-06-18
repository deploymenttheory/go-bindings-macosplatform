//go:build darwin

package extension

import (
	"log"

	rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/lulu/config"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/lulu/rules"
)

// Run is the network-extension entry point. It loads the rule set, optionally
// reconciles it against a declarative config document, registers the filter
// provider class and the XPC daemon, enters NEProvider system-extension mode,
// and runs the main run loop forever. This is the Go equivalent of LuLu's
// Extension/main.m calling [NEProvider startSystemExtensionMode].
//
// NOTE: macOS only loads this as a system extension when it is the principal
// class of a signed .systemextension bundle embedded in a signed app with the
// com.apple.developer.networking.networkextension entitlement. See the README.
func Run(rulesPath, configPath string) error {
	eng := rules.New(rulesPath)
	if err := eng.Load(); err != nil {
		return err
	}
	// Declarative bootstrap: reconcile the rule set to a config document.
	if configPath != "" {
		cfg, err := config.Load(configPath)
		if err != nil {
			return err
		}
		added, deleted := config.Apply(cfg, eng)
		if err := eng.Save(); err != nil {
			return err
		}
		log.Printf("applied config %s: +%d -%d rules", configPath, added, deleted)
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
