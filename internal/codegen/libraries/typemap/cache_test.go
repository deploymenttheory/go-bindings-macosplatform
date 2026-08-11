package typemap

import "testing"

// TestCacheHitReturnsCorrectType verifies that a cached lookup returns the same
// result as the first (uncached) call.
func TestCacheHitReturnsCorrectType(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "AppKit")

	first := m.GoType("NSString *", ctx, nil)
	if first != "*foundation.NSString" {
		t.Fatalf("first call: got %q, want *foundation.NSString", first)
	}
	if m.cache == nil || len(m.cache) == 0 {
		t.Fatal("cache should be populated after first call")
	}

	// Second call must hit the cache and return the same result.
	ctx2 := newTestCtx(m, "AppKit")
	second := m.GoType("NSString *", ctx2, nil)
	if second != first {
		t.Fatalf("cached call: got %q, want %q", second, first)
	}
}

// TestCacheReplaysCrossFrameworkImports verifies that cache hits still populate
// the ImportSet, so callers always get the correct import side effects.
func TestCacheReplaysCrossFrameworkImports(t *testing.T) {
	m := newTestMapper()

	// Warm the cache.
	warm := newTestCtx(m, "AppKit")
	m.GoType("NSString *", warm, nil)

	// Hit the cache with a fresh ImportSet.
	hit := newTestCtx(m, "AppKit")
	imports := make(ImportSet)
	m.GoType("NSString *", hit, imports)

	if imports["foundation"] == "" {
		t.Fatal("cache hit did not replay foundation import into imports")
	}
}

// TestCacheDifferentiatesByFramework verifies that the same ObjC type resolves
// differently in different framework contexts (same-fw vs cross-fw).
func TestCacheDifferentiatesByFramework(t *testing.T) {
	m := newTestMapper()

	appkitCtx := newTestCtx(m, "AppKit")
	foundCtx := newTestCtx(m, "Foundation")

	inAppKit := m.GoType("NSString *", appkitCtx, nil)
	inFoundation := m.GoType("NSString *", foundCtx, nil)

	if inAppKit != "*foundation.NSString" {
		t.Errorf("AppKit context: got %q, want *foundation.NSString", inAppKit)
	}
	if inFoundation != "*NSString" {
		t.Errorf("Foundation context: got %q, want *NSString", inFoundation)
	}
}

// TestCacheDifferentiatesByGenericParams verifies that the same ObjC generic type
// resolves differently depending on whether the type param is in scope.
func TestCacheDifferentiatesByGenericParams(t *testing.T) {
	m := newTestMapper()

	// Class with generic param "ObjectType" in scope.
	withParam := newTestCtx(m, "AppKit")
	withParam.GenericParams = []string{"ObjectType"}

	// Class without any generic params.
	withoutParam := newTestCtx(m, "AppKit")

	inScope := m.GoType("NSArray<ObjectType> *", withParam, nil)
	outOfScope := m.GoType("NSArray<ObjectType> *", withoutParam, nil)

	if inScope != "*foundation.NSArray[T]" {
		t.Errorf("param in scope: got %q, want *foundation.NSArray[T]", inScope)
	}
	if outOfScope != "*foundation.NSArray[objptr.Object]" {
		t.Errorf("param not in scope: got %q, want *foundation.NSArray[objptr.Object]", outOfScope)
	}
}

// TestCacheNormalisedStringSharesEntry verifies that differently-qualified ObjC
// strings that normalise to the same form share a single cache entry.
func TestCacheNormalisedStringSharesEntry(t *testing.T) {
	m := newTestMapper()
	ctx := newTestCtx(m, "AppKit")

	m.GoType("NSString *const _Nonnull", ctx, nil)
	before := len(m.cache)
	m.GoType("NSString *", ctx, nil) // same after normalisation
	after := len(m.cache)

	if after != before {
		t.Errorf("expected cache size to stay at %d (shared entry), got %d", before, after)
	}
}

// BenchmarkGoTypeCached measures the cost of a cache hit vs a cache miss.
func BenchmarkGoTypeCached(b *testing.B) {
	m := newTestMapper()
	ctx := newTestCtx(m, "AppKit")

	// Warm: one miss per unique type.
	types := []string{
		"NSString *", "NSArray *", "NSDictionary *", "NSObject *",
		"int64_t", "uint64_t", "CGFloat", "BOOL", "void *", "NSUInteger",
	}
	for _, qt := range types {
		m.GoType(qt, ctx, nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newTestCtx(m, "AppKit")
		for _, qt := range types {
			m.GoType(qt, c, nil)
		}
	}
}

// BenchmarkGoTypeUncached measures resolution without any cache benefit.
// Run with -count=1 to compare against BenchmarkGoTypeCached.
func BenchmarkGoTypeUncached(b *testing.B) {
	types := []string{
		"NSString *", "NSArray *", "NSDictionary *", "NSObject *",
		"int64_t", "uint64_t", "CGFloat", "BOOL", "void *", "NSUInteger",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Fresh mapper each iteration — no cache.
		m := newTestMapper()
		ctx := newTestCtx(m, "AppKit")
		for _, qt := range types {
			m.GoType(qt, ctx, nil)
		}
	}
}
