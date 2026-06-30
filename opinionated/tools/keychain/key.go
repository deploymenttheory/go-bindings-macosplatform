//go:build darwin

package keychain

import (
	"fmt"

	foundation "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/foundation"
	security "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/security"
	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/obj"
)

// ── Key (kSecClassKey) ───────────────────────────────────────────────────────
//
// A cryptographic key is stored by reference. CreateKey builds a SecKeyRef from
// raw key material with SecKeyCreateWithData (given the key's type and class)
// and adds it. The key material must be in the format SecKeyCreateWithData
// expects — for an elliptic-curve private key that is the ANSI X9.63 encoding
// 0x04 || X || Y || K. (A key already wrapped in a PKCS#12 is instead added,
// together with its certificate, via an identity import.) There is no Update:
// key material is immutable.

// KeyType is the algorithm of a [Key].
type KeyType int

const (
	KeyEC  KeyType = iota // elliptic curve (kSecAttrKeyTypeECSECPrimeRandom)
	KeyRSA                // RSA (kSecAttrKeyTypeRSA)
)

func (t KeyType) constant() obj.Object {
	if t == KeyRSA {
		return security.KSecAttrKeyTypeRSA()
	}
	return security.KSecAttrKeyTypeECSECPrimeRandom()
}

// Key is a cryptographic key stored in the keychain.
type Key struct {
	Label string  // human-readable label (kSecAttrLabel)
	Type  KeyType // the key's algorithm, for Create
	Data  []byte  // the private key material, for Create (see package note for the format)

	// ApplicationLabel is set by the system on read (for an EC/RSA key it is the
	// hash of the public key); it is ignored by Create.
	ApplicationLabel []byte // kSecAttrApplicationLabel
}

// CreateKey builds a private key from k.Data (interpreted per k.Type) and adds
// it to the keychain. It fails if k.Data is not valid key material, or if the
// key is already present.
func CreateKey(k Key) error {
	attrs := foundation.NewMutableDictionary().
		Set(security.KSecAttrKeyType(), k.Type.constant()).
		Set(security.KSecAttrKeyClass(), security.KSecAttrKeyClassPrivate())

	// SecKeyCreateWithData reports a malformed key through its CFErrorRef
	// out-parameter, which the idiomatic wrapper surfaces as a Go error.
	keyRef, err := security.SecKeyCreateWithData(newData(k.Data), attrs)
	if err != nil {
		return fmt.Errorf("keychain: invalid key data for the given key type: %w", err)
	}

	add := []attr{ref(security.KSecValueRef(), keyRef)}
	if k.Label != "" {
		add = append(add, str(security.KSecAttrLabel(), k.Label))
	}
	return create(security.KSecClassKey(), add)
}

// ReadKey returns the key with the given label; found is false (nil error) when
// none exists.
func ReadKey(label string) (k Key, found bool, err error) {
	d, found, err := readOne(security.KSecClassKey(), []attr{
		str(security.KSecAttrLabel(), label),
	}, false)
	if err != nil || !found {
		return Key{}, found, err
	}
	return decodeKey(d), true, nil
}

// DeleteKey removes the key with the given label.
func DeleteKey(label string) error {
	return remove(security.KSecClassKey(), []attr{
		str(security.KSecAttrLabel(), label),
	})
}

// ListKeys returns every key in the keychain.
func ListKeys() ([]Key, error) {
	ds, err := readAll(security.KSecClassKey(), nil, false)
	if err != nil {
		return nil, err
	}
	out := make([]Key, 0, len(ds))
	for _, d := range ds {
		out = append(out, decodeKey(d))
	}
	return out, nil
}

func decodeKey(d *foundation.Dictionary) Key {
	return Key{
		Label:            attrString(d, security.KSecAttrLabel()),
		ApplicationLabel: attrBytes(d, security.KSecAttrApplicationLabel()),
	}
}
