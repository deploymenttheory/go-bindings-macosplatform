package idiofw

import "testing"

// TestExactClassPointerBoundaries locks the exact-boundary matching that keeps
// longer identifiers (NSDataDetector, NSDateComponents) and out-parameters
// (double pointers) out of the value conversions.
func TestExactClassPointerBoundaries(t *testing.T) {
	cases := []struct {
		objcType  string
		className string
		want      bool
	}{
		{"NSData *", "NSData", true},
		{"NSData * _Nullable", "NSData", true},
		{"NSDataDetector *", "NSData", false},
		{"NSMutableData *", "NSData", false},
		{"NSData **", "NSData", false},            // out-parameter
		{"NSData * _Nullable *", "NSData", false}, // out-parameter, annotated
		{"NSDate *", "NSDate", true},
		{"NSDateComponents *", "NSDate", false},
		{"NSDateFormatter *", "NSDate", false},
		{"NSDateInterval *", "NSDate", false},
		{"NSData (^)(void)", "NSData", false}, // block returning NSData
		{"NSData", "NSData", false},           // not a pointer
	}
	for _, c := range cases {
		if got := isExactClassPointer(c.objcType, c.className); got != c.want {
			t.Errorf("isExactClassPointer(%q, %q) = %v; want %v", c.objcType, c.className, got, c.want)
		}
	}
}

func TestDictionaryAndSetPredicates(t *testing.T) {
	dictCases := []struct {
		objcType string
		want     bool
	}{
		{"NSDictionary<NSString *, NSNumber *> *", true},
		{"NSDictionary *", true},
		{"NSMutableDictionary<NSString *, NSNumber *> *", false},
		{"NSDictionary<NSString *, NSNumber *> **", false},
	}
	for _, c := range dictCases {
		if got := looksLikeNSDictionary(c.objcType); got != c.want {
			t.Errorf("looksLikeNSDictionary(%q) = %v; want %v", c.objcType, got, c.want)
		}
	}

	setCases := []struct {
		objcType string
		want     bool
	}{
		{"NSSet<NSString *> *", true},
		{"NSSet *", true},
		{"NSMutableSet<NSString *> *", false},
		{"NSCountedSet *", false},
		{"NSOrderedSet<NSString *> *", false},
		{"NSSet<NSString *> **", false},
	}
	for _, c := range setCases {
		if got := looksLikeNSSet(c.objcType); got != c.want {
			t.Errorf("looksLikeNSSet(%q) = %v; want %v", c.objcType, got, c.want)
		}
	}
}

func TestExtractDictKV(t *testing.T) {
	cases := []struct {
		objcType  string
		wantKey   string
		wantValue string
		wantOK    bool
	}{
		{"NSDictionary<NSString *, NSNumber *> *", "NSString *", "NSNumber *", true},
		// A nested generic value splits at the top-level comma only.
		{"NSDictionary<NSString *, NSArray<NSString *> *> *", "NSString *", "NSArray<NSString *> *", true},
		{"NSDictionary<NSString *, NSDictionary<NSString *, NSNumber *> *> *", "NSString *", "NSDictionary<NSString *, NSNumber *> *", true},
		{"NSDictionary *", "", "", false}, // ungenericized
	}
	for _, c := range cases {
		key, value, ok := extractDictKV(c.objcType)
		if key != c.wantKey || value != c.wantValue || ok != c.wantOK {
			t.Errorf("extractDictKV(%q) = (%q, %q, %v); want (%q, %q, %v)",
				c.objcType, key, value, ok, c.wantKey, c.wantValue, c.wantOK)
		}
	}
}

// TestZeroValueForValueMappedObjects locks the error-path zeros for object-kind
// results whose idiomatic type is a Go value (NSURL→string, NSDate→time.Time)
// rather than a wrapper pointer.
func TestZeroValueForValueMappedObjects(t *testing.T) {
	cases := []struct {
		kind objKind
		typ  string
		want string
	}{
		{kindObject, "string", `""`},
		{kindObject, "time.Time", "time.Time{}"},
		{kindObject, "[]byte", "nil"},
		{kindObject, "map[string]string", "nil"},
		{kindObject, "*VirtualMachine", "nil"},
		{kindArray, "[]string", "nil"},
		{kindString, "string", `""`},
	}
	for _, c := range cases {
		if got := zeroValue(c.kind, c.typ); got != c.want {
			t.Errorf("zeroValue(%v, %q) = %s; want %s", c.kind, c.typ, got, c.want)
		}
	}
}
