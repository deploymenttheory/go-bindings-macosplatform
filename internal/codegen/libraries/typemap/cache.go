package typemap

import "strings"

// typeCacheKey uniquely identifies a GoType resolution call.
// All fields that influence the resolved Go type string are included.
// Fields that are stable across the entire codegen run (ClassNameIndex, OwnerIndex,
// ModulePrefix) are held in the Mapper itself and need not be in the key.
type typeCacheKey struct {
	// qt is the already-normalised ObjC type string (after Normalise + TrimSpace).
	// Using the normalised form increases the hit rate: "NSString *const _Nonnull"
	// and "NSString *" resolve to the same cache entry.
	qt string
	// framework is the package being generated; it determines whether a class
	// reference is same-framework (unqualified) or cross-framework (qualified).
	framework string
	// gpKey is ctx.GenericParams joined by comma. It determines whether an ObjC
	// type parameter (e.g. "ObjectType") is currently in scope as Go's [T].
	gpKey string
	// className is ctx.ClassName; only matters for "instancetype" resolution.
	className string
	// isReturn distinguishes parameter position from return position: block types
	// resolve to func(...) in parameter position but unsafe.Pointer in return position.
	isReturn bool
	// isClassMethod: when true, generic type parameters resolve to objptr.Object
	// rather than T because class methods have no receiver type variable in scope.
	isClassMethod bool
}

// typeCacheVal pairs the resolved Go type string with the cross-framework imports
// that resolution discovered. Imports are replayed into the caller's ImportSet on a hit
// so callers always receive the correct import side effects.
type typeCacheVal struct {
	goType  string
	imports map[string]string // nil when no cross-framework imports were needed
}

// cacheKeyFor builds the cache key from a normalised type string and the
// per-call context fields that affect resolution.
func cacheKeyFor(n string, ctx Context) typeCacheKey {
	return typeCacheKey{
		qt:            n,
		framework:     ctx.Framework,
		gpKey:         strings.Join(ctx.GenericParams, ","),
		className:     ctx.ClassName,
		isReturn:      ctx.IsReturn,
		isClassMethod: ctx.IsClassMethod,
	}
}

// resolveWithCache checks the cache for key; on a miss it calls resolve with a
// private imports collector, stores the result, and propagates discovered imports
// into the caller-supplied imports ImportSet. Cache reads are guarded by a
// read-lock; writes by a full write-lock so concurrent GoType calls from
// parallel emitters are safe.
func (m *Mapper) resolveWithCache(key typeCacheKey, ctx Context, imports ImportSet, resolve func(Context, ImportSet) string) string {
	m.mu.RLock()
	entry, ok := m.cache[key]
	m.mu.RUnlock()
	if ok {
		if imports != nil {
			for k, v := range entry.imports {
				imports[k] = v
			}
		}
		return entry.goType
	}

	// Cache miss: resolve without holding any lock (resolution is read-only on
	// stable Mapper fields) then write the result under a full lock.
	tmpImports := make(ImportSet)

	result := resolve(ctx, tmpImports)

	if imports != nil {
		for k, v := range tmpImports {
			imports[k] = v
		}
	}

	var cachedImports map[string]string
	if len(tmpImports) > 0 {
		cachedImports = tmpImports
	}

	m.mu.Lock()
	if m.cache == nil {
		m.cache = make(map[typeCacheKey]typeCacheVal)
	}
	m.cache[key] = typeCacheVal{goType: result, imports: cachedImports}
	m.mu.Unlock()

	return result
}
