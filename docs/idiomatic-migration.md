# Idiomatic layer migration guide

This release rebuilds the emitted `opinionated/idiomatic/` surface to
hand-crafted-quality Go. The changes below ship together as one breaking
release train. The raw `bindings/` packages are unchanged except for
parameter names (positional only — no call-site changes needed).

## Type-mapping changes (breaking)

Signatures across every idiomatic framework package now use Go-native types
at the boundary:

| Objective-C | Before | After |
|---|---|---|
| `NSURL *` return | `obj.Object` (or `*URL`) | `string` (file URLs as paths, others as absolute URLs; round-trips with the string paths parameters already took) |
| `NSArray<NSURL *> *` | `[]*URL` / `obj.Object` | `[]string` |
| `NSDate *` | `*Date` / `float64` seconds | `time.Time` (zero `time.Time` ↔ nil NSDate) |
| `NSData *` | `*Data` (+ `unsafe.Pointer` byte access) | `[]byte` (copied per crossing; `NSMutableData` keeps its wrapper) |
| `NSDictionary<NSString *, V> *` | `obj.Object` | `map[string]V` (non-string-keyed and ungenericized dictionaries stay `obj.Object`) |
| `NSSet<T> *` | `obj.Object` | `[]T` (element order unspecified) |
| Cross-framework class returns | `obj.Object` | typed wrappers into the foundational packages, e.g. `Progress() *foundation.Progress` |

Migration: delete `obj.As(...)` downcasts on these returns and use the typed
value directly; replace `date.TimeIntervalSince1970()`-style conversions with
the `time.Time` you now receive; replace NSData plumbing with plain `[]byte`.

## Deterministic lifecycle (additive)

Every wrapper now has `Release()`: idempotent, safe under concurrent callers,
promoted through embedding, also on `obj.Object`. After `Release`, methods on
the wrapper are no-ops returning zero values (Objective-C nil-messaging), and
the garbage-collector finalizer becomes a no-op. Use `defer x.Release()` for
scarce resources; doing nothing keeps the previous finalizer-only behavior.

Every generated call also now pins its receiver and object arguments with
`defer runtime.KeepAlive(...)`, fixing a latent use-after-finalize race when
a wrapper became unreachable mid-call.

## Renames (breaking)

- Error-returning methods and constructors drop redundant `Error` /
  `WithError` / `AndReturnError` name suffixes
  (`NewStringWithContentsOfFileEncodingError` → curated `NewStringFromFile`;
  `ContentsOfDirectoryAtPathError` → `ContentsOfDirectoryAtPath`).
- A trailing `Options` label is dropped when the final parameter is an
  options bitmask; a trailing `With<X>` is dropped when `<X>` merely repeats
  the sole parameter's type (`CommonPrefixWithStringOptions` → `CommonPrefix`).
- Curated renames for high-traffic Foundation APIs live in
  `metadata/frameworks/foundation/idiomatic.json`
  (`MountedVolumes`, `RemoveItem`, `CopyItem`, `MoveItem`, `CreateDirectory`,
  `FileExists`, `ReplaceAll`, `Contains`, `Append`, …).
- Parameter names are now initialism-aware (`cpuCount`, `urlString`) and the
  reserved-word escapes read naturally (`str`, `data`, `length`,
  `identifier`, `err`). Positional only — no call-site impact.

## C-surface quality (breaking for hypervisor/vmnet consumers)

- Functions returning a registered status-code typedef (`hv_return_t`,
  `vmnet_return_t`) now return `error` (nil on success) instead of an `int`
  code, matchable via `errors.Is` against generated sentinels (vmnet) or
  inspectable via `errors.As` with `*errkit.Error`.
- Snake-case C enums have Go names: `Hv_exit_reason_t` → `ExitReason`,
  `HV_EXIT_REASON_VTIMER_ACTIVATED` → `ExitReasonVtimerActivated`,
  `Hv_gic_distributor_reg_t` → `GICDistributorReg`.
- C struct fields are PascalCase (`Virtual_address` → `VirtualAddress`).
  Layout and order are unchanged (ABI).

## Delegates (new)

Every delegate-shaped protocol (`*Delegate`, `*DataSource`, `*Observer`)
bridgeable to Go is now a generated interface plus an Objective-C shim built
at runtime — including the previously missing `VZVirtualMachineDelegate`:

```go
type stopWatcher struct{}

func (stopWatcher) GuestDidStopVirtualMachine(vm *virtualization.VirtualMachine) {
    log.Println("guest stopped")
}

vm.WithDelegate(stopWatcher{})
```

Required protocol methods form the interface; each `@optional` method is a
separate one-method `…Handler` interface — implement the ones you need on the
same value and the framework calls them (the shim answers
`respondsToSelector:` from your value's actual method set). The shim's
lifetime is tied to its owner via an associated object, so the weak delegate
property never dangles.

Per-framework tuning lives in `metadata/frameworks/<fw>/idiomatic.json`
(`delegates.include` / `delegates.exclude`, `rename_methods`,
`rename_functions`, `error_typedefs`).

## Package documentation

Every package's `doc.go` now carries sectioned godoc: `# Construction`,
`# Lifecycle`, `# Errors`, `# Main-thread requirements`, `# Delegates`, and
the `# Types` provider index.
