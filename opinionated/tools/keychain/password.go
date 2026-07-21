//go:build darwin

package keychain

import (
	foundation "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
	security "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/security"
)

// ── Generic password (kSecClassGenericPassword) ──────────────────────────────

// GenericPassword is an application secret identified by a service and account.
type GenericPassword struct {
	Service string // the application/service name (kSecAttrService)
	Account string // the account/username (kSecAttrAccount)
	Label   string // human-readable label shown in Keychain Access (kSecAttrLabel)
	Secret  []byte // the secret bytes (kSecValueData)
}

func (p GenericPassword) identity() []attr {
	return []attr{
		str(security.KSecAttrService(), p.Service),
		str(security.KSecAttrAccount(), p.Account),
	}
}

// CreateGenericPassword adds p; it fails if an item with the same service and
// account already exists.
func CreateGenericPassword(p GenericPassword) error {
	attrs := p.identity()
	if p.Label != "" {
		attrs = append(attrs, str(security.KSecAttrLabel(), p.Label))
	}
	attrs = append(attrs, blob(security.KSecValueData(), p.Secret))
	return create(security.KSecClassGenericPassword(), attrs)
}

// ReadGenericPassword returns the item for service and account; found is false
// (nil error) when none exists.
func ReadGenericPassword(service, account string) (p GenericPassword, found bool, err error) {
	d, found, err := readOne(security.KSecClassGenericPassword(), []attr{
		str(security.KSecAttrService(), service),
		str(security.KSecAttrAccount(), account),
	}, true)
	if err != nil || !found {
		return GenericPassword{}, found, err
	}
	return decodeGenericPassword(d), true, nil
}

// UpdateGenericPassword changes the secret (and label) of the item identified by
// p.Service and p.Account.
func UpdateGenericPassword(p GenericPassword) error {
	changes := []attr{blob(security.KSecValueData(), p.Secret)}
	if p.Label != "" {
		changes = append(changes, str(security.KSecAttrLabel(), p.Label))
	}
	return update(security.KSecClassGenericPassword(), p.identity(), changes)
}

// DeleteGenericPassword removes the item for service and account.
func DeleteGenericPassword(service, account string) error {
	return remove(security.KSecClassGenericPassword(), []attr{
		str(security.KSecAttrService(), service),
		str(security.KSecAttrAccount(), account),
	})
}

// ListGenericPasswords returns the metadata of every generic-password item. It
// does not request kSecValueData (so Secret is nil) — both to avoid prompting for
// access to secrets owned by other applications and because enumeration should
// not bulk-export secrets. Use ReadGenericPassword for an item's secret.
func ListGenericPasswords() ([]GenericPassword, error) {
	ds, err := readAll(security.KSecClassGenericPassword(), nil, false)
	if err != nil {
		return nil, err
	}
	out := make([]GenericPassword, 0, len(ds))
	for _, d := range ds {
		out = append(out, decodeGenericPassword(d))
	}
	return out, nil
}

func decodeGenericPassword(d *foundation.Dictionary) GenericPassword {
	return GenericPassword{
		Service: attrString(d, security.KSecAttrService()),
		Account: attrString(d, security.KSecAttrAccount()),
		Label:   attrString(d, security.KSecAttrLabel()),
		Secret:  attrBytes(d, security.KSecValueData()),
	}
}

// ── Internet password (kSecClassInternetPassword) ────────────────────────────

// InternetPassword is a credential for a network server. The protocol is fixed
// to HTTPS here (kSecAttrProtocolHTTPS); a fuller API would expose the protocol
// and port as well.
type InternetPassword struct {
	Server  string // host the credential is for (kSecAttrServer)
	Account string // the username (kSecAttrAccount)
	Label   string // human-readable label (kSecAttrLabel)
	Secret  []byte // the secret bytes (kSecValueData)
}

func (p InternetPassword) identity() []attr {
	return []attr{
		str(security.KSecAttrServer(), p.Server),
		ref(security.KSecAttrProtocol(), security.KSecAttrProtocolHTTPS()),
		str(security.KSecAttrAccount(), p.Account),
	}
}

// CreateInternetPassword adds p; it fails if an item with the same server,
// protocol, and account already exists.
func CreateInternetPassword(p InternetPassword) error {
	attrs := p.identity()
	if p.Label != "" {
		attrs = append(attrs, str(security.KSecAttrLabel(), p.Label))
	}
	attrs = append(attrs, blob(security.KSecValueData(), p.Secret))
	return create(security.KSecClassInternetPassword(), attrs)
}

// ReadInternetPassword returns the item for server and account; found is false
// (nil error) when none exists.
func ReadInternetPassword(server, account string) (p InternetPassword, found bool, err error) {
	d, found, err := readOne(security.KSecClassInternetPassword(), []attr{
		str(security.KSecAttrServer(), server),
		ref(security.KSecAttrProtocol(), security.KSecAttrProtocolHTTPS()),
		str(security.KSecAttrAccount(), account),
	}, true)
	if err != nil || !found {
		return InternetPassword{}, found, err
	}
	return decodeInternetPassword(d), true, nil
}

// UpdateInternetPassword changes the secret (and label) of the item identified
// by p.Server and p.Account.
func UpdateInternetPassword(p InternetPassword) error {
	changes := []attr{blob(security.KSecValueData(), p.Secret)}
	if p.Label != "" {
		changes = append(changes, str(security.KSecAttrLabel(), p.Label))
	}
	return update(security.KSecClassInternetPassword(), p.identity(), changes)
}

// DeleteInternetPassword removes the item for server and account.
func DeleteInternetPassword(server, account string) error {
	return remove(security.KSecClassInternetPassword(), []attr{
		str(security.KSecAttrServer(), server),
		ref(security.KSecAttrProtocol(), security.KSecAttrProtocolHTTPS()),
		str(security.KSecAttrAccount(), account),
	})
}

// ListInternetPasswords returns the metadata of every internet-password item
// (Secret is nil; see [ListGenericPasswords] for why). Use ReadInternetPassword
// for an item's secret.
func ListInternetPasswords() ([]InternetPassword, error) {
	ds, err := readAll(security.KSecClassInternetPassword(), nil, false)
	if err != nil {
		return nil, err
	}
	out := make([]InternetPassword, 0, len(ds))
	for _, d := range ds {
		out = append(out, decodeInternetPassword(d))
	}
	return out, nil
}

func decodeInternetPassword(d *foundation.Dictionary) InternetPassword {
	return InternetPassword{
		Server:  attrString(d, security.KSecAttrServer()),
		Account: attrString(d, security.KSecAttrAccount()),
		Label:   attrString(d, security.KSecAttrLabel()),
		Secret:  attrBytes(d, security.KSecValueData()),
	}
}
