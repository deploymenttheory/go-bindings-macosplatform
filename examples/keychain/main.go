//go:build darwin

// Command keychain is a runnable proof that the macOS Security framework
// bindings can store, retrieve, update, and delete an internet-password
// keychain item end to end — using only the idiomatic security package, with no
// raw FFI, CFDictionary building, or OSStatus decoding at the call site.
//
//	go run ./examples/keychain
//
// It writes to (and cleans up after itself in) the default login keychain.
package main

import (
	"fmt"
	"os"

	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/custom/keychain"
)

const (
	server  = "keychain-example.weave.test"
	account = "weave-demo-user"
	secret  = "s3cr3t-original"
	updated = "s3cr3t-rotated"
	label   = "Weave SDK Keychain Example"
)

func main() {
	// Start from a clean slate in case a previous run was interrupted.
	if err := keychain.DeleteInternetPassword(server, label); err != nil {
		fail("pre-clean", err)
	}

	fmt.Printf("→ store   account=%q server=%q\n", account, server)
	if err := keychain.StoreInternetPassword(keychain.InternetPassword{
		Server:   server,
		Account:  account,
		Password: secret,
		Label:    label,
	}); err != nil {
		fail("store", err)
	}

	fmt.Println("→ retrieve")
	gotUser, gotPass, found, err := keychain.FindInternetPassword(server, label)
	switch {
	case err != nil:
		fail("find", err)
	case !found:
		fail("find", fmt.Errorf("item not found immediately after store"))
	case gotUser != account || gotPass != secret:
		fail("verify", fmt.Errorf("round-trip mismatch: got (%q,%q), want (%q,%q)", gotUser, gotPass, account, secret))
	}
	fmt.Printf("  ✓ account=%q password=%q\n", gotUser, gotPass)

	fmt.Println("→ update  (store again with a rotated password)")
	if err := keychain.StoreInternetPassword(keychain.InternetPassword{
		Server:   server,
		Account:  account,
		Password: updated,
		Label:    label,
	}); err != nil {
		fail("update", err)
	}
	if _, gotPass, _, err = keychain.FindInternetPassword(server, label); err != nil {
		fail("find-after-update", err)
	} else if gotPass != updated {
		fail("update-verify", fmt.Errorf("password did not rotate: got %q, want %q", gotPass, updated))
	}
	fmt.Printf("  ✓ password=%q\n", gotPass)

	fmt.Println("→ delete")
	if err := keychain.DeleteInternetPassword(server, label); err != nil {
		fail("delete", err)
	}
	if _, _, found, err = keychain.FindInternetPassword(server, label); err != nil {
		fail("find-after-delete", err)
	} else if found {
		fail("delete-verify", fmt.Errorf("item still present after delete"))
	}
	fmt.Println("  ✓ confirmed gone")

	fmt.Println("\nPASS: keychain store → retrieve → update → delete round-trip")
}

func fail(op string, err error) {
	fmt.Fprintf(os.Stderr, "FAIL (%s): %v\n", op, err)
	// Best-effort cleanup so a failed run doesn't leave the item behind.
	_ = keychain.DeleteInternetPassword(server, label)
	os.Exit(1)
}
