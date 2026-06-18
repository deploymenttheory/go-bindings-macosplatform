//go:build darwin

package extension

import (
	"log"
	"time"

	rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/warden/config"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/warden/rules"
)

// reconcileInterval is how often a config-governed daemon re-applies its config
// to correct any drift (e.g. direct tampering of the persisted rules file).
const reconcileInterval = 60 * time.Second

// Run is the network-extension entry point. It loads the rule set, optionally
// reconciles it against a declarative config document, registers the filter
// provider class and the XPC daemon, enters NEProvider system-extension mode,
// and runs the main run loop forever. This is the Go equivalent of Warden's
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
	// Declarative bootstrap: when governed by a config the daemon is "managed" —
	// the document is the authoritative, complete firewall state. Reconcile to it
	// now and keep reconciling on an interval so out-of-band drift is reverted.
	managed := configPath != ""
	if managed {
		if err := reconcile(eng, configPath); err != nil {
			return err
		}
		go func() {
			t := time.NewTicker(reconcileInterval)
			defer t.Stop()
			for range t.C {
				if err := reconcile(eng, configPath); err != nil {
					log.Printf("reconcile: %v", err)
				}
			}
		}()
	}
	if err := RegisterProvider(eng); err != nil {
		return err
	}
	if err := StartDaemon(eng, managed); err != nil {
		return err
	}

	// Enter NE system-extension mode and run the run loop (callbacks/flows are
	// delivered on it).
	rt.Send[rt.ID](classID("NEProvider"), rt.RegisterName("startSystemExtensionMode"))
	runLoop := rt.Send[rt.ID](classID("NSRunLoop"), rt.RegisterName("currentRunLoop"))
	rt.Send[rt.ID](runLoop, rt.RegisterName("run"))
	return nil
}

// reconcile authoritatively applies the config document to the engine and
// persists the result, logging any change. Used at startup and on the managed-
// mode reconcile ticker.
func reconcile(eng *rules.Engine, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	added, deleted := config.Apply(cfg, eng)
	if added+deleted > 0 {
		if err := eng.Save(); err != nil {
			return err
		}
		log.Printf("reconcile %s: +%d -%d rules", configPath, added, deleted)
	}
	return nil
}
