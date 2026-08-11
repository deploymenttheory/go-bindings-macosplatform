//go:build darwin

// Package objptr holds the minimal "carries an Objective-C / C pointer"
// interface shared by generated wrapper and protocol types. It is pure Go (no
// cgo), so purego-backed library packages can embed it without pulling the cgo
// runtime — and therefore build under CGO_ENABLED=0. The cgo runtime aliases
// its Object to this one, so the two are the same type and cgo-backed and
// purego-backed packages interoperate.
package objptr

import "unsafe"

// Object is the constraint satisfied by every generated ObjC/C wrapper type:
// it exposes the underlying pointer.
type Object interface {
	Ptr() unsafe.Pointer
}
