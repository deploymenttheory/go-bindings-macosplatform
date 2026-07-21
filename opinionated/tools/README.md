# `opinionated/tools`

Hand-authored helper packages that provide capabilities the **generated** bindings
can't express on their own. The code generator never touches anything under here
(it only writes under `bindings/`), so these are written and maintained by hand.

They exist because some Apple capabilities don't map cleanly to "send a selector
to an Objective-C object": main-thread/queue dispatch (Grand Central Dispatch),
the Keychain's `SecItem*` dictionary API, and `os_log` emission (C macros, not an
ObjC class). Each package wraps one such capability behind a small, Go-shaped API.

Most are pure [`purego`](https://github.com/ebitengine/purego) (no CGo); `oslog`
is the exception (see below). All are `//go:build darwin`.

## Packages

### `grandcentraldispatch/`
Run Go functions on the correct GCD dispatch queue/thread for Apple's threading
model. The idiomatic layer auto-dispatches `@MainActor` calls, but callers still
need to own and service the main thread, and queue-confined frameworks (e.g.
Virtualization) need their own serial queue — that's what these provide. Pure
purego (libdispatch via `dlopen`/`dlsym`).

- **`mainthread`** — run work on the process **main queue** (AppKit's `@MainActor`
  requirement). `Do(fn)` (inline when already on the main thread), `IsMain()`,
  `PumpMainRunLoop(seconds)` (drive the main run loop yourself), and
  `DispatchMain()` (hand the thread to `dispatch_main`).
- **`serialqueue`** — run work on a dedicated **serial** dispatch queue, off the
  main thread, for frameworks that are *queue-confined* rather than main-actor
  (e.g. `VZVirtualMachine`, which must be used on the queue it was created on).
  `New(label)`, `Do(fn)` (inline + re-entrancy-safe via dispatch specifics), and
  `Handle()` (the `dispatch_queue_t` to pass to `init(configuration:queue:)`).

### `keychain/`
Go-shaped helpers over the Security framework Keychain (`SecItem*`), whose
dictionary-based API doesn't fit the idiomatic class wrappers. Covers
certificates, identities, and generic passwords with `Create`/`Read`/`Update`/
`Delete`/`List` helpers (e.g. `CreateGenericPassword`, `ReadIdentity`,
`ListCertificates`). Pure purego.

### `oslog/`
Emit messages to the macOS unified logging system, mirroring Swift's
`os.Logger(subsystem:category:)` — `NewLogger(subsystem, category)` and typed
log levels. **Uses a small CGo shim** (the only package here that does): `os_log`
ships as C macros that expand to `_os_log_impl` with a compiler-packed argument
buffer and the caller image's `__dso_handle`, so emission can only be done
correctly through C. The generated `bindings/frameworks/oslog` covers only the
read side (`OSLogStore`, `OSLogEntry`).
