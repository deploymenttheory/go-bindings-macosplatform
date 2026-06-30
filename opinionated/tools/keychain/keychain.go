//go:build darwin

// Package keychain is an ergonomic CRUD wrapper over the macOS Security
// framework's keychain item API (SecItemAdd/CopyMatching/Update/Delete), with
// typed facades for each keychain item class.
//
// It is written entirely against the idiomatic layer and its runtime support
// packages — no raw bindings, no manual ObjC message sends, no CFDictionary
// plumbing or OSStatus decoding. CoreFoundation constants come from the idiomatic
// security package (security.KSecClass() etc.) as obj.Object values, queries are
// built with the idiomatic foundation dictionary builder, and the SecItem* calls
// return Go errors directly.
//
// Every value handed to or read back from the keychain is an obj.Object — the
// interface every idiomatic wrapper satisfies. The constants, the dictionaries
// built here, and the wrappers returned by the foundation constructors are all
// obj.Object, so they pass straight into the SecItem* calls with no raw-pointer
// conversions: the whole file stays in the idiomatic layer.
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

	foundation "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/foundation"
	security "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/security"
	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/obj"
	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/rt"

	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
)

// errSecItemNotFound is the OSStatus returned when no matching item exists.
const errSecItemNotFound = -25300

// Error is returned for a non-success keychain OSStatus. It carries the numeric
// code for programmatic checks (compare against the documented sec error codes,
// or rely on the package's not-found handling internally).
type Error struct {
	Status int // the OSStatus code
}

func (e *Error) Error() string {
	return fmt.Sprintf("keychain: OSStatus %d", e.Status)
}

// describe converts the runtime OSStatus error returned by the SecItem* wrappers
// into a keychain [Error]. nil and non-OSStatus errors pass through unchanged.
func describe(err error) error {
	var oserr *purego.OSStatusError
	if !errors.As(err, &oserr) {
		return err
	}
	return &Error{Status: oserr.Status.Int()}
}

// isNotFound reports whether err is the errSecItemNotFound OSStatus.
func isNotFound(err error) bool {
	var oserr *purego.OSStatusError
	return errors.As(err, &oserr) && oserr.Status.Int() == errSecItemNotFound
}

// ── class-agnostic CRUD core ─────────────────────────────────────────────────
//
// Each facade describes an item as a kSecClass value plus a list of attributes
// (key/value obj.Object pairs). The core builds the dictionary and performs the
// SecItem* call.

// attr is one key/value pair for an item dictionary.
type attr struct{ key, value obj.Object }

// str builds a string-valued attribute (e.g. an account name).
func str(key obj.Object, s string) attr { return attr{key, obj.Wrap(purego.NSString(s))} }

// blob builds an NSData-valued attribute (e.g. kSecValueData).
func blob(key obj.Object, b []byte) attr { return attr{key, newData(b)} }

// ref builds an attribute whose value is an existing object/CFTypeRef
// (e.g. kSecValueRef with a SecCertificateRef).
func ref(key, value obj.Object) attr { return attr{key, value} }

// dict builds a mutable item dictionary: the class plus each attribute. The
// returned *MutableDictionary is itself an obj.Object, so it passes straight into
// the SecItem* calls.
func dict(class obj.Object, attrs []attr) *foundation.MutableDictionary {
	d := foundation.NewMutableDictionary().Set(security.KSecClass(), class)
	for _, a := range attrs {
		d.Set(a.key, a.value)
	}
	return d
}

// trueValue is kCFBooleanTrue as an NSNumber, for the kSecReturn* flags.
func trueValue() obj.Object { return foundation.NewNumberWithBool(true) }

// create adds a new item; it fails (errSecDuplicateItem) if a matching item
// already exists.
func create(class obj.Object, attrs []attr) error {
	_, err := security.SecItemAdd(dict(class, attrs))
	return describe(err)
}

// readOne returns the single item matching query, or found=false when absent.
// withData additionally requests the item's kSecValueData.
func readOne(class obj.Object, query []attr, withData bool) (*foundation.Dictionary, bool, error) {
	q := dict(class, query).
		Set(security.KSecMatchLimit(), security.KSecMatchLimitOne()).
		Set(security.KSecReturnAttributes(), trueValue())
	if withData {
		q.Set(security.KSecReturnData(), trueValue())
	}
	switch result, err := security.SecItemCopyMatching(q); {
	case err == nil:
		return foundation.DictionaryFromID(obj.ID(result)), true, nil
	case isNotFound(err):
		return nil, false, nil
	default:
		return nil, false, describe(err)
	}
}

// readAll returns every item matching query (nil when none match). withData
// additionally requests each item's kSecValueData.
func readAll(class obj.Object, query []attr, withData bool) ([]*foundation.Dictionary, error) {
	q := dict(class, query).
		Set(security.KSecMatchLimit(), security.KSecMatchLimitAll()).
		Set(security.KSecReturnAttributes(), trueValue())
	if withData {
		q.Set(security.KSecReturnData(), trueValue())
	}
	switch result, err := security.SecItemCopyMatching(q); {
	case err == nil:
		return purego.NSArrayToSlice(obj.ID(result), func(e purego.ID) *foundation.Dictionary {
			return foundation.DictionaryFromID(purego.Retain(e))
		}), nil
	case isNotFound(err):
		return nil, nil
	default:
		return nil, describe(err)
	}
}

// update modifies the attributes of the item(s) matching query.
func update(class obj.Object, query, changes []attr) error {
	c := foundation.NewMutableDictionary()
	for _, a := range changes {
		c.Set(a.key, a.value)
	}
	return describe(security.SecItemUpdate(dict(class, query), c))
}

// remove deletes the item(s) matching query. Deleting a non-existent item is not
// an error.
func remove(class obj.Object, query []attr) error {
	switch err := security.SecItemDelete(dict(class, query)); {
	case err == nil, isNotFound(err):
		return nil
	default:
		return describe(err)
	}
}

// ── result decoding ──────────────────────────────────────────────────────────

// attrString reads a string-valued attribute from a result dictionary ("" when
// absent).
func attrString(d *foundation.Dictionary, key obj.Object) string {
	return purego.GoString(obj.ID(d.ObjectForKey(key)))
}

// attrBytes reads an NSData-valued attribute from a result dictionary (nil when
// absent), copied into a Go slice.
func attrBytes(d *foundation.Dictionary, key obj.Object) []byte {
	return obj.Bytes(d.ObjectForKey(key))
}

// newData boxes a Go byte slice as an NSData (an obj.Object).
func newData(b []byte) obj.Object {
	return obj.Wrap(rt.BytesToNSData(b))
}
