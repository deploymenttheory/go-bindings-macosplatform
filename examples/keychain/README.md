# Keychain example

A runnable proof that the macOS **Security** framework bindings can manage a
keychain item end to end — store, retrieve, update, and delete — using only the
hand-authored [`opinionated/custom/keychain`](../../opinionated/custom/keychain)
package, with no raw FFI, CFDictionary building, or OSStatus decoding at the call
site.

```sh
go run ./examples/keychain
```

Expected output:

```
→ store   account="weave-demo-user" server="keychain-example.weave.test"
→ retrieve
  ✓ account="weave-demo-user" password="s3cr3t-original"
→ update  (store again with a rotated password)
  ✓ password="s3cr3t-rotated"
→ delete
  ✓ confirmed gone

PASS: keychain store → retrieve → update → delete round-trip
```

The example writes an internet-password item to your default login keychain and
cleans up after itself (including on failure). It is `//go:build darwin` only.
