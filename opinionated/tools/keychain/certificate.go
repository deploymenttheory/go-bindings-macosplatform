//go:build darwin

package keychain

import (
	"errors"

	foundation "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/foundation"
	security "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/security"
)

// ── Certificate (kSecClassCertificate) ───────────────────────────────────────
//
// A certificate's content is its DER encoding and is immutable, so there is no
// Update: change a certificate by deleting and re-creating it. On macOS a
// certificate is added by reference (kSecValueRef holding a SecCertificateRef),
// which CreateCertificate builds from the DER via SecCertificateCreateWithData.

// Certificate is an X.509 certificate stored in the keychain.
type Certificate struct {
	Label string // human-readable label (kSecAttrLabel)
	DER   []byte // the DER-encoded certificate (kSecValueData on read)
}

// CreateCertificate adds c. It fails if the certificate (by content) is already
// present, or if c.DER is not a valid DER-encoded certificate.
func CreateCertificate(c Certificate) error {
	certRef := security.SecCertificateCreateWithData(nil, newData(c.DER))
	if certRef == nil {
		return errors.New("keychain: invalid DER certificate")
	}
	attrs := []attr{ref(security.KSecValueRef(), certRef)}
	if c.Label != "" {
		attrs = append(attrs, str(security.KSecAttrLabel(), c.Label))
	}
	return create(security.KSecClassCertificate(), attrs)
}

// ReadCertificate returns the certificate with the given label; found is false
// (nil error) when none exists.
func ReadCertificate(label string) (c Certificate, found bool, err error) {
	d, found, err := readOne(security.KSecClassCertificate(), []attr{
		str(security.KSecAttrLabel(), label),
	}, true)
	if err != nil || !found {
		return Certificate{}, found, err
	}
	return decodeCertificate(d), true, nil
}

// DeleteCertificate removes the certificate with the given label.
func DeleteCertificate(label string) error {
	return remove(security.KSecClassCertificate(), []attr{
		str(security.KSecAttrLabel(), label),
	})
}

// ListCertificates returns every certificate in the keychain.
func ListCertificates() ([]Certificate, error) {
	ds, err := readAll(security.KSecClassCertificate(), nil, true)
	if err != nil {
		return nil, err
	}
	out := make([]Certificate, 0, len(ds))
	for _, d := range ds {
		out = append(out, decodeCertificate(d))
	}
	return out, nil
}

func decodeCertificate(d *foundation.Dictionary) Certificate {
	return Certificate{
		Label: attrString(d, security.KSecAttrLabel()),
		DER:   attrBytes(d, security.KSecValueData()),
	}
}
