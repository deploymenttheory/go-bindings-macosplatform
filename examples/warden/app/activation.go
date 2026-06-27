//go:build darwin

package app

import (
	"errors"

	rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/warden/shared"

	// Side-effect import: loads SystemExtensions.framework.
	_ "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/systemextensions"
)

// ActivateExtension submits an OSSystemExtensionRequest to install/activate the
// network system extension with the given bundle identifier, mirroring Warden's
// extension activation. The system presents its own approval UI; a full app
// would also set a request delegate to observe completion.
//
// This only succeeds from a properly signed app bundle carrying the
// System Extension and Network Extension entitlements (see README). It returns an
// error if the SystemExtensions runtime is unavailable (e.g. the framework failed
// to load); the asynchronous approval result is reported by the OS, not here.
func ActivateExtension(bundleID string) error {
	return submitRequest("activationRequestForExtension:queue:", bundleID)
}

// DeactivateExtension submits a deactivation request for bundleID.
func DeactivateExtension(bundleID string) error {
	return submitRequest("deactivationRequestForExtension:queue:", bundleID)
}

// submitRequest builds an OSSystemExtensionRequest via the given factory selector
// and hands it to the shared OSSystemExtensionManager.
func submitRequest(requestSelector, bundleID string) error {
	mgr := rt.Send[rt.ID](shared.ClassID("OSSystemExtensionManager"), rt.RegisterName("sharedManager"))
	if mgr == 0 {
		return errors.New("warden: OSSystemExtensionManager unavailable (SystemExtensions.framework not loaded)")
	}
	req := rt.Send[rt.ID](shared.ClassID("OSSystemExtensionRequest"),
		rt.RegisterName(requestSelector), rt.NSString(bundleID), mainQueue())
	rt.Send[rt.ID](mgr, rt.RegisterName("submitRequest:"), req)
	return nil
}
