# Handling ObjC Variadic Methods

## The Problem

CGo cannot call variadic C or Objective-C functions. This is a fundamental constraint of the CGo bridge, not a limitation of this generator. Even simple cases fail:

```go
// This does not compile — CGo rejects variadic calls
C.printf(C.CString("Hello %s\n"), C.CString("world"))
// error: unexpected type: ...
```

ObjC has several categories of variadic methods, and each requires a different bridging strategy.

---

## Categories and Strategies

### 1. Format-String Variadics (bridged via `@"%@"` wrapper)

Methods annotated with `NS_FORMAT_FUNCTION`, e.g.:
- `NSString stringWithFormat:`
- `NSString initWithFormat:`
- `NSException raise:format:`

**Why these can be bridged:** The variadic arguments are format parameters (like `%@`, `%d`). Go callers pre-format the string using `fmt.Sprintf` before calling the binding. The bridge only needs to pass the final formatted string as a value, not as a format template.

**Strategy:** For single-keyword format selectors (where the format keyword is the last part), the bridge wraps the argument with `@"%@"`:

```objc
// Generated bridge for -[NSString stringWithFormat:]
// Instead of:
id _result = [NSString stringWithFormat:(__bridge id)format];  // UNSAFE: format treated as template

// We generate:
id _result = [NSString stringWithFormat:@"%@", (__bridge id)format];  // SAFE: format treated as value
```

This matches the C wrapper pattern from the CGo documentation: replace the variadic call with a non-variadic call that accepts a pre-formatted value.

**For multi-keyword format selectors** (e.g. `initWithFormat:options:locale:`, where additional keyword arguments appear after the format), ObjC message syntax requires variadic args at the end, making the `@"%@"` insertion invalid. These are bridged using `#pragma clang diagnostic` suppression around the call.

**Go usage:**

```go
// Pre-format on the Go side, then pass the result string
msg := fmt.Sprintf("Hello %s, you have %d messages", name, count)
s := foundation.NSStringStringWithFormat(ctx, foundation.NSStringFromString(msg))
```

### 2. Nil-Terminated Object Collections (bridged via Go-level wrappers)

Methods that accept a nil-terminated list of ObjC objects:
- `NSArray arrayWithObjects:`
- `NSArray initWithObjects:`
- `NSSet setWithObjects:`
- `NSSet initWithObjects:`
- `NSOrderedSet orderedSetWithObjects:`
- `NSOrderedSet initWithObjects:`

**Why these can be bridged:** Apple provides non-variadic equivalents (`arrayWithObjects:count:`, `setWithObjects:count:`, etc.) and mutable collection types with `addObject:`. The variadic form is syntactic sugar.

**Strategy:** Pure Go wrappers that use existing bridged methods in a loop. No C code required.

```go
// Generated in frameworks/foundation/foundation_variadic.go
func NSArrayOf[T objc.Object](ctx context.Context, objects ...T) *NSArray[T] {
    arr := NSMutableArrayArrayWithCapacity(ctx, uint64(len(objects)))
    for _, o := range objects {
        arr.AddObject(ctx, o)
    }
    _cpy := arr.Copy(ctx)
    return NewNSArrayT[T](_cpy.Ptr())
}
```

**Available wrappers:**

| Function | ObjC equivalent |
|----------|----------------|
| `NSArrayOf[T](ctx, ...T)` | `+[NSArray arrayWithObjects:]` |
| `NSMutableArrayOf[T](ctx, ...T)` | `+[NSMutableArray arrayWithObjects:]` |
| `NSSetOf[T](ctx, ...T)` | `+[NSSet setWithObjects:]` |
| `NSMutableSetOf[T](ctx, ...T)` | `+[NSMutableSet setWithObjects:]` |

### 3. Dictionary-Style Key-Value Variadics (bridged via C wrapper)

Methods that accept alternating object/key or key/value pairs terminated by `nil`:
- `NSDictionary dictionaryWithObjectsAndKeys:`
- `NSDictionary initWithObjectsAndKeys:`

**Why these need a C wrapper:** There is no direct non-variadic ObjC equivalent that accepts paired arrays in a single call. The pure Go loop approach would require `setObject:forKey:` which has a compile-time `NSCopying` constraint on the key that can't be expressed with `objc.Object`.

**Strategy:** An embedded static C function that accepts parallel arrays of objects and keys, then calls `setObject:forKey:` in a loop at the ObjC level where the `NSCopying` constraint is a runtime check:

```c
// Embedded in frameworks/foundation/foundation_variadic.go (CGo preamble)
static void* goBindings_NSDictionaryFromPairs(void** objects, void** keys,
                                              NSUInteger count, void** outException) {
    @autoreleasepool {
        @try {
            NSMutableDictionary *d = [[NSMutableDictionary alloc] initWithCapacity:count];
            for (NSUInteger i = 0; i < count; i++) {
                [d setObject:(__bridge id)objects[i] forKey:(__bridge id<NSCopying>)keys[i]];
            }
            return (__bridge void *)[d retain];
        } @catch (NSException *_ex) {
            if (outException) *outException = (__bridge void *)[_ex retain];
            return nil;
        }
    }
}
```

