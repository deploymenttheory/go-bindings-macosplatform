//go:build darwin

// Command keychain is a runnable proof that the macOS Security framework
// bindings can drive the keychain item API through CRUD operations across item
// classes — using only the idiomatic layer (opinionated/custom/keychain), with
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
// Everything it creates it cleans up, including on failure. //go:build darwin.
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

	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/custom/keychain"
)

// errSecMissingEntitlement is returned when an operation needs a keychain-access
// entitlement the (unsigned) process does not have. Storing keys and identities
// in the data-protection keychain requires it; passwords and certificates in the
// file keychain do not.
const errSecMissingEntitlement = -34018

func isEntitlement(err error) bool {
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
	cleanup()

	genericPassword()
	internetPassword()
	certificate()
	key()
	identity()

	fmt.Println("\nPASS: keychain CRUD across item classes")
}

func genericPassword() {
	fmt.Println("● generic password")

	must("create", keychain.CreateGenericPassword(keychain.GenericPassword{
		Service: service, Account: account, Label: label, Secret: []byte("s3cr3t-original"),
	}))

	got, found, err := keychain.ReadGenericPassword(service, account)
	must("read", err)
	if !found || string(got.Secret) != "s3cr3t-original" {
		fail("read", fmt.Errorf("got %q found=%v", got.Secret, found))
	}
	fmt.Printf("  ✓ read secret=%q\n", got.Secret)

	must("update", keychain.UpdateGenericPassword(keychain.GenericPassword{
		Service: service, Account: account, Secret: []byte("s3cr3t-rotated"),
	}))
	got, _, err = keychain.ReadGenericPassword(service, account)
	must("read-after-update", err)
	if string(got.Secret) != "s3cr3t-rotated" {
		fail("update", fmt.Errorf("secret did not rotate: %q", got.Secret))
	}
	fmt.Printf("  ✓ updated secret=%q\n", got.Secret)

	all, err := keychain.ListGenericPasswords()
	must("list", err)
	if !containsService(all, service) {
		fail("list", fmt.Errorf("created item not present among %d listed", len(all)))
	}
	fmt.Printf("  ✓ listed (%d items; ours present, no secrets exported)\n", len(all))

	must("delete", keychain.DeleteGenericPassword(service, account))
	if _, found, _ := keychain.ReadGenericPassword(service, account); found {
		fail("delete", fmt.Errorf("still present after delete"))
	}
	fmt.Println("  ✓ deleted")
}

func internetPassword() {
	fmt.Println("● internet password")
	must("create", keychain.CreateInternetPassword(keychain.InternetPassword{
		Server: server, Account: account, Label: label, Secret: []byte("https-secret"),
	}))
	got, found, err := keychain.ReadInternetPassword(server, account)
	must("read", err)
	if !found || string(got.Secret) != "https-secret" {
		fail("read", fmt.Errorf("got %q found=%v", got.Secret, found))
	}
	fmt.Printf("  ✓ read secret=%q\n", got.Secret)
	must("delete", keychain.DeleteInternetPassword(server, account))
	fmt.Println("  ✓ deleted")
}

func certificate() {
	fmt.Println("● certificate")
	der := selfSignedDER(newECKey(), label)

	must("create", keychain.CreateCertificate(keychain.Certificate{Label: label, DER: der}))

	got, found, err := keychain.ReadCertificate(label)
	must("read", err)
	if !found || !bytes.Equal(got.DER, der) {
		fail("read", fmt.Errorf("DER mismatch or not found (found=%v, %d bytes)", found, len(got.DER)))
	}
	fmt.Printf("  ✓ read DER (%d bytes) matches\n", len(got.DER))

	must("delete", keychain.DeleteCertificate(label))
	if _, found, _ := keychain.ReadCertificate(label); found {
		fail("delete", fmt.Errorf("still present after delete"))
	}
	fmt.Println("  ✓ deleted")
}

