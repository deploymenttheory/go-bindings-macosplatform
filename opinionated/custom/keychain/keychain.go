//go:build darwin

// Package keychain is an ergonomic CRUD wrapper over the macOS Security
// framework's keychain item API (SecItemAdd/CopyMatching/Update/Delete), with
// typed facades for each keychain item class.
//
// It is written entirely against the idiomatic layer and the runtime bridging
// helpers — no raw bindings, no manual ObjC message sends, no CFDictionary
// plumbing or OSStatus decoding. CoreFoundation constants come from the idiomatic
// security package (security.KSecClass() etc.), queries are built with the
// idiomatic foundation dictionary builder, and the SecItem* calls return Go
// errors directly.
//
// The five item classes share one class-agnostic CRUD core in this file; each
// class adds a typed struct and Create/Read/Update/Delete/List that map its
// fields to the relevant kSec* attributes. The verbs follow the standard CRUD
// mapping onto Security:
//
//	Create -> SecItemAdd          (fails if a matching item already exists)
//	Read   -> SecItemCopyMatching (kSecMatchLimitOne)
//	Update -> SecItemUpdate
//	Delete -> SecItemDelete
//	List   -> SecItemCopyMatching (kSecMatchLimitAll)
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
// into a keychain [Error] enriched with Security's message. nil and non-OSStatus
// errors pass through unchanged.
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

// isNotFound reports whether err is the errSecItemNotFound OSStatus.
func isNotFound(err error) bool {
	var oserr *purego.OSStatusError
	return errors.As(err, &oserr) && oserr.Status.Int() == errSecItemNotFound
}

// ── class-agnostic CRUD core ─────────────────────────────────────────────────
//
// Each facade describes an item as a kSecClass value plus a list of attributes
// (key/value object pointers). The core builds the CFDictionary and performs the
// SecItem* call.

// attr is one key/value pair for an item dictionary.
type attr struct{ key, value purego.ID }

// str builds a string-valued attribute (e.g. an account name).
func str(key purego.ID, s string) attr { return attr{key, purego.NSString(s)} }

// blob builds an NSData-valued attribute (e.g. kSecValueData).
func blob(key purego.ID, b []byte) attr { return attr{key, newData(b)} }

// ref builds an attribute whose value is an existing object/CFTypeRef id
// (e.g. kSecValueRef with a SecCertificateRef).
func ref(key, value purego.ID) attr { return attr{key, value} }

// dict builds a mutable item dictionary: the class plus each attribute.
func dict(class purego.ID, attrs []attr) *foundation.MutableDictionary {
	d := foundation.NewMutableDictionary().Set(security.KSecClass(), class)
	for _, a := range attrs {
		d.Set(a.key, a.value)
	}
	return d
}

// trueValue is kCFBooleanTrue as an NSNumber id, for the kSecReturn* flags.
func trueValue() purego.ID { return foundation.NewNumberWithBool(true).ID() }

// create adds a new item; it fails (errSecDuplicateItem) if a matching item
// already exists.
func create(class purego.ID, attrs []attr) error {
	_, err := security.SecItemAdd(dict(class, attrs).ID())
	return describe(err)
}

// readOne returns the single item matching query, or found=false when absent.
// withData additionally requests the item's kSecValueData.
func readOne(class purego.ID, query []attr, withData bool) (*foundation.Dictionary, bool, error) {
	q := dict(class, query).
		Set(security.KSecMatchLimit(), security.KSecMatchLimitOne()).
		Set(security.KSecReturnAttributes(), trueValue())
	if withData {
		q.Set(security.KSecReturnData(), trueValue())
	}
	switch id, err := security.SecItemCopyMatching(q.ID()); {
	case err == nil:
		return foundation.DictionaryFromID(id), true, nil
	case isNotFound(err):
		return nil, false, nil
	default:
		return nil, false, describe(err)
	}
}

// readAll returns every item matching query (nil when none match). withData
// additionally requests each item's kSecValueData.
func readAll(class purego.ID, query []attr, withData bool) ([]*foundation.Dictionary, error) {
	q := dict(class, query).
		Set(security.KSecMatchLimit(), security.KSecMatchLimitAll()).
		Set(security.KSecReturnAttributes(), trueValue())
	if withData {
		q.Set(security.KSecReturnData(), trueValue())
	}
	switch id, err := security.SecItemCopyMatching(q.ID()); {
	case err == nil:
		arr := foundation.ArrayFromID(id)
		return purego.NSArrayToSlice(arr.ID(), func(e purego.ID) *foundation.Dictionary {
			return foundation.DictionaryFromID(purego.Retain(e))
		}), nil
	case isNotFound(err):
		return nil, nil
	default:
		return nil, describe(err)
	}
}

// update modifies the attributes of the item(s) matching query.
func update(class purego.ID, query, changes []attr) error {
	c := foundation.NewMutableDictionary()
	for _, a := range changes {
		c.Set(a.key, a.value)
	}
	return describe(security.SecItemUpdate(dict(class, query).ID(), c.ID()))
}

// remove deletes the item(s) matching query. Deleting a non-existent item is not
// an error.
func remove(class purego.ID, query []attr) error {
	switch err := security.SecItemDelete(dict(class, query).ID()); {
	case err == nil, isNotFound(err):
		return nil
	default:
		return describe(err)
	}
}

// ── result decoding ──────────────────────────────────────────────────────────

// attrString reads a string-valued attribute from a result dictionary ("" when
// absent).
func attrString(d *foundation.Dictionary, key purego.ID) string {
	return purego.GoString(d.ObjectForKey(key))
}

// attrBytes reads an NSData-valued attribute from a result dictionary (nil when
// absent), copied into a Go slice.
func attrBytes(d *foundation.Dictionary, key purego.ID) []byte {
	id := d.ObjectForKey(key)
	if id == 0 {
		return nil
	}
	data := foundation.DataFromID(purego.Retain(id))
	n := data.Length()
	if n == 0 {
		return nil
	}
	return append([]byte(nil), unsafe.Slice((*byte)(data.Bytes()), n)...)
}

// newData boxes a Go byte slice as an NSData/CFData id.
func newData(b []byte) purego.ID {
	if len(b) == 0 {
		return foundation.NewDataWithBytesLength(nil, 0).ID()
	}
	return foundation.NewDataWithBytesLength(unsafe.Pointer(&b[0]), uint(len(b))).ID()
}