```go
// Generated Go wrapper
func NSDictionaryOf(ctx context.Context, objectsAndKeys ...objc.Object) *NSDictionary[objc.Object] {
    n := len(objectsAndKeys) / 2
    objs := make([]unsafe.Pointer, n)
    keys := make([]unsafe.Pointer, n)
    for i := 0; i < n; i++ {
        objs[i] = objectsAndKeys[i*2].Ptr()
        keys[i] = objectsAndKeys[i*2+1].Ptr()
    }
    var objsPtr, keysPtr *unsafe.Pointer
    if n > 0 {
        objsPtr = &objs[0]
        keysPtr = &keys[0]
    }
    // void** in CGo is *unsafe.Pointer — do NOT wrap in unsafe.Pointer(...)
    ptr := unsafe.Pointer(C.goBindings_NSDictionaryFromPairs(objsPtr, keysPtr, C.NSUInteger(n), &_exc))
    ...
}
```

**Available wrappers:**

| Function | ObjC equivalent |
|----------|----------------|
| `NSDictionaryOf(ctx, val0, key0, val1, key1, ...)` | `+[NSDictionary dictionaryWithObjectsAndKeys:]` |
| `NSMutableDictionaryOf(ctx, val0, key0, ...)` | `+[NSMutableDictionary initWithObjectsAndKeys:]` |

Pairs are interleaved as `(value, key)` — matching the ObjC argument order.

**Keys must be NSCopying-conformant** (NSString, NSNumber, etc.). This is enforced by the ObjC runtime at call time; no compile-time check is applied.

### 4. Truly Non-Bridgeable Variadics (comment stubs)

Some ObjC variadic methods have no practical bridging path:

| Method | Reason |
|--------|--------|
| `NSGradient initWithColorsAndLocations:` | Alternating `NSColor*`/`CGFloat` pairs; no non-variadic equivalent with the same semantics |
| `MPSNDArrayDescriptor descriptorWithDataType:dimensionSizes:` | Variadic `NSUInteger` dimension values |
| `MPSStateResourceList resourceListWithBufferSizes:` | Same |
| `NSCoder encodeValuesOfObjCTypes:` / `decodeValuesOfObjCTypes:` | ObjC type encoding strings with typed var-args |
| `CIFilter apply:` | Complex kernel application variadics |
| `SBObject sendEvent:id:parameters:` | Apple Events descriptors |

The generator emits a comment stub for these so callers know the method exists but cannot be called directly:

```go
// Variadic class method +[NSGradient initWithColorsAndLocations:] — not bridged (CGo cannot express C variadic arguments).
```

**Workarounds:** For most of these, Apple provides alternative non-variadic APIs:
- `NSGradient` → use `initWithColors:atLocations:colorSpace:` with explicit `CGFloat[]` and `NSArray*`
- `CIFilter` → use `filterWithName:withInputParameters:` with an `NSDictionary`
- `MPSNDArrayDescriptor` → build the descriptor incrementally via setter methods

### 5. ObjC Runtime Lifecycle Methods (filtered, not bridged)

`+initialize` and `+load` are called automatically by the ObjC runtime when a class is first used or the dylib is loaded. Bridging them explicitly causes the runtime to call them twice, resulting in undefined behaviour.

The generator filters these from bridge and Go wrapper generation and emits documentation comments instead:

```go
// Class method +[NSObject initialize] — not bridged (ObjC runtime lifecycle; called automatically by the runtime).
// Class method +[NSObject load] — not bridged (ObjC runtime lifecycle; called automatically by the runtime).
```

---

## CGo Type Notes for Variadic Bridges

When writing C wrappers that accept arrays of ObjC pointers, the correct CGo types are:

| C type | CGo Go type | Notes |
|--------|-------------|-------|
| `void*` | `unsafe.Pointer` | Single ObjC object pointer |
| `void**` | `*unsafe.Pointer` | Array of ObjC pointers; **do not** wrap in `unsafe.Pointer(...)` |
| `NSUInteger` | `C.NSUInteger` | 64-bit on arm64 |
| `void** outException` | `&_exc` where `var _exc unsafe.Pointer` | Standard exception out-param |

The common mistake is converting `*unsafe.Pointer` to `unsafe.Pointer` before passing to a `void**` parameter — this loses the indirection level and causes a compile error.

---

## Where the Code Lives

| Location | Purpose |
|----------|---------|
| `internal/codegen/emit/raw/variadic_wrappers.go` | Generator for `foundation_variadic.go` |
| `internal/codegen/emit/raw/helpers.go` | `isFormatStringVariadic`, `shouldSkipBridgeMethod` |
| `internal/codegen/emit/raw/bridge.go` | `buildObjCCall` (format `@"%@"` injection), `buildBridgeTryBody` (SEL return fix) |
| `frameworks/foundation/foundation_variadic.go` | Generated: NSArrayOf, NSSetOf, NSDictionaryOf, etc. |

---

## Adding a New Variadic Wrapper

1. **Identify the category** — format-string, nil-terminated objects, paired key-values, or non-bridgeable.
2. **Find the non-variadic equivalent** in the metadata (check `.gometa.json` or SDK headers).
3. **Choose the bridge strategy:**
   - Pure Go loop via existing bridged methods → extend `FoundationVariadicWrappers` in `variadic_wrappers.go`
   - Needs C for type constraints → add a `static` C helper in the CGo preamble of the generated file
4. **Emit a comment stub** for the original variadic selector so it appears in the source but doesn't compile.
5. **Add a test** in `variadic_wrappers_test.go` asserting the wrapper function signature appears in the output.