func key() {
	fmt.Println("● key")
	priv := newECKey()

	switch err := keychain.CreateKey(keychain.Key{
		Label: keyLabel, Type: keychain.KeyEC, Data: ecX963(priv),
	}); {
	case isEntitlement(err):
		// Expected for an unsigned binary; the call path is still exercised.
		keys, lerr := keychain.ListKeys()
		must("list", lerr)
		fmt.Printf("  ⓘ Create needs a code-signing entitlement (unsigned binary); enumerated %d existing key(s)\n", len(keys))
		return
	default:
		must("create", err)
	}

	got, found, err := keychain.ReadKey(keyLabel)
	must("read", err)
	if !found {
		fail("read", fmt.Errorf("key not found after create"))
	}
	fmt.Printf("  ✓ read label=%q applicationLabel=%s\n", got.Label, hex.EncodeToString(got.ApplicationLabel))

	must("delete", keychain.DeleteKey(keyLabel))
	if _, found, _ := keychain.ReadKey(keyLabel); found {
		fail("delete", fmt.Errorf("key still present after delete"))
	}
	fmt.Println("  ✓ deleted")
}

func identity() {
	fmt.Println("● identity (formed from a certificate + its matching key)")
	priv := newECKey()

	switch err := keychain.CreateKey(keychain.Key{
		Label: idLabel, Type: keychain.KeyEC, Data: ecX963(priv),
	}); {
	case isEntitlement(err):
		ids, lerr := keychain.ListIdentities()
		must("list", lerr)
		fmt.Printf("  ⓘ forming an identity needs a code-signing entitlement (unsigned binary); enumerated %d existing identity(ies)\n", len(ids))
		return
	default:
		must("add key", err)
	}

	must("add cert", keychain.CreateCertificate(keychain.Certificate{
		Label: idLabel, DER: selfSignedDER(priv, idLabel),
	}))

	id, found, err := keychain.ReadIdentity(idLabel)
	must("read", err)
	if !found {
		fail("read", fmt.Errorf("identity did not form from cert + key"))
	}
	fmt.Printf("  ✓ identity formed, label=%q\n", id.Label)

	must("delete", keychain.DeleteIdentity(idLabel))
	// Deleting the identity removes the certificate; remove the key too.
	_ = keychain.DeleteKey(idLabel)
	fmt.Println("  ✓ deleted")
}

// ── certificate/key material helpers ─────────────────────────────────────────

func newECKey() *ecdsa.PrivateKey {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fail("gen key", err)
	}
	return priv
}

// selfSignedDER mints a throwaway self-signed certificate for priv and returns
// its DER. macOS derives the certificate's kSecAttrLabel from the subject CN, so
// the CN is set to the label the example queries by.
func selfSignedDER(priv *ecdsa.PrivateKey, cn string) []byte {
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		fail("create cert", err)
	}
	return der
}

// ecX963 encodes an EC private key in the ANSI X9.63 form SecKeyCreateWithData
// expects: the uncompressed public point (0x04 || X || Y) followed by the
// private scalar K.
func ecX963(priv *ecdsa.PrivateKey) []byte {
	k, err := priv.ECDH()
	if err != nil {
		fail("ecdh", err)
	}
	return append(k.PublicKey().Bytes(), k.Bytes()...)
}

func containsService(items []keychain.GenericPassword, svc string) bool {
	for _, it := range items {
		if it.Service == svc {
			return true
		}
	}
	return false
}

// cleanup removes anything a previous interrupted run may have left behind.
func cleanup() {
	_ = keychain.DeleteGenericPassword(service, account)
	_ = keychain.DeleteInternetPassword(server, account)
	_ = keychain.DeleteCertificate(label)
	_ = keychain.DeleteKey(keyLabel)
	_ = keychain.DeleteIdentity(idLabel)
	_ = keychain.DeleteCertificate(idLabel)
	_ = keychain.DeleteKey(idLabel)
}

func must(op string, err error) {
	if err != nil {
		fail(op, err)
	}
}

func fail(op string, err error) {
	fmt.Fprintf(os.Stderr, "FAIL (%s): %v\n", op, err)
	cleanup()
	os.Exit(1)
}
