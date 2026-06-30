//go:build darwin

// Command keychain is a runnable proof that the macOS Security framework
// bindings can drive the keychain item API through CRUD operations across item
// classes — using only the custom layer (opinionated/tools/keychain), with
// no raw FFI, CFDictionary building, or OSStatus decoding at the call site.
//
//	go run ./examples/keychain
//
// It exercises:
//   - generic password:  Create, Read, Update, Read, List, Delete
//   - internet password: Create, Read, Delete
//   - certificate:       Create, Read, Delete (a self-signed cert minted here)
//   - key:               Create, Read, Delete (an EC private key)
//   - identity:          formed from a certificate + its matching key, then Read
//
// Everything it creates it cleans up, including on failure.
package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	// ADOPTION: this example uses the *custom* layer (opinionated/tools/keychain),
	// not the raw Security bindings. The keychain item API is a multi-call workflow —
	// each operation builds a CFDictionary of kSec* attributes and decodes an
	// OSStatus — and the custom layer collapses that into CreateGenericPassword,
	// ReadKey, and friends. When a framework's *task* is this involved, look for a
	// custom package before reaching for the raw bindings. See examples/README.md
	// for the full layer guide.
	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/tools/keychain"
)

// errSecMissingEntitlement is returned when an operation needs a keychain-access
// entitlement the (unsigned) process does not have. Storing keys and identities
// in the data-protection keychain requires it; passwords and certificates in the
// file keychain do not.
const errSecMissingEntitlement = -34018

// isMissingEntitlement reports whether err is the keychain "missing entitlement"
// status, which an unsigned binary expectedly hits when creating keys/identities.
func isMissingEntitlement(err error) bool {
	var ke *keychain.Error
	return errors.As(err, &ke) && ke.Status == errSecMissingEntitlement
}

const (
	service  = "keychain-example.weave.test"
	server   = "keychain-example.weave.test"
	account  = "weave-demo-user"
	label    = "Weave SDK Keychain Example"
	keyLabel = "Weave SDK Keychain Example Key"
	idLabel  = "Weave SDK Keychain Example Identity"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
}

// run drives every item-class demo in turn, returning the first failure. Cleanup
// runs before it starts (clearing a prior interrupted run) and again on the way
// out, so the keychain is left as it was found whether run succeeds or fails.
func run() error {
	cleanup()       // clear leftovers from a prior interrupted run
	defer cleanup() // best-effort: remove anything left behind on early return

	steps := []struct {
		name string
		fn   func() error
	}{
		{"generic password", genericPassword},
		{"internet password", internetPassword},
		{"certificate", certificate},
		{"key", key},
		{"identity", identity},
	}
	for _, step := range steps {
		if err := step.fn(); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}

	fmt.Println("\nPASS: keychain CRUD across item classes")
	return nil
}

// genericPassword runs the full CRUD + List cycle for a generic password.
func genericPassword() error {
	fmt.Println("● generic password")

	if err := keychain.CreateGenericPassword(keychain.GenericPassword{
		Service: service, Account: account, Label: label, Secret: []byte("s3cr3t-original"),
	}); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	got, found, err := keychain.ReadGenericPassword(service, account)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if !found || string(got.Secret) != "s3cr3t-original" {
		return fmt.Errorf("read: got %q found=%v", got.Secret, found)
	}
	fmt.Printf("  ✓ read secret=%q\n", got.Secret)

	if err := keychain.UpdateGenericPassword(keychain.GenericPassword{
		Service: service, Account: account, Secret: []byte("s3cr3t-rotated"),
	}); err != nil {
		return fmt.Errorf("update: %w", err)
	}
	got, _, err = keychain.ReadGenericPassword(service, account)
	if err != nil {
		return fmt.Errorf("read-after-update: %w", err)
	}
	if string(got.Secret) != "s3cr3t-rotated" {
		return fmt.Errorf("update: secret did not rotate: %q", got.Secret)
	}
	fmt.Printf("  ✓ updated secret=%q\n", got.Secret)

	all, err := keychain.ListGenericPasswords()
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	if !containsService(all, service) {
		return fmt.Errorf("list: created item not present among %d listed", len(all))
	}
	fmt.Printf("  ✓ listed (%d items; ours present, no secrets exported)\n", len(all))

	if err := keychain.DeleteGenericPassword(service, account); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if _, found, err := keychain.ReadGenericPassword(service, account); err != nil {
		return fmt.Errorf("delete-verify: %w", err)
	} else if found {
		return errors.New("delete: still present after delete")
	}
	fmt.Println("  ✓ deleted")
	return nil
}

// internetPassword runs Create, Read, Delete for an internet password.
func internetPassword() error {
	fmt.Println("● internet password")

	if err := keychain.CreateInternetPassword(keychain.InternetPassword{
		Server: server, Account: account, Label: label, Secret: []byte("https-secret"),
	}); err != nil {
		return fmt.Errorf("create: %w", err)
	}
	got, found, err := keychain.ReadInternetPassword(server, account)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if !found || string(got.Secret) != "https-secret" {
		return fmt.Errorf("read: got %q found=%v", got.Secret, found)
	}
	fmt.Printf("  ✓ read secret=%q\n", got.Secret)

	if err := keychain.DeleteInternetPassword(server, account); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	fmt.Println("  ✓ deleted")
	return nil
}

