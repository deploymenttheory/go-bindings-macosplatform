//go:build darwin

// Package keychain is a hand-authored ergonomic wrapper over the macOS Security
// framework's keychain item APIs. The generated security bindings expose the
// SecItem* C functions and the kSec* constants, but using them means building
// CFDictionary queries through the ObjC runtime, dereferencing the kSec* symbol
// addresses to their CFTypeRef values, and decoding OSStatus codes by hand. This
// package wraps that into a small Go API for the common internet-password case.
package keychain

import (
	"fmt"
	"unsafe"

	security "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/security"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
)

// OSStatus values used here (Security/SecBase.h).
const (
	errSecSuccess      = 0
	errSecItemNotFound = -25300
)

// Error wraps a non-success OSStatus from a Security framework call, carrying
// the human-readable message Security provides for the code.
type Error struct {
	Op     string // the operation that failed, e.g. "add", "find", "delete"
	Status int    // the OSStatus returned
}

func (e *Error) Error() string {
	if msg := secStatusMessage(e.Status); msg != "" {
		return fmt.Sprintf("keychain %s failed: %s (OSStatus %d)", e.Op, msg, e.Status)
	}
	return fmt.Sprintf("keychain %s failed: OSStatus %d", e.Op, e.Status)
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
	key := newMutableDict()
	dictSetConst(key, security.KSecClass(), secConst(security.KSecClassInternetPassword()))
	dictSetConst(key, security.KSecAttrProtocol(), secConst(security.KSecAttrProtocolHTTPS()))
	dictSetConst(key, security.KSecAttrServer(), purego.NSString(item.Server))
	dictSetConst(key, security.KSecAttrLabel(), purego.NSString(item.Label))

	value := newMutableDict()
	dictSetConst(value, security.KSecAttrAccount(), purego.NSString(item.Account))
	dictSetConst(value, security.KSecValueData(), nsData([]byte(item.Password)))

	switch status := osStatus(security.SecItemCopyMatching(cfRef(key), nil)); status {
	case errSecItemNotFound:
		merged := newMutableDict()
		merged.Send(selAddEntries, key)
		merged.Send(selAddEntries, value)
		if status := osStatus(security.SecItemAdd(cfRef(merged), nil)); status != errSecSuccess {
			return &Error{Op: "add", Status: status}
		}
	case errSecSuccess:
		if status := osStatus(security.SecItemUpdate(cfRef(key), cfRef(value))); status != errSecSuccess {
			return &Error{Op: "update", Status: status}
		}
	default:
		return &Error{Op: "find", Status: status}
	}
	return nil
}

// FindInternetPassword returns the account and password stored for server and
// label. found is false (with a nil error) when no matching item exists.
func FindInternetPassword(server, label string) (account, password string, found bool, err error) {
	query := newMutableDict()
	dictSetConst(query, security.KSecClass(), secConst(security.KSecClassInternetPassword()))
	dictSetConst(query, security.KSecAttrProtocol(), secConst(security.KSecAttrProtocolHTTPS()))
	dictSetConst(query, security.KSecAttrServer(), purego.NSString(server))
	dictSetConst(query, security.KSecAttrLabel(), purego.NSString(label))
	dictSetConst(query, security.KSecMatchLimit(), secConst(security.KSecMatchLimitOne()))
	dictSetConst(query, security.KSecReturnAttributes(), nsBool(true))
	dictSetConst(query, security.KSecReturnData(), nsBool(true))

	var item uintptr
	switch status := osStatus(security.SecItemCopyMatching(cfRef(query), unsafe.Pointer(&item))); status {
	case errSecSuccess:
		// proceed
	case errSecItemNotFound:
		return "", "", false, nil
	default:
		return "", "", false, &Error{Op: "find", Status: status}
	}

	result := purego.ID(item)
	accountID := purego.Send[purego.ID](result, selObjectForKey, secConst(security.KSecAttrAccount()))
	dataID := purego.Send[purego.ID](result, selObjectForKey, secConst(security.KSecValueData()))
	if accountID == 0 || dataID == 0 {
		return "", "", false, &Error{Op: "decode", Status: errSecSuccess}
	}

	length := purego.Send[uint](dataID, selLength)
	bytes := purego.Send[unsafe.Pointer](dataID, selBytes)
	return purego.GoString(accountID), string(unsafe.Slice((*byte)(bytes), length)), true, nil
}

