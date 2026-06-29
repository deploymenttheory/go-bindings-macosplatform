# Examples — and how to adopt the bindings in your own project

These are runnable, self-cleaning programs that exercise the macOS bindings the
way a real app would. Each one is documented for *what it does* in its own README;
this page is the cross-cutting guide for *how and why* to use the bindings, so you
can apply the same patterns to your own project adopting a framework or library.

Everything here is macOS-only (`//go:build darwin`).

## The examples

| Example | What it shows | Bindings it uses |
|---|---|---|
| [`keychain`](keychain) | CRUD across keychain item classes (passwords, certificates, keys, identities) | the **custom** layer over the Security framework |
| [`warden`](warden) | A declarative, policy-driven network firewall (NEFilter content filter + an XPC control daemon) | the **idiomatic** NetworkExtension/SystemExtensions/Foundation wrappers, the **idiomatic** libproc library, and the **runtime** for what has no typed wrapper (NSXPC, ObjC subclassing, dispatch) |

Run either with `go run ./examples/<name>` (warden is a multi-binary app — see its
README). An unsigned `go run` can't do everything a signed app can (see
[Prerequisites](#prerequisites-every-adopter-hits)); both examples detect that and
still exercise the code paths they can.

## Which layer should I use, and why?

The repo offers the same macOS API at four levels. Pick the highest one that can do
what you need — you only drop lower when the level above doesn't cover your case.

| Layer | Import path | Use it for |
|---|---|---|
| **Idiomatic** | `opinionated/idiomatic/framework/<name>`, `opinionated/idiomatic/libraries/<name>` | **The default.** Calling a framework class or C function with Go types, Go errors, and automatic memory management. |
| **Custom** | `opinionated/custom/<name>` | Hand-crafted, task-oriented helpers for a workflow that would otherwise be many low-level calls (e.g. `keychain` turns the whole SecItem dance into `CreateGenericPassword`). |
| **Raw bindings** | `bindings/frameworks/<name>`, `bindings/libraries/<name>` | The exact ObjC/C surface, when you need a method the idiomatic wrapper doesn't expose. Closer to the raw API, but still GC-tracked and (for `@MainActor` classes) main-thread-wrapped like the idiomatic layer. |
| **Runtime** | `bindings/runtime/purego` (ObjC frameworks), `bindings/runtime/cgo` (C libraries) | The dispatch machinery the layers above are built on. Reach for it **only** for things that have no typed binding at all: defining a custom ObjC subclass, NSXPC, dispatch queues, or sending a selector by hand. |

Decision flow:

```
Calling a framework class / C function?           → idiomatic
A multi-step task with a provided helper package?  → custom
Need an exact API the idiomatic layer lacks?       → raw bindings
Subclassing, NSXPC, dispatch, manual selectors?    → runtime
```

`warden` is the worked example of mixing levels deliberately: it calls
`OSSystemExtensionManager`/`NEFilterNewFlowVerdict` through the **idiomatic**
wrappers, but its NEFilter subclass, XPC daemon, and dispatch queue have no typed
form, so those use the **runtime** directly. That split is the rule, not a
workaround — the idiomatic layer wraps *existing* classes; it can't define new
subclasses or model XPC/dispatch.

## Finding the binding for your framework

Apple framework `Foo.framework` maps to package `foo` in every layer:
`bindings/frameworks/foo`, `opinionated/idiomatic/framework/foo`. (Apple C
libraries live under `…/libraries/<name>` instead.)

Type and method names:

- **Raw bindings** keep the ObjC name exactly: class `NSView` → type
  `appkit.NSView`; selector `setFrame:` → method `SetFrame`.
- **Idiomatic** wrappers trim a class-name prefix **only when every class in the
  framework shares one** (≥2 uppercase chars). Foundation and AppKit classes all
  start with `NS`, so `NSString` → `foundation.String` and `NSView` →
  `appkit.View`; SystemExtensions classes all start with `OS`, so
  `OSSystemExtensionManager` → `systemextensions.SystemExtensionManager`. When a
  framework's classes don't share a prefix the **full name is kept** —
  NetworkExtension mixes `NE…` and `NW…` classes, so it stays
  `networkextension.NEFilterNewFlowVerdict` (warden uses exactly that). Instance
  selector `objectAtIndex:` → method `ObjectAtIndex`; a class method like
  `+sharedManager` becomes a package-level function `SharedManager()`.

You don't have to guess the trimmed name. Two reliable lookups:

1. Each idiomatic wrapper's doc comment says exactly which class it wraps —
   `grep -rn "wrapper over the Objective-C class NSView" opinionated/idiomatic/framework/appkit`.
2. Every generated package has a `doc.go` index listing its types.

The full naming contract is [`docs/naming.md`](../docs/naming.md); the generated
package layout is described in [`CLAUDE.md`](../CLAUDE.md).

## Crossing the idiomatic↔runtime boundary

When you mix levels (as `warden` does), you need to convert between a raw object
pointer (`objc.ID`, used by the runtime) and an idiomatic wrapper. Use the public
[`obj`](../opinionated/idiomatic/obj) package — never the internal `objref`:

| You have | You want | Use |
|---|---|---|
| a raw `objc.ID` (e.g. from a runtime callback or `dlsym`) | an idiomatic argument | `obj.Wrap(id)` → `obj.Object` |
| a raw `objc.ID` | a specific wrapper type | the type's `XxxFromID(id)` constructor |
| an idiomatic wrapper | the raw `objc.ID` (e.g. to hand to NSXPC or return from an IMP) | `obj.ID(wrapper)` |

`bindings/runtime/purego`'s `ID` is a type alias of `objc.ID`, so the two are
interchangeable with no conversion. See `warden/app/activation.go` (wrapping the
dispatch main queue with `obj.Wrap`) and `warden/extension/provider.go` (extracting
a verdict's id with `obj.ID`) for live uses.

## Prerequisites every adopter hits

- **darwin only.** All of this is `//go:build darwin`; it links against system
  frameworks at runtime via `dlopen`, so it builds and runs on macOS only.
- **Code signing + entitlements.** Many APIs are gated by the OS, not the SDK. An
  unsigned `go run` binary can read the keychain but can't store keys
  (`errSecMissingEntitlement`), and can't load a system extension at all. Protected
  APIs need a signed app bundle carrying the right entitlements (and sometimes an
  Apple Developer provisioning profile) — each example's README lists what it needs.
- **The main thread.** AppKit and other UI-isolated (`@MainActor`) APIs must run on
  the main thread. Both the idiomatic layer **and** the raw `bindings/frameworks/`
  packages wrap those calls in `purego.Main` automatically. Only when you drop to the
  runtime (`bindings/runtime/purego`) and send selectors by hand do you wrap UI calls
  in `purego.Main` yourself. (You still need an AppKit run loop on the locked main
  thread for the dispatch to be serviced.)
- **Object lifetime.** Idiomatic wrappers are garbage-collected (they retain on
  creation and release via a finalizer) — just let them go out of scope. If you work
  at the runtime level you own the references: `purego.Retain`/`Release` as needed.
- **Errors.** The idiomatic and custom layers surface failures as Go `error`s
  (`OSStatus`/`NSError` are decoded for you). At the raw/runtime level you check the
  return code or the `NSError` out-parameter yourself.
