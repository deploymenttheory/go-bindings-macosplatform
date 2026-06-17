//go:build darwin

// Hand-authored ergonomic accessors for [Dictionary]. NSDictionary is generic
// over its key and value types, which the binding generator degrades to objc.ID,
// so the generated ObjectForKey/AllKeys/AllValues hand back raw objc.ID / raw
// NSArray values. These helpers add the Go-string-keyed lookups and typed
// conversions needed to walk dictionaries (property lists, the process
// environment) without dropping to the raw bindings. They are non-generated so
// the emitter preserves them.
package foundation

import (
	raw "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
	"github.com/ebitengine/purego/objc"
)

// stringKey builds an autoreleased NSString key id from a Go string.
func stringKey(key string) objc.ID { return raw.NSStringStringWithUTF8String(key).Ptr() }

// DictionaryFromID wraps a raw NSDictionary object id as an idiomatic
// [Dictionary]. NSDictionary is generic, so the generated accessors (and
// property-list / environment APIs) hand back the raw dictionary type rather
// than *Dictionary; this bridges such values onto the ergonomic helpers below.
// Returns nil for a nil id.
func DictionaryFromID(id objc.ID) *Dictionary {
	if id == 0 {
		return nil
	}
	return &Dictionary{inner: raw.NSDictionaryFromID[objc.ID, objc.ID](purego.Retain(id))}
}

// ObjectForStringKey looks up a value by a Go string key, returning the raw
// element id (0 when absent). Use it when the value type has no dedicated
// accessor below.
func (x *Dictionary) ObjectForStringKey(key string) objc.ID {
	return x.inner.ObjectForKey(stringKey(key))
}

// StringForKey returns the NSString value for a Go string key as a Go string.
// ok is false when the key is absent.
func (x *Dictionary) StringForKey(key string) (value string, ok bool) {
	id := x.inner.ObjectForKey(stringKey(key))
	if id == 0 {
		return "", false
	}
	return purego.GoString(id), true
}

// NumberForKey returns the NSNumber value for a Go string key, or nil when the
// key is absent.
func (x *Dictionary) NumberForKey(key string) *Number {
	id := x.inner.ObjectForKey(stringKey(key))
	if id == 0 {
		return nil
	}
	return &Number{inner: raw.NSNumberFromID(purego.Retain(id))}
}

// DictionaryForKey returns the nested NSDictionary value for a Go string key, or
// nil when the key is absent — the building block for walking property lists.
func (x *Dictionary) DictionaryForKey(key string) *Dictionary {
	id := x.inner.ObjectForKey(stringKey(key))
	if id == 0 {
		return nil
	}
	return &Dictionary{inner: raw.NSDictionaryFromID[objc.ID, objc.ID](purego.Retain(id))}
}

// StringKeys returns the dictionary's keys as Go strings (non-NSString keys
// yield their -description).
func (x *Dictionary) StringKeys() []string {
	keys := x.inner.AllKeys()
	if keys == nil {
		return nil
	}
	return purego.NSArrayToSlice(keys.Ptr(), func(id objc.ID) string { return purego.GoString(id) })
}

// ToStringMap copies a string→string dictionary (e.g. the process environment)
// into a Go map. Keys whose value is absent or non-string are skipped.
func (x *Dictionary) ToStringMap() map[string]string {
	out := map[string]string{}
	for _, k := range x.StringKeys() {
		if v, ok := x.StringForKey(k); ok {
			out[k] = v
		}
	}
	return out
}