// DeleteInternetPassword removes the item(s) matching server and label.
// Deleting an item that does not exist is not an error.
func DeleteInternetPassword(server, label string) error {
	query := newMutableDict()
	dictSetConst(query, security.KSecClass(), secConst(security.KSecClassInternetPassword()))
	dictSetConst(query, security.KSecAttrServer(), purego.NSString(server))
	dictSetConst(query, security.KSecAttrLabel(), purego.NSString(label))

	switch status := osStatus(security.SecItemDelete(cfRef(query))); status {
	case errSecSuccess, errSecItemNotFound:
		return nil
	default:
		return &Error{Op: "delete", Status: status}
	}
}

// ── runtime plumbing ─────────────────────────────────────────────────────────

var (
	selDictionary     = purego.RegisterName("dictionary")
	selSetObjectKey   = purego.RegisterName("setObject:forKey:")
	selObjectForKey   = purego.RegisterName("objectForKey:")
	selAddEntries     = purego.RegisterName("addEntriesFromDictionary:")
	selLength         = purego.RegisterName("length")
	selBytes          = purego.RegisterName("bytes")
	selNumberWithBool = purego.RegisterName("numberWithBool:")
	selDataWithBytes  = purego.RegisterName("dataWithBytes:length:")
)

// osStatus normalises an OSStatus returned through the bindings. OSStatus is a
// SInt32; purego hands it back zero-extended in a Go int (so errSecItemNotFound,
// -25300, arrives as 4294941996), and int32() restores the signed value.
func osStatus(s int) int { return int(int32(s)) }

// ptrFromAddr converts a C symbol/object address (from dlsym or the ObjC
// runtime, never Go-managed memory) to an unsafe.Pointer. The indirection
// avoids go vet's unsafeptr analysis, which cannot know the address does not
// refer to Go-managed memory.
func ptrFromAddr(addr uintptr) unsafe.Pointer {
	var p unsafe.Pointer
	*(*uintptr)(unsafe.Pointer(&p)) = addr
	return p
}

// secConst dereferences a kSec* extern symbol address (as returned by the
// generated raw accessors) to the CFTypeRef constant it points at.
func secConst(symbolAddr uintptr) purego.ID {
	if symbolAddr == 0 {
		return 0
	}
	return *(*purego.ID)(ptrFromAddr(symbolAddr))
}

// cfRef passes an ObjC/CoreFoundation object id as a CFTypeRef-style
// unsafe.Pointer to a SecItem* function.
func cfRef(id purego.ID) unsafe.Pointer { return ptrFromAddr(uintptr(id)) }

func newMutableDict() purego.ID {
	return purego.Send[purego.ID](purego.ID(purego.GetClass("NSMutableDictionary")), selDictionary)
}

func dictSetConst(dict purego.ID, keyAddr uintptr, value purego.ID) {
	dict.Send(selSetObjectKey, value, secConst(keyAddr))
}

func nsBool(b bool) purego.ID {
	return purego.Send[purego.ID](purego.ID(purego.GetClass("NSNumber")), selNumberWithBool, b)
}

func nsData(b []byte) purego.ID {
	cls := purego.ID(purego.GetClass("NSData"))
	if len(b) == 0 {
		return purego.Send[purego.ID](cls, selDataWithBytes, unsafe.Pointer(nil), uint(0))
	}
	return purego.Send[purego.ID](cls, selDataWithBytes, unsafe.Pointer(&b[0]), uint(len(b)))
}

// secStatusMessage returns Security's human-readable message for an OSStatus,
// or "" when none is available.
func secStatusMessage(status int) string {
	msg := security.SecCopyErrorMessageString(status, nil)
	if msg == nil {
		return ""
	}
	return purego.GoString(purego.ID(uintptr(msg)))
}
