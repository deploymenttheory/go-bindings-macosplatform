# Keychain example

A runnable proof that the macOS **Security** framework bindings can drive the
keychain item API as **CRUD across item classes** — using only the idiomatic
layer ([`opinionated/custom/keychain`](../../opinionated/custom/keychain)), with
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
