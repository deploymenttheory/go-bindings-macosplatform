//go:build darwin

package app

import (
	"errors"

	sysext "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/systemextensions"
	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/obj"
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
	return submitRequest(true, bundleID)
}

// DeactivateExtension submits a deactivation request for bundleID.
func DeactivateExtension(bundleID string) error {
	return submitRequest(false, bundleID)
}

// submitRequest builds an OSSystemExtensionRequest (activation or deactivation)
// for bundleID and hands it to the shared OSSystemExtensionManager via the
// idiomatic SystemExtensions wrappers. The request delivers its delegate
// callbacks on the main dispatch queue.
func submitRequest(activate bool, bundleID string) error {
	// ADOPTION: SystemExtensions has typed idiomatic wrappers, so the manager and
	// request are built with them (sysext.SharedManager / ...RequestForExtensionQueue
	// / SubmitRequest) instead of hand-sending selectors. The one value the idiomatic
	// API can't supply is the dispatch queue — it's not a framework class — so we get
	// it from the runtime (mainQueue, below) and bridge it across with obj.Wrap, which
	// lifts a raw objc.ID into the obj.Object the idiomatic factory expects. obj.Wrap
	// and obj.ID are the public seam between the runtime and idiomatic layers.
	mgr := sysext.SharedManager()
	if mgr == nil {
		return errors.New("warden: OSSystemExtensionManager unavailable (SystemExtensions.framework not loaded)")
	}
	queue := obj.Wrap(mainQueue())
	req := sysext.ActivationRequestForExtensionQueue(bundleID, queue)
	if !activate {
		req = sysext.DeactivationRequestForExtensionQueue(bundleID, queue)
	}
	mgr.SubmitRequest(req)
	return nil
}
