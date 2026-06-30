//go:build darwin

package keychain

import (
	foundation "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/foundation"
	security "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/security"
)

// ── Identity (kSecClassIdentity) ─────────────────────────────────────────────
//
// An identity is the pairing of a certificate with its matching private key; it
// is not stored directly but formed by the keychain when both halves are present
// (typically after importing a PKCS#12). There is therefore no Create or Update
// here. Read/List/Delete identify identities by their certificate label.

// Identity is a certificate-plus-private-key pair stored in the keychain.
type Identity struct {
	Label string // the certificate's label (kSecAttrLabel)
}

// ReadIdentity returns the identity whose certificate has the given label; found
// is false (nil error) when none exists.
func ReadIdentity(label string) (id Identity, found bool, err error) {
	d, found, err := readOne(security.KSecClassIdentity(), []attr{
		str(security.KSecAttrLabel(), label),
	}, false)
	if err != nil || !found {
		return Identity{}, found, err
	}
	return decodeIdentity(d), true, nil
}

// DeleteIdentity removes the identity whose certificate has the given label.
func DeleteIdentity(label string) error {
	return remove(security.KSecClassIdentity(), []attr{
		str(security.KSecAttrLabel(), label),
	})
}

// ListIdentities returns every identity in the keychain.
func ListIdentities() ([]Identity, error) {
	ds, err := readAll(security.KSecClassIdentity(), nil, false)
	if err != nil {
		return nil, err
	}
	out := make([]Identity, 0, len(ds))
	for _, d := range ds {
		out = append(out, decodeIdentity(d))
	}
	return out, nil
}

func decodeIdentity(d *foundation.Dictionary) Identity {
	return Identity{Label: attrString(d, security.KSecAttrLabel())}
}
