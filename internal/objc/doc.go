//go:build darwin

// Package objc provides the CGo runtime layer shared by all generated framework
// packages.
//
// It defines the [Object] interface that every generated ObjC wrapper type
// satisfies (exposing its underlying ObjC pointer via Ptr), and supplies the
// low-level operations that generated code relies on at runtime:
//
//   - Memory management: [Retain] and [Release] wrap ObjC retain/release; all
//     bridge functions return +1-retained pointers and a Go finalizer calls
//     Release so the caller never needs to manage object lifetimes manually.
//   - Main-thread dispatch: [RunOnMainThread] executes a closure on the main
//     GCD queue using dispatch_sync_f, which is required for all AppKit and
//     UIKit calls.
//   - String conversion: helpers to marshal between NSString * and Go string.
//   - ObjC exception bridging: utilities used by [tel] to convert a caught
//     NSException back into a Go panic.
//
// All files in this package are compiled with -fno-objc-arc; reference
// counting is handled explicitly through the functions above.
package objc