// certificate runs Create, Read, Delete for a self-signed certificate.
func certificate() error {
	fmt.Println("● certificate")
	priv, err := newECKey()
	if err != nil {
		return err
	}
	der, err := selfSignedDER(priv, label)
	if err != nil {
		return err
	}

	if err := keychain.CreateCertificate(keychain.Certificate{Label: label, DER: der}); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	got, found, err := keychain.ReadCertificate(label)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if !found || !bytes.Equal(got.DER, der) {
		return fmt.Errorf("read: DER mismatch or not found (found=%v, %d bytes)", found, len(got.DER))
	}
	fmt.Printf("  ✓ read DER (%d bytes) matches\n", len(got.DER))

	if err := keychain.DeleteCertificate(label); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if _, found, err := keychain.ReadCertificate(label); err != nil {
		return fmt.Errorf("delete-verify: %w", err)
	} else if found {
		return errors.New("delete: still present after delete")
	}
	fmt.Println("  ✓ deleted")
	return nil
}

// key runs Create, Read, Delete for an EC private key. On an unsigned binary the
// Create is expected to fail for lack of a code-signing entitlement; the demo then
// falls back to read-only enumeration so the call path is still exercised.
func key() error {
	fmt.Println("● key")
	priv, err := newECKey()
	if err != nil {
		return err
	}
	material, err := ecX963(priv)
	if err != nil {
		return err
	}

	switch err := keychain.CreateKey(keychain.Key{
		Label: keyLabel, Type: keychain.KeyEC, Data: material,
	}); {
	case isMissingEntitlement(err):
		keys, lerr := keychain.ListKeys()
		if lerr != nil {
			return fmt.Errorf("list: %w", lerr)
		}
		fmt.Printf("  ⓘ Create needs a code-signing entitlement (unsigned binary); enumerated %d existing key(s)\n", len(keys))
		return nil
	case err != nil:
		return fmt.Errorf("create: %w", err)
	}

	got, found, err := keychain.ReadKey(keyLabel)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if !found {
		return errors.New("read: key not found after create")
	}
	fmt.Printf("  ✓ read label=%q applicationLabel=%s\n", got.Label, hex.EncodeToString(got.ApplicationLabel))

	if err := keychain.DeleteKey(keyLabel); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if _, found, err := keychain.ReadKey(keyLabel); err != nil {
		return fmt.Errorf("delete-verify: %w", err)
	} else if found {
		return errors.New("delete: key still present after delete")
	}
	fmt.Println("  ✓ deleted")
	return nil
}

// identity forms an identity from a certificate and its matching key, then reads
// it back. Like key, it falls back to read-only enumeration on an unsigned binary.
func identity() error {
	fmt.Println("● identity (formed from a certificate + its matching key)")
	priv, err := newECKey()
	if err != nil {
		return err
	}
	material, err := ecX963(priv)
	if err != nil {
		return err
	}

	switch err := keychain.CreateKey(keychain.Key{
		Label: idLabel, Type: keychain.KeyEC, Data: material,
	}); {
	case isMissingEntitlement(err):
		ids, lerr := keychain.ListIdentities()
		if lerr != nil {
			return fmt.Errorf("list: %w", lerr)
		}
		fmt.Printf("  ⓘ forming an identity needs a code-signing entitlement (unsigned binary); enumerated %d existing identity(ies)\n", len(ids))
		return nil
	case err != nil:
		return fmt.Errorf("add key: %w", err)
	}

	der, err := selfSignedDER(priv, idLabel)
	if err != nil {
		return err
	}
	if err := keychain.CreateCertificate(keychain.Certificate{Label: idLabel, DER: der}); err != nil {
		return fmt.Errorf("add cert: %w", err)
	}

	id, found, err := keychain.ReadIdentity(idLabel)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if !found {
		return errors.New("read: identity did not form from cert + key")
	}
	fmt.Printf("  ✓ identity formed, label=%q\n", id.Label)

	if err := keychain.DeleteIdentity(idLabel); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	// Deleting the identity removes the certificate; remove the key too.
	_ = keychain.DeleteKey(idLabel)
	fmt.Println("  ✓ deleted")
	return nil
}

// ── certificate/key material helpers ─────────────────────────────────────────

// newECKey generates a fresh P-256 private key.
func newECKey() (*ecdsa.PrivateKey, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate EC key: %w", err)
	}
	return priv, nil
}

// selfSignedDER mints a throwaway self-signed certificate for priv and returns
// its DER. macOS derives the certificate's kSecAttrLabel from the subject CN, so
// the CN is set to the label the example queries by.
func selfSignedDER(priv *ecdsa.PrivateKey, cn string) ([]byte, error) {
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %w", err)
	}
	return der, nil
}

// ecX963 encodes an EC private key in the ANSI X9.63 form SecKeyCreateWithData
// expects: the uncompressed public point (0x04 || X || Y) followed by the
// private scalar K.
func ecX963(priv *ecdsa.PrivateKey) ([]byte, error) {
	k, err := priv.ECDH()
	if err != nil {
		return nil, fmt.Errorf("convert key to ECDH form: %w", err)
	}
	return append(k.PublicKey().Bytes(), k.Bytes()...), nil
}

// containsService reports whether any listed item has the given service.
func containsService(items []keychain.GenericPassword, svc string) bool {
	for _, it := range items {
		if it.Service == svc {
			return true
		}
	}
	return false
}

// cleanup removes anything a previous interrupted run may have left behind. Errors
// are intentionally ignored: each delete is best-effort and a missing item is the
// normal case.
func cleanup() {
	_ = keychain.DeleteGenericPassword(service, account)
	_ = keychain.DeleteInternetPassword(server, account)
	_ = keychain.DeleteCertificate(label)
	_ = keychain.DeleteKey(keyLabel)
	_ = keychain.DeleteIdentity(idLabel)
	_ = keychain.DeleteCertificate(idLabel)
	_ = keychain.DeleteKey(idLabel)
}
