//go:build darwin

package app

import (
	rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"

	// Side-effect import: loads SystemExtensions.framework.
	_ "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/systemextensions"
)

// ActivateExtension submits an OSSystemExtensionRequest to install/activate the
// network system extension with the given bundle identifier, mirroring Warden's
// extension activation. The system presents its own approval UI; a full app
// would also set a request delegate to observe completion.
//
// This only succeeds from a properly signed app bundle carrying the
// System Extension and Network Extension entitlements (see README).
func ActivateExtension(bundleID string) {
	req := rt.Send[rt.ID](classID("OSSystemExtensionRequest"),
		rt.RegisterName("activationRequestForExtension:queue:"),
		rt.NSString(bundleID), mainQueue())
	mgr := rt.Send[rt.ID](classID("OSSystemExtensionManager"), rt.RegisterName("sharedManager"))
	rt.Send[rt.ID](mgr, rt.RegisterName("submitRequest:"), req)
}

// DeactivateExtension submits a deactivation request for bundleID.
func DeactivateExtension(bundleID string) {
	req := rt.Send[rt.ID](classID("OSSystemExtensionRequest"),
		rt.RegisterName("deactivationRequestForExtension:queue:"),
		rt.NSString(bundleID), mainQueue())
	mgr := rt.Send[rt.ID](classID("OSSystemExtensionManager"), rt.RegisterName("sharedManager"))
	rt.Send[rt.ID](mgr, rt.RegisterName("submitRequest:"), req)
}
