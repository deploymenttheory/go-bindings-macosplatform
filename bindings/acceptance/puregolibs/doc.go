// Package puregolibs holds the live acceptance suite for the purego-backed C
// libraries (clibraries.json "backend": "purego").
//
// The package's defining property is that it MUST build and pass with
// CGO_ENABLED=0: every import chain below it is pure Go, so a library that
// silently regressed to a cgo dependency fails at compile time, and every test
// makes real calls into the real system dylibs through the dlopen/purego
// bindings — round-trips with verifiable results, not just nil checks.
package puregolibs
