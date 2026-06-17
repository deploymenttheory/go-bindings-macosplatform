//go:build darwin

// Package keychain is an ergonomic wrapper over the macOS Security framework's
// keychain item APIs for the common internet-password case.
//
// It is written entirely against the idiomatic layer and the runtime bridging
// helpers — no raw bindings, no manual ObjC message sends, no CFDictionary
// plumbing or OSStatus decoding. The CoreFoundation constants come from the
// idiomatic security package (security.KSecClass() etc.), queries are built with
// the idiomatic foundation dictionary builder, and the SecItem* calls return Go
// errors directly.
package keychain

import (
	"errors"
	"fmt"
	"unsafe"

	foundation "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/foundation"
	security "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/security"

	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
)

// errSecItemNotFound is the OSStatus returned when no matching item exists.
const errSecItemNotFound = -25300

// Error is returned for a non-success keychain OSStatus. It carries both the
// numeric code (for programmatic checks) and the Security framework's
// human-readable message.
type Error struct {
	Status  int    // the OSStatus code
	Message string // Security's description of the code, if available
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("keychain: %s (OSStatus %d)", e.Message, e.Status)
	}
	return fmt.Sprintf("keychain: OSStatus %d", e.Status)
}

// describe converts the runtime OSStatus error returned by the SecItem* wrappers
// into a keychain [Error] enriched with Security's message. It passes nil and
// non-OSStatus errors through unchanged.
func describe(err error) error {
	var oserr *purego.OSStatusError
	if !errors.As(err, &oserr) {
		return err
	}
	status := oserr.Status.Int()
	return &Error{
		Status:  status,
		Message: purego.CFString(security.SecCopyErrorMessageString(status, nil)),
	}
}

// InternetPassword describes an internet-password keychain item. Server and
// Label together identify the item for store/find/delete.
type InternetPassword struct {
	Server   string // host the credential is for, e.g. "example.com"
	Account  string // the username
	Password string // the secret
	Label    string // human-readable label shown in Keychain Access
}

// StoreInternetPassword adds item to the default keychain, or updates the
// existing item when one with the same Server and Label is already present.
func StoreInternetPassword(item InternetPassword) error {
	query := baseQuery(item.Server, item.Label)

	switch _, err := security.SecItemCopyMatching(query.ID()); {
	case err == nil:
		attrs := foundation.NewMutableDictionary().
			Set(security.KSecAttrAccount(), purego.NSString(item.Account)).
			Set(security.KSecValueData(), newData(item.Password))
		return describe(security.SecItemUpdate(query.ID(), attrs.ID()))
	case isNotFound(err):
		add := baseQuery(item.Server, item.Label).
			Set(security.KSecAttrAccount(), purego.NSString(item.Account)).
			Set(security.KSecValueData(), newData(item.Password))
		_, err := security.SecItemAdd(add.ID())
		return describe(err)
	default:
		return describe(err)
	}
}

// FindInternetPassword returns the account and password stored for server and
// label. found is false (with a nil error) when no matching item exists.
func FindInternetPassword(server, label string) (account, password string, found bool, err error) {
	query := baseQuery(server, label).
		Set(security.KSecMatchLimit(), security.KSecMatchLimitOne()).
		Set(security.KSecReturnAttributes(), foundation.NewNumberWithBool(true).ID()).
		Set(security.KSecReturnData(), foundation.NewNumberWithBool(true).ID())

	resultID, err := security.SecItemCopyMatching(query.ID())
	switch {
	case err == nil:
		// proceed
	case isNotFound(err):
		return "", "", false, nil
	default:
		return "", "", false, describe(err)
	}

	result := foundation.DictionaryFromID(resultID)
	accountID := result.ObjectForKey(security.KSecAttrAccount())
	valueID := result.ObjectForKey(security.KSecValueData())
	if accountID == 0 || valueID == 0 {
		return "", "", false, errors.New("keychain item has unexpected format")
	}

	// valueID is borrowed from result; retain before adopting it as a wrapper so
	// the wrapper's releasing finalizer is balanced.
	data := foundation.DataFromID(purego.Retain(valueID))
	password = string(unsafe.Slice((*byte)(data.Bytes()), data.Length()))
	return purego.GoString(accountID), password, true, nil
}

// DeleteInternetPassword removes the item(s) matching server and label.
// Deleting an item that does not exist is not an error.
func DeleteInternetPassword(server, label string) error {
	query := foundation.NewMutableDictionary().
		Set(security.KSecClass(), security.KSecClassInternetPassword()).
		Set(security.KSecAttrServer(), purego.NSString(server)).
		Set(security.KSecAttrLabel(), purego.NSString(label))

	switch err := security.SecItemDelete(query.ID()); {
	case err == nil, isNotFound(err):
		return nil
	default:
		return describe(err)
	}
}

// baseQuery builds the class/protocol/server/label dictionary that identifies an
// internet-password item.
func baseQuery(server, label string) *foundation.MutableDictionary {
	return foundation.NewMutableDictionary().
		Set(security.KSecClass(), security.KSecClassInternetPassword()).
		Set(security.KSecAttrProtocol(), security.KSecAttrProtocolHTTPS()).
		Set(security.KSecAttrServer(), purego.NSString(server)).
		Set(security.KSecAttrLabel(), purego.NSString(label))
}

// newData boxes a secret as an NSData/CFData id for kSecValueData.
func newData(s string) purego.ID {
	b := []byte(s)
	if len(b) == 0 {
		return foundation.NewDataWithBytesLength(nil, 0).ID()
	}
	return foundation.NewDataWithBytesLength(unsafe.Pointer(&b[0]), uint(len(b))).ID()
}

// isNotFound reports whether err is the errSecItemNotFound OSStatus.
func isNotFound(err error) bool {
	var oserr *purego.OSStatusError
	return errors.As(err, &oserr) && oserr.Status.Int() == errSecItemNotFound
}
