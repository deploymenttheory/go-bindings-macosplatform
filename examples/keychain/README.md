# Keychain example

A runnable proof that the macOS **Security** framework bindings can drive the
keychain item API as **CRUD across item classes** — using only the hand-written
tools layer ([`opinionated/tools/keychain`](../../opinionated/tools/keychain)), with
no raw FFI, CFDictionary building, or OSStatus decoding at the call site.

```sh
go run ./examples/keychain
```

It exercises:

| Item class | Operations |
|---|---|
| generic password | Create, Read, Update, Read, List, Delete |
| internet password | Create, Read, Delete |
| certificate | Create, Read, Delete (a self-signed cert minted in-process) |
| key | Create (EC), Read, Delete |
| identity | formed from a certificate + its matching key, then Read |

Expected output:

```
● generic password
  ✓ read secret="s3cr3t-original"
  ✓ updated secret="s3cr3t-rotated"
  ✓ listed (… items; ours present, no secrets exported)
  ✓ deleted
● internet password
  ✓ read secret="https-secret"
  ✓ deleted
● certificate
  ✓ read DER (… bytes) matches
  ✓ deleted
● keys / identities (read-only)
  ✓ … key(s), … identity(ies) enumerated

PASS: keychain CRUD across item classes
```

The verbs map onto Security as `Create→SecItemAdd`, `Read→SecItemCopyMatching`
(`kSecMatchLimitOne`), `Update→SecItemUpdate`, `Delete→SecItemDelete`,
`List→SecItemCopyMatching` (`kSecMatchLimitAll`). `List` returns metadata only
(no `kSecValueData`) so it never prompts for other applications' secrets.

Notes on the non-password classes:
- **Certificate** content is immutable (it *is* the DER), so there is no Update;
  macOS derives the item's label from the certificate's subject CN.
- **Key** is created from key material with `SecKeyCreateWithData` (the example
  encodes a P-256 private key as ANSI X9.63), then added by reference.
- **Identity** is not created directly — the keychain forms it once a certificate
  and its matching private key are both present.
- **Entitlement:** storing keys/identities targets the data-protection keychain,
  which requires a keychain-access-group entitlement. An unsigned `go run` binary
  does not have one, so `CreateKey` returns `errSecMissingEntitlement (-34018)`;
  the example reports this and falls back to read-only enumeration. The Create
  path itself is real and works in a signed, entitled app. Passwords and
  certificates use the file keychain and need no entitlement.

The example writes to your default login keychain and cleans up after itself
(including on failure). It is `//go:build darwin` only.

## Adopting this

This example uses the hand-written **tools** layer — [`opinionated/tools/keychain`](../../opinionated/tools/keychain).
The keychain item API (`SecItemAdd`/`CopyMatching`/`Update`/`Delete`) is a workflow,
not a single class: each call builds a `CFDictionary` of `kSec…` attributes and
decodes an `OSStatus`. The custom layer collapses that into `CreateGenericPassword`,
`ReadKey`, and friends, so the call site has no raw FFI, dictionary building, or
status decoding. When a framework's *task* is this involved, look for a custom
package before reaching for the raw bindings.

For the cross-cutting picture — when to use the idiomatic vs custom vs raw vs
runtime layers, how to find the binding for any framework, and the signing /
entitlement prerequisites — see the [examples adoption guide](../README.md). The
entitlement note above is the concrete case of the "protected APIs need signing"
rule from that guide.
